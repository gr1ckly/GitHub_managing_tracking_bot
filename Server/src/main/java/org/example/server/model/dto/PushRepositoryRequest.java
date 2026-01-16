package org.example.server.model.dto;

public record PushRepositoryRequest(Long chatId, String repositoryUrl) {
}
