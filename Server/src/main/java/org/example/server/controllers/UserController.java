package org.example.server.controllers;

import org.example.server.model.dto.RegisterUserRequestDTO;
import org.example.server.services.UserService;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/users")
public class UserController {
    private final UserService userService;

    public UserController(UserService userService){
        this.userService = userService;
    }

    @PostMapping("/register")
    public ResponseEntity<?> register(@RequestBody RegisterUserRequestDTO request){
        Long id = request.chatId();
        String name = request.username();
        return ResponseEntity.status(HttpStatus.CREATED).body(userService.register(id, name));
    }
}
