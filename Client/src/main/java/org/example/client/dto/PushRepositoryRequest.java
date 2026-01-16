package org.example.client.dto;

public record PushRepositoryRequest(Long chatId, String repositoryUrl) {
}
