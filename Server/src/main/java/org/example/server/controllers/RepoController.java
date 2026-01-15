package org.example.server.controllers;

import org.example.server.model.dto.AddRepositoryRequest;
import org.example.server.model.dto.PushRepositoryRequest;
import org.example.server.model.dto.WatchRepositoryRequest;
import org.example.server.services.RepoService;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/repos")
public class RepoController {

    private final RepoService repoService;

    public RepoController(RepoService repoService) {
        this.repoService = repoService;
    }

    @PostMapping("/register")
    public ResponseEntity<String> registerRepo(@RequestBody AddRepositoryRequest request) {
        return ResponseEntity.ok(repoService.addRepository(request));
    }

    @PostMapping("/push")
    public ResponseEntity<String> push(@RequestBody PushRepositoryRequest request) {
        return ResponseEntity.ok(repoService.pushRepository(request));
    }

    @PostMapping("/watch")
    public ResponseEntity<String> watch(@RequestBody WatchRepositoryRequest request) {
        return ResponseEntity.ok(repoService.watchRepository(request));
    }

    @GetMapping("/tree")
    public ResponseEntity<String> tree(@RequestParam("chatId") Long chatId) {
        return ResponseEntity.ok(repoService.repoTree(chatId));
    }
}
