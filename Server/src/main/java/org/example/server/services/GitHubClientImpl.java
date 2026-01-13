package org.example.server.services;

import org.example.server.infra.GitHubClient;
import org.example.server.model.dto.GitHubUser;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpStatus;
import org.springframework.http.HttpStatusCode;
import org.springframework.stereotype.Component;
import org.springframework.stereotype.Service;
import org.springframework.web.reactive.function.client.WebClient;
import org.springframework.web.reactive.function.client.WebClientResponseException;

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
}
