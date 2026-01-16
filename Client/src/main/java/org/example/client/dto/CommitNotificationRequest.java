package org.example.client.dto;

public record CommitNotificationRequest(Long chatId, String message) {
}
