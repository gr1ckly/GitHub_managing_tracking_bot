package org.example.server.services;

import org.example.server.infra.GitHubClient;
import org.example.server.model.dto.GitHubUser;
import org.example.server.model.dto.GitHubTreeItem;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpStatus;
import org.springframework.http.HttpStatusCode;
import org.springframework.http.MediaType;
import org.springframework.http.HttpHeaders;
import org.springframework.stereotype.Component;
import org.springframework.stereotype.Service;
import org.springframework.web.reactive.function.client.WebClient;
import org.springframework.web.reactive.function.client.WebClientResponseException;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.Base64;

@Component
public class GitHubClientImpl implements GitHubClient {
    private static final Logger log = LoggerFactory.getLogger(GitHubClientImpl.class);
    private final WebClient webClient;

    public GitHubClientImpl(WebClient webClient) {
        this.webClient = webClient;
    }

    @Override
    public boolean validateToken(String token) {
        try {
            HttpStatusCode status = webClient.get()
                    .uri("/user")
                    .headers(h -> h.setBearerAuth(token))
                    .retrieve()
                    .toBodilessEntity()
                    .block().getStatusCode();
            log.warn(status.toString());
            return true;
        } catch (WebClientResponseException e) {
            if (e.getStatusCode().value() == 401 || e.getStatusCode().value() == 403) {
                return false; // токен невалиден/нет прав
            }
            throw e; // другие ошибки — сеть/лимиты
        }
    }

    @Override
    public GitHubUser getCurrentUser(String token) {
        return webClient.get()
                .uri("/user")
                .headers(h -> h.setBearerAuth(token))
                .retrieve()
                .bodyToMono(GitHubUser.class)
                .block();
    }

    @Override
    public String resolveDefaultBranch(String token, String owner, String repo) {
        Map<?, ?> resp = webClient.get()
                .uri("/repos/{owner}/{repo}", owner, repo)
                .headers(h -> h.setBearerAuth(token))
                .retrieve()
                .bodyToMono(Map.class)
                .block();
        return resp != null && resp.get("default_branch") != null ? resp.get("default_branch").toString() : "main";
    }

    @Override
    public List<GitHubTreeItem> fetchRepoTree(String token, String owner, String repo, String branch) {
        Map<?, ?> resp = webClient.get()
                .uri(uriBuilder -> uriBuilder.path("/repos/{owner}/{repo}/git/trees/{branch}")
                        .queryParam("recursive", "1")
                        .build(owner, repo, branch))
                .headers(h -> h.setBearerAuth(token))
                .retrieve()
                .bodyToMono(Map.class)
                .block();
        List<GitHubTreeItem> result = new ArrayList<>();
        if (resp != null && resp.get("tree") instanceof List<?> treeList) {
            for (Object o : treeList) {
                if (o instanceof Map<?, ?> node) {
                    Object pathObj = node.get("path");
                    Object typeObj = node.get("type");
                    Object shaObj = node.get("sha");
                    if (pathObj != null && typeObj != null) {
                        String path = pathObj.toString();
                        String type = typeObj.toString();
                        String sha = shaObj != null ? shaObj.toString() : "";
                        result.add(new GitHubTreeItem(path, type, sha));
                    }
                }
            }
        }
        return result;
    }

    @Override
    public byte[] downloadFile(String token, String owner, String repo, String path, String branch) {
        try {
            var response = webClient.get()
                    .uri("/repos/{owner}/{repo}/contents/{path}?ref={branch}", owner, repo, path, branch)
                    .headers(h -> {
                        h.setBearerAuth(token);
                        h.set(HttpHeaders.ACCEPT, "application/vnd.github.raw");
                    })
                    .accept(MediaType.APPLICATION_OCTET_STREAM, MediaType.APPLICATION_JSON)
                    .exchangeToMono(r -> {
                        MediaType ct = r.headers().contentType().orElse(MediaType.APPLICATION_OCTET_STREAM);
                        if (MediaType.APPLICATION_OCTET_STREAM.isCompatibleWith(ct) || ct.getSubtype().equalsIgnoreCase("octet-stream")) {
                            return r.bodyToMono(byte[].class);
                        }
                        return r.bodyToMono(Map.class).map(map -> {
                            Object contentObj = map.get("content");
                            if (contentObj != null) {
                                String base64 = contentObj.toString().replaceAll("\\s", "");
                                return Base64.getDecoder().decode(base64);
                            }
                            return new byte[0];
                        });
                    })
                    .block();
            return response != null ? response : new byte[0];
        } catch (WebClientResponseException e) {
            log.warn("Не удалось скачать файл {}: {}", path, e.getMessage());
            throw e;
        }
    }
}
