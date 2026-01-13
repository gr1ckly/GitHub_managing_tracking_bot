package org.example.server.model.dto;

public record UpdateTokenRequestDto(Long chatId, String token) {
}