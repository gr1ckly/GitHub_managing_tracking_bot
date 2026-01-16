package org.example.server.model.dto;

public record GitHubTreeItem(String path, String type, String sha) {
    public boolean isFile() {
        return "blob".equalsIgnoreCase(type);
    }

    public boolean isDir() {
        return "tree".equalsIgnoreCase(type);
    }
}
