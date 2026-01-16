package org.example.server.model.dto;

public record FileDownload(String fileName, byte[] bytes) {
}
