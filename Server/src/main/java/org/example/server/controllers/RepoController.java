package org.example.server.controllers;

import org.example.server.model.dto.AddRepositoryRequest;
import org.example.server.model.dto.PushRepositoryRequest;
import org.example.server.model.dto.TreeEntryDto;
import org.example.server.services.RepoService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
@RequestMapping("/api/repos")
public class RepoController {
    private static final Logger log = LoggerFactory.getLogger(RepoController.class);

    private final RepoService repoService;

    public RepoController(RepoService repoService) {
        this.repoService = repoService;
    }

    @PostMapping("/register")
    public ResponseEntity<String> registerRepo(@RequestBody AddRepositoryRequest request) {
        log.info("Received addRepo request: chatId={}, repositoryUrl={}", request.chatId(), request.repositoryUrl());
        try {
            String result = repoService.addRepository(request);
            log.info("Successfully added repository: chatId={}, repositoryUrl={}, result={}", 
                    request.chatId(), request.repositoryUrl(), result);
            return ResponseEntity.ok(result);
        } catch (Exception e) {
            log.error("Failed to add repository: chatId={}, repositoryUrl={}, error={}", 
                    request.chatId(), request.repositoryUrl(), e.getMessage(), e);
            throw e;
        }
    }

    @PostMapping("/push")
    public ResponseEntity<String> push(@RequestBody PushRepositoryRequest request) {
        return ResponseEntity.ok(repoService.pushRepository(request));
    }

    @GetMapping("/tree")
    public ResponseEntity<List<TreeEntryDto>> tree(@RequestParam("chatId") Long chatId,
                                                   @RequestParam(value = "path", required = false) String path) {
        return ResponseEntity.ok(repoService.listEntries(chatId, path));
    }
}
