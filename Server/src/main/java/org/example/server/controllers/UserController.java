package org.example.server.controllers;

import org.example.server.model.dto.RegisterUserRequestDTO;
import org.example.server.model.dto.UpdateTokenRequestDto;
import org.example.server.services.UserService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/users")
public class UserController {
    private static final Logger log = LoggerFactory.getLogger(UserController.class);
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

    @PostMapping("/token")
    public ResponseEntity<?> token(@RequestBody UpdateTokenRequestDto request){
        Long id = request.chatId();
        String token = request.token();
        log.warn("айди: " + id + ", токен: " + token);
        return ResponseEntity.status(HttpStatus.CREATED).body(userService.token(id, token));
    }
}
