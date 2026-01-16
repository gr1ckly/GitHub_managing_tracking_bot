package org.example.client.service;

import org.example.client.config.ServerApiProperties;
import org.example.client.dto.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.MediaType;
import org.springframework.http.RequestEntity;
import org.springframework.http.ResponseEntity;
import org.springframework.stereotype.Service;
import org.springframework.util.LinkedMultiValueMap;
import org.springframework.util.MultiValueMap;
import org.springframework.web.client.RestTemplate;

import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.net.URI;
import java.util.Arrays;
import java.util.Collections;
import java.util.List;

@Service
public class ServerClientService {
    private static final Logger log = LoggerFactory.getLogger(ServerClientService.class);

    private final RestTemplate restTemplate;
    private final ServerApiProperties properties;

    public ServerClientService(RestTemplate restTemplate, ServerApiProperties properties) {
        this.restTemplate = restTemplate;
        this.properties = properties;
    }

    public String registerUser(Long chatId, String username) {
        URI uri = URI.create(properties.getBaseUrl() + "/api/users/register");
        RegisterUserRequest request = new RegisterUserRequest(chatId, username);
        restTemplate.postForEntity(uri, request, Void.class);
        return "Регистрация выполнена";
    }

    public String updateToken(Long chatId, String token) {
        URI uri = URI.create(properties.getBaseUrl() + "/api/users/token");
        UpdateTokenRequest request = new UpdateTokenRequest(chatId, token);
        restTemplate.postForEntity(uri, request, Void.class);
        return "PAT-токен сохранен";
    }

    public String addRepository(Long chatId, String repoUrl) {
        URI uri = URI.create(properties.getBaseUrl() + "/api/repos/register");
        AddRepositoryRequest request = new AddRepositoryRequest(chatId, repoUrl);
        restTemplate.postForEntity(uri, request, Void.class);
        return "Репозиторий добавлен";
    }

    public String pushRepository(Long chatId, String repoUrl) {
        URI uri = URI.create(properties.getBaseUrl() + "/api/repos/push");
        PushRepositoryRequest request = new PushRepositoryRequest(chatId, repoUrl);
        restTemplate.postForEntity(uri, request, Void.class);
        return "Изменения отправлены";
    }

    public String watchRepository(Long chatId, String repoUrl) {
        URI uri = URI.create(properties.getBaseUrl() + "/api/repos/watch");
        WatchRepositoryRequest request = new WatchRepositoryRequest(chatId, repoUrl);
        restTemplate.postForEntity(uri, request, Void.class);
        return "Отслеживание репозитория включено";
    }

    public String requestEditLink(Long chatId, String path) {
        URI uri = URI.create(properties.getBaseUrl() + "/api/files/edit-link");
        RequestEditLink request = new RequestEditLink(chatId, path);
        ResponseEntity<EditLinkResponse> response = restTemplate.postForEntity(uri, request, EditLinkResponse.class);
        return response.getBody() != null ? response.getBody().link() : "Ссылка недоступна";
    }

    public String deleteFile(Long chatId, String path) {
        URI uri = URI.create(properties.getBaseUrl() + "/api/files/delete");
        DeleteFileRequest request = new DeleteFileRequest(chatId, path);
        restTemplate.postForEntity(uri, request, Void.class);
        return "Файл удален";
    }

    public String uploadFile(Long chatId, String path, String fileName, byte[] bytes) {
        URI uri = URI.create(properties.getBaseUrl() + "/api/files/upload");
        MultiValueMap<String, Object> body = new LinkedMultiValueMap<>();
        body.add("chatId", chatId.toString());
        body.add("path", path);
        body.add("file", new InMemoryResource(bytes, fileName));
        RequestEntity<MultiValueMap<String, Object>> request = RequestEntity
                .post(uri)
                .contentType(MediaType.MULTIPART_FORM_DATA)
                .body(body);
        restTemplate.exchange(request, Void.class);
        return "Файл загружен";
    }

    public String fetchRepositoryTree(Long chatId) {
        URI uri = URI.create(properties.getBaseUrl() + "/api/repos/tree?chatId=" + chatId);
        ResponseEntity<String> response = restTemplate.getForEntity(uri, String.class);
        return response.getBody() == null ? "Дерево репозитория недоступно" : response.getBody();
    }

    public List<TreeEntryDto> fetchTreeEntries(Long chatId, String path) {
        String url = properties.getBaseUrl() + "/api/repos/tree?chatId=" + chatId;
        if (path != null && !path.isBlank()) {
            url += "&path=" + path;
        }
        ResponseEntity<TreeEntryDto[]> resp = restTemplate.getForEntity(URI.create(url), TreeEntryDto[].class);
        TreeEntryDto[] body = resp.getBody();
        return body == null ? Collections.emptyList() : Arrays.asList(body);
    }

    public String fetchFileContent(Long chatId, String path) {
        String url = properties.getBaseUrl() + "/api/files/content?chatId=" + chatId + "&path=" + path;
        ResponseEntity<String> resp = restTemplate.getForEntity(URI.create(url), String.class);
        return resp.getBody() == null ? "Файл пуст или недоступен" : resp.getBody();
    }

    public byte[] downloadFile(Long chatId, String path) {
        String url = properties.getBaseUrl() + "/api/files/download?chatId=" + chatId + "&path=" + path;
        ResponseEntity<byte[]> resp = restTemplate.getForEntity(URI.create(url), byte[].class);
        byte[] body = resp.getBody();
        return body == null ? new byte[0] : body;
    }

    /**
     * Multipart resource backed by byte array to avoid temp files.
     */
    private static class InMemoryResource extends org.springframework.core.io.AbstractResource {
        private final byte[] bytes;
        private final String filename;

        private InMemoryResource(byte[] bytes, String filename) {
            this.bytes = bytes;
            this.filename = filename;
        }

        @Override
        public String getDescription() {
            return filename;
        }

        @Override
        public String getFilename() {
            return filename;
        }

        @Override
        public ByteArrayInputStream getInputStream() throws IOException {
            return new ByteArrayInputStream(bytes);
        }

        @Override
        public long contentLength() throws IOException {
            return bytes.length;
        }
    }
}
