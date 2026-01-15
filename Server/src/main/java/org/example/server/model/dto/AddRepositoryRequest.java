package org.example.server.model.dto;

public record AddRepositoryRequest(Long chatId, String repositoryUrl) {
}
