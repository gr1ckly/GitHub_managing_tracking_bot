package org.example.server.services;

import org.example.server.model.dto.*;
import org.example.server.model.entity.*;
import org.example.server.model.enums.FileState;
import org.example.server.repos.EditorSessionRepository;
import org.example.server.repos.FileRepository;
import org.example.server.repos.RepoRepository;
import org.example.server.repos.TokensRepository;
import org.example.server.repos.UserRepoRepository;
import org.example.server.repos.UserRepository;
import org.example.server.integrations.RepTrackerClient;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.transaction.support.TransactionSynchronization;
import org.springframework.transaction.support.TransactionSynchronizationManager;
import org.springframework.web.multipart.MultipartFile;

import java.io.IOException;
import java.nio.file.Paths;
import java.time.OffsetDateTime;
import java.util.List;
import java.util.Optional;
import java.util.UUID;
import java.util.stream.Collectors;

@Service
public class RepoService {
    private static final Logger log = LoggerFactory.getLogger(RepoService.class);
    private final UserRepository userRepository;
    private final RepoRepository repoRepository;
    private final UserRepoRepository userRepoRepository;
    private final FileRepository fileRepository;
    private final EditorSessionRepository editorSessionRepository;
    private final StorageService storageService;
    private final TokensRepository tokensRepository;
    private final GitHubClientImpl gitHubClient;
    private final RepTrackerClient repTrackerClient;

    public RepoService(UserRepository userRepository,
                       RepoRepository repoRepository,
                       UserRepoRepository userRepoRepository,
                       FileRepository fileRepository,
                       EditorSessionRepository editorSessionRepository,
                       StorageService storageService,
                       TokensRepository tokensRepository,
                       GitHubClientImpl gitHubClient,
                       RepTrackerClient repTrackerClient) {
        this.userRepository = userRepository;
        this.repoRepository = repoRepository;
        this.userRepoRepository = userRepoRepository;
        this.fileRepository = fileRepository;
        this.editorSessionRepository = editorSessionRepository;
        this.storageService = storageService;
        this.tokensRepository = tokensRepository;
        this.gitHubClient = gitHubClient;
        this.repTrackerClient = repTrackerClient;
    }

    @Transactional
    public String addRepository(AddRepositoryRequest request) {
        try {
            User user = requireUser(request.chatId());
            Repo repo = repoRepository.findByUrl(request.repositoryUrl())
                    .orElseGet(() -> {
                        Repo r = new Repo();
                        r.setUrl(request.repositoryUrl());
                        String[] parts = parseOwnerAndName(request.repositoryUrl());
                        r.setOwner(parts[0]);
                        r.setName(parts[1]);
                        r.setAddedAt(OffsetDateTime.now());
                        return repoRepository.save(r);
                    });
            boolean linked = userRepoRepository.findByUserAndRepo_Id(user, repo.getId()).isPresent();
            if (!linked) {
                UserRepo link = new UserRepo();
                UserRepoId linkId = new UserRepoId();
                linkId.setUserId(user.getId());
                linkId.setRepoId(repo.getId());
                link.setId(linkId);
                link.setUser(user);
                link.setRepo(repo);
                link.setAddedAt(OffsetDateTime.now());
                userRepoRepository.save(link);
            }

            // Синхронизируем дерево репозитория в БД
            syncRepoTreeFromGitHub(user, repo);
            // Уведомляем внешний сервис о подписке
            repTrackerClient.addTrackingRepo(repo.getUrl(), user.getChatId());
        }catch(Exception e){
            log.warn(e.getLocalizedMessage());
        }
        return "Репозиторий добавлен";
    }

    @Transactional
    public String uploadFile(Long chatId, String path, MultipartFile multipartFile) {
        User user = requireUser(chatId);
        Repo repo = requireUserRepo(user);
        String cleanPath = Paths.get(path).normalize().toString();
        String objectKey = repo.getId() + "/" + cleanPath;
        byte[] bytes;
        try {
            bytes = multipartFile.getBytes();
        } catch (IOException e) {
            throw new IllegalStateException("Не удалось прочитать файл", e);
        }
        // Сначала кладем в хранилище; если БД падает, удалим объект. Если MinIO недоступен — кидаем исключение, транзакция откатится.
        try {
            storageService.putObject(objectKey, bytes, multipartFile.getContentType());
        } catch (RuntimeException ex) {
            log.warn("Не удалось сохранить объект {} в хранилище: {}", objectKey, ex.getMessage());
            throw new IllegalStateException("Не удалось сохранить файл в хранилище", ex);
        }
        try {
            File fileEntity = fileRepository.findByRepoAndPath(repo, cleanPath).orElseGet(File::new);
            fileEntity.setRepo(repo);
            fileEntity.setPath(cleanPath);
            fileEntity.setStorageKey(objectKey);
            fileEntity.setState(FileState.ADDED);
            fileEntity.setCreatedAt(Optional.ofNullable(fileEntity.getCreatedAt()).orElse(OffsetDateTime.now()));
            fileEntity.setUpdatedAt(OffsetDateTime.now());
            fileRepository.save(fileEntity);
        } catch (RuntimeException ex) {
            // Откат в хранилище, если БД не сохранилась
            try {
                storageService.deleteObject(objectKey);
            } catch (Exception cleanup) {
                log.warn("Не удалось удалить объект {} после ошибки БД: {}", objectKey, cleanup.getMessage());
            }
            throw ex;
        }
        return "Файл сохранен";
    }

