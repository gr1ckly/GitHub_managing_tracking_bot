package org.example.client.bot;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;
import org.telegram.telegrambots.client.okhttp.OkHttpTelegramClient;
import org.telegram.telegrambots.meta.api.methods.botapimethods.PartialBotApiMethod;
import org.telegram.telegrambots.meta.api.methods.send.SendMessage;
import org.telegram.telegrambots.meta.api.methods.send.SendDocument;
import org.telegram.telegrambots.meta.api.objects.InputFile;
import org.telegram.telegrambots.meta.api.objects.message.Message;
import org.telegram.telegrambots.meta.exceptions.TelegramApiException;
import org.telegram.telegrambots.meta.generics.TelegramClient;

import java.io.ByteArrayInputStream;

/**
 * Thin wrapper over TelegramClient to hide SDK specifics.
 */
@Service
public class TelegramSender {
    private static final Logger log = LoggerFactory.getLogger(TelegramSender.class);

    private final TelegramClient telegramClient;

    public TelegramSender(org.example.client.config.BotProperties properties) {
        this.telegramClient = new OkHttpTelegramClient(properties.getToken());
    }

    public void send(BotResponse response) {
        PartialBotApiMethod<?> method = response.toSendMessage();
        if (method instanceof SendMessage sendMessage) {
            try {
                Message result = telegramClient.execute(sendMessage);
                log.debug("Sent message {} to chat {}", result.getMessageId(), sendMessage.getChatId());
            } catch (TelegramApiException e) {
                log.error("Failed to send message to chat {}: {}", sendMessage.getChatId(), e.getMessage());
            }
        } else {
            log.warn("Unsupported response type: {}", method.getClass().getSimpleName());
        }
    }

    public void sendDocument(Long chatId, String filename, byte[] bytes) {
        SendDocument doc = SendDocument.builder()
                .chatId(chatId.toString())
                .document(new InputFile(new ByteArrayInputStream(bytes), filename))
                .build();
        try {
            telegramClient.execute(doc);
        } catch (TelegramApiException e) {
            log.error("Failed to send document to chat {}: {}", chatId, e.getMessage());
        }
    }

    public TelegramClient rawClient() {
        return telegramClient;
    }
}
