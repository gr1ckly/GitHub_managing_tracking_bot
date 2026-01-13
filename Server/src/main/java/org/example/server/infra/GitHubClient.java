package org.example.server.infra;

import org.example.server.model.dto.GitHubUser;

public interface GitHubClient {
    boolean validateToken(String token);
    GitHubUser getCurrentUser(String token);
}

