package org.example.server.controllers;

import org.example.server.model.dto.DeleteFileRequest;
import org.example.server.model.dto.EditLinkResponse;
import org.example.server.model.dto.FileDownload;
import org.example.server.model.dto.RequestEditLink;
import org.example.server.services.RepoService;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
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

    @GetMapping("/content")
    public ResponseEntity<String> content(@RequestParam("chatId") Long chatId,
                                          @RequestParam("path") String path) {
        return ResponseEntity.ok(repoService.getFileContent(chatId, path));
    }

    @GetMapping("/download")
    public ResponseEntity<byte[]> download(@RequestParam("chatId") Long chatId,
                                           @RequestParam("path") String path) {
        FileDownload download = repoService.downloadFile(chatId, path);
        return ResponseEntity.ok()
                .header(HttpHeaders.CONTENT_DISPOSITION, "attachment; filename=\"" + download.fileName() + "\"")
                .contentType(MediaType.APPLICATION_OCTET_STREAM)
                .body(download.bytes());
    }
}
