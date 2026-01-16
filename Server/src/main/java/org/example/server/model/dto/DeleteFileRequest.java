package org.example.server.model.dto;

public record DeleteFileRequest(Long chatId, String path) {
}
