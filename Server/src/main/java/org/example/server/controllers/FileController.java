package org.example.server.controllers;

import org.example.server.model.dto.DeleteFileRequest;
import org.example.server.model.dto.EditLinkResponse;
import org.example.server.model.dto.RequestEditLink;
import org.example.server.services.RepoService;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.multipart.MultipartFile;

@RestController
@RequestMapping("/api/files")
public class FileController {
    private final RepoService repoService;

    public FileController(RepoService repoService) {
        this.repoService = repoService;
    }

    @PostMapping("/upload")
    public ResponseEntity<String> upload(@RequestParam("chatId") Long chatId,
                                         @RequestParam("path") String path,
                                         @RequestParam("file") MultipartFile file) {
        return ResponseEntity.ok(repoService.uploadFile(chatId, path, file));
    }

    @PostMapping("/edit-link")
    public ResponseEntity<EditLinkResponse> editLink(@RequestBody RequestEditLink request) {
        return ResponseEntity.ok(repoService.requestEditLink(request));
    }

    @PostMapping("/delete")
    public ResponseEntity<String> delete(@RequestBody DeleteFileRequest request) {
        return ResponseEntity.ok(repoService.deleteFile(request));
    }
}
