package org.example.server.services;

import org.example.server.model.dto.*;
import org.example.server.model.entity.File;
import org.example.server.model.entity.Notification;
import org.example.server.model.entity.Repo;
import org.example.server.model.entity.User;
import org.example.server.model.entity.UserRepo;
import org.example.server.model.entity.UserRepoId;
import org.example.server.model.enums.FileState;
import org.example.server.repos.*;
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
    private final NotificationRepository notificationRepository;
    private final EditorSessionRepository editorSessionRepository;
    private final StorageService storageService;

    public RepoService(UserRepository userRepository,
                       RepoRepository repoRepository,
                       UserRepoRepository userRepoRepository,
                       FileRepository fileRepository,
                       NotificationRepository notificationRepository,
                       EditorSessionRepository editorSessionRepository,
                       StorageService storageService) {
        this.userRepository = userRepository;
        this.repoRepository = repoRepository;
        this.userRepoRepository = userRepoRepository;
        this.fileRepository = fileRepository;
        this.notificationRepository = notificationRepository;
        this.editorSessionRepository = editorSessionRepository;
        this.storageService = storageService;
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

    @Transactional
    public String watchRepository(WatchRepositoryRequest request) {
        User user = requireUser(request.chatId());
        Repo repo = requireUserRepo(user);
        Notification notification = notificationRepository.findByUserAndRepo(user, repo)
                .orElseGet(() -> {
                    Notification n = new Notification();
                    n.setUser(user);
                    n.setRepo(repo);
                    n.setCreatedAt(OffsetDateTime.now());
                    return n;
                });
        notification.setEnabled(true);
        notificationRepository.save(notification);
        return "Отслеживание включено";
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
