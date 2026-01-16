package org.example.server.infra;

import org.example.server.model.dto.GitHubUser;
import org.example.server.model.dto.GitHubTreeItem;

import java.util.List;

public interface GitHubClient {
    boolean validateToken(String token);
    GitHubUser getCurrentUser(String token);
    List<GitHubTreeItem> fetchRepoTree(String token, String owner, String repo, String branch);
    byte[] downloadFile(String token, String owner, String repo, String path, String branch);
    String resolveDefaultBranch(String token, String owner, String repo);
}
