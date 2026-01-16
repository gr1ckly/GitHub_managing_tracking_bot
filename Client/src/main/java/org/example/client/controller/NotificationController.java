package org.example.client.controller;

import org.example.client.bot.BotResponse;
import org.example.client.bot.TelegramSender;
import org.example.client.dto.CommitNotificationRequest;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/telegram")
public class NotificationController {
    private static final Logger log = LoggerFactory.getLogger(NotificationController.class);

    private final TelegramSender telegramSender;

    public NotificationController(TelegramSender telegramSender) {
        this.telegramSender = telegramSender;
    }

    @PostMapping("/notify")
    public ResponseEntity<Void> notifyUser(@RequestBody CommitNotificationRequest request) {
        log.info("Incoming commit notification for chat {}", request.chatId());
        telegramSender.send(BotResponse.text(request.chatId(), request.message()));
        return ResponseEntity.accepted().build();
    }
}