    @Transactional
    public EditLinkResponse requestEditLink(RequestEditLink request) {
        User user = requireUser(request.chatId());
        Repo repo = requireUserRepo(user);
        File file = fileRepository.findByRepoAndPath(repo, request.path())
                .orElseThrow(() -> new IllegalArgumentException("Файл не найден"));
        String link = "https://editor.local/session/" + UUID.randomUUID();
        var session = new org.example.server.model.entity.EditorSession();
        session.setFile(file);
        session.setSessionUrl(link);
        session.setForUser(user);
        session.setCreatedAt(OffsetDateTime.now());
        session.setExpiresAt(OffsetDateTime.now().plusHours(1));
        editorSessionRepository.save(session);
        return new EditLinkResponse(link);
    }

    @Transactional
    public String deleteFile(DeleteFileRequest request) {
        User user = requireUser(request.chatId());
        Repo repo = requireUserRepo(user);
        File file = fileRepository.findByRepoAndPath(repo, request.path())
                .orElseThrow(() -> new IllegalArgumentException("Файл не найден"));
        // Помечаем как ожидает удаления, потом пытаемся удалить из S3
        file.setState(FileState.PENDING_DELETE);
        file.setUpdatedAt(OffsetDateTime.now());
        fileRepository.save(file);

        TransactionSynchronizationManager.registerSynchronization(new TransactionSynchronization() {
            @Override
            public void afterCommit() {
                try {
                    if (file.getStorageKey() != null) {
                        storageService.deleteObject(file.getStorageKey());
                    }
                    fileRepository.findById(file.getId()).ifPresent(fresh -> {
                        fresh.setState(FileState.DELETED);
                        fresh.setUpdatedAt(OffsetDateTime.now());
                        fileRepository.save(fresh);
                    });
                } catch (Exception e) {
                    log.warn("Не удалось удалить объект {}: {}", file.getStorageKey(), e.getMessage());
                }
            }
        });
        return "Файл помечен к удалению";
    }

    @Transactional
    public String pushRepository(PushRepositoryRequest request) {
        User user = requireUser(request.chatId());
        Repo repo = requireUserRepo(user);
        // Заглушка под будущее подключение к Git: здесь можно триггерить фоновые задачи push
        log.info("Запрос push от chatId {} для репо {}", user.getChatId(), repo.getUrl());
        return "Запрос на push принят";
    }

    @Transactional(readOnly = true)
    public String repoTree(Long chatId) {
        User user = requireUser(chatId);
        Repo repo = requireUserRepo(user);
        List<File> files = fileRepository.findByRepo(repo);
        if (files.isEmpty()) {
            return "Репозиторий пуст";
        }
        return files.stream()
                .map(File::getPath)
                .sorted()
                .collect(Collectors.joining("\n"));
    }

    @Transactional(readOnly = true)
    public List<TreeEntryDto> listEntries(Long chatId, String parentPath) {
        User user = requireUser(chatId);
        Repo repo = requireUserRepo(user);
        String normalized = parentPath == null ? "" : parentPath;
        if (!normalized.isEmpty() && normalized.startsWith("/")) {
            normalized = normalized.substring(1);
        }
        final String prefix = normalized.isEmpty() ? "" : normalized + "/";
        List<File> files = fileRepository.findByRepo(repo);
        return files.stream()
                .map(File::getPath)
                .filter(p -> p.startsWith(prefix))
                .map(p -> p.substring(prefix.length()))
                .filter(p -> !p.isEmpty())
                .map(p -> {
                    int slash = p.indexOf('/');
                    boolean dir = slash >= 0;
                    String name = dir ? p.substring(0, slash) : p;
                    String fullPath = dir ? prefix + name : prefix + name;
                    return new TreeEntryDto(name, fullPath, dir);
                })
                .collect(Collectors.toMap(TreeEntryDto::path, t -> t, (a, b) -> a)) // remove duplicates
                .values()
                .stream()
                .sorted((a, b) -> {
                    if (a.directory() == b.directory()) {
                        return a.name().compareTo(b.name());
                    }
                    return a.directory() ? -1 : 1;
                })
                .collect(Collectors.toList());
    }

