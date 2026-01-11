package org.example.client.bot;

import org.telegram.telegrambots.meta.api.objects.Document;
import org.telegram.telegrambots.meta.api.objects.message.Message;

import java.util.Optional;

public class BotCommandContext {
    private final Long chatId;
    private final String command;
    private final String payload;
    private final Message message;
    private final Document document;

    public BotCommandContext(Long chatId, String command, String payload, Message message, Document document) {
        this.chatId = chatId;
        this.command = command;
        this.payload = payload;
        this.message = message;
        this.document = document;
    }

    public Long chatId() {
        return chatId;
    }

    public String command() {
        return command;
    }

    public String payload() {
        return payload;
    }

    public Message message() {
        return message;
    }

    public Optional<Document> document() {
        return Optional.ofNullable(document);
    }
}
