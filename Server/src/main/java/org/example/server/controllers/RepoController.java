package org.example.server.controllers;

import org.example.server.model.dto.AddRepositoryRequest;
import org.example.server.model.dto.PushRepositoryRequest;
import org.example.server.model.dto.TreeEntryDto;
import org.example.server.services.RepoService;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;

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

    @GetMapping("/tree")
    public ResponseEntity<List<TreeEntryDto>> tree(@RequestParam("chatId") Long chatId,
                                                   @RequestParam(value = "path", required = false) String path) {
        return ResponseEntity.ok(repoService.listEntries(chatId, path));
    }
}