    @Transactional(readOnly = true)
    public String getFileContent(Long chatId, String path) {
        FileDownload download = downloadFile(chatId, path);
        return new String(download.bytes());
    }

    @Transactional
    public FileDownload downloadFile(Long chatId, String path) {
        User user = requireUser(chatId);
        Repo repo = requireUserRepo(user);
        String cleanPath = path.startsWith("/") ? path.substring(1) : path;
        File file = fileRepository.findByRepoAndPath(repo, cleanPath)
                .orElseThrow(() -> new IllegalArgumentException("Файл не найден"));

        byte[] contentBytes;
        if (file.getStorageKey() != null) {
            try {
                contentBytes = storageService.getObjectBytes(file.getStorageKey());
            } catch (RuntimeException e) {
                log.warn("Не удалось прочитать из хранилища {}, попробуем скачать из GitHub", file.getStorageKey());
                contentBytes = downloadAndCacheFile(user, repo, cleanPath, file);
            }
        } else {
            contentBytes = downloadAndCacheFile(user, repo, cleanPath, file);
        }
        String fileName = cleanPath.contains("/") ? cleanPath.substring(cleanPath.lastIndexOf('/') + 1) : cleanPath;
        return new FileDownload(fileName, contentBytes);
    }

    private byte[] downloadAndCacheFile(User user, Repo repo, String path, File file) {
        String token = tokensRepository.findByUser(user)
                .map(Token::getToken)
                .orElseThrow(() -> new IllegalStateException("Нет сохраненного токена пользователя"));
        String branch = repo.getOwner() != null ? gitHubClient.resolveDefaultBranch(token, repo.getOwner(), repo.getName()) : "main";
        byte[] bytes = gitHubClient.downloadFile(token, repo.getOwner(), repo.getName(), path, branch);
        String objectKey = repo.getId() + "/" + path;
        storageService.putObject(objectKey, bytes, "application/octet-stream");
        file.setStorageKey(objectKey);
        file.setUpdatedAt(OffsetDateTime.now());
        fileRepository.save(file);
        return bytes;
    }

    private void syncRepoTreeFromGitHub(User user, Repo repo) {
        String token = tokensRepository.findByUser(user)
                .map(Token::getToken)
                .orElseThrow(() -> new IllegalStateException("Нет сохраненного токена пользователя"));
        String branch = gitHubClient.resolveDefaultBranch(token, repo.getOwner(), repo.getName());
        List<GitHubTreeItem> items = gitHubClient.fetchRepoTree(token, repo.getOwner(), repo.getName(), branch);
        items.stream()
                .filter(GitHubTreeItem::isFile)
                .forEach(item -> {
                    String path = item.path();
                    File file = fileRepository.findByRepoAndPath(repo, path).orElseGet(File::new);
                    file.setRepo(repo);
                    file.setPath(path);
                    file.setState(FileState.ADDED);
                    file.setCreatedAt(Optional.ofNullable(file.getCreatedAt()).orElse(OffsetDateTime.now()));
                    file.setUpdatedAt(OffsetDateTime.now());
                    fileRepository.save(file);
                });
    }

    @Transactional
    public void retryPendingDeletes() {
        fileRepository.findAll().stream()
                .filter(f -> f.getState() == FileState.PENDING_DELETE)
                .forEach(file -> {
                    try {
                        if (file.getStorageKey() != null) {
                            storageService.deleteObject(file.getStorageKey());
                        }
                        file.setState(FileState.DELETED);
                        file.setUpdatedAt(OffsetDateTime.now());
                        fileRepository.save(file);
                    } catch (Exception e) {
                        log.warn("Повторное удаление объекта {} не удалось: {}", file.getStorageKey(), e.getMessage());
                    }
                });
    }

    private User requireUser(Long chatId) {
        return userRepository.findByChatId(chatId)
                .orElseThrow(() -> new IllegalArgumentException("Пользователь не найден"));
    }

    private Repo requireUserRepo(User user) {
        return userRepoRepository.findFirstByUser(user)
                .map(UserRepo::getRepo)
                .orElseThrow(() -> new IllegalArgumentException("У пользователя не привязан репозиторий"));
    }

    private String[] parseOwnerAndName(String url) {
        try {
            String trimmed = url.replaceAll(".git$", "");
            String[] parts = trimmed.split("/");
            String owner = parts[parts.length - 2];
            String name = parts[parts.length - 1];
            return new String[]{owner, name};
        } catch (Exception e) {
            return new String[]{"unknown", "unknown"};
        }
    }
}
