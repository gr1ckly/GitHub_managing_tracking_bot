package org.example.client.bot;

import org.telegram.telegrambots.meta.api.methods.botapimethods.PartialBotApiMethod;
import org.telegram.telegrambots.meta.api.methods.send.SendMessage;
import org.telegram.telegrambots.meta.api.objects.replykeyboard.InlineKeyboardMarkup;

/**
 * Lightweight response wrapper to decouple handlers from Telegram SDK.
 */
public class BotResponse {
    private final Long chatId;
    private final String text;
    private final InlineKeyboardMarkup keyboard;

    private BotResponse(Long chatId, String text, InlineKeyboardMarkup keyboard) {
        this.chatId = chatId;
        this.text = text;
        this.keyboard = keyboard;
    }

    public static BotResponse text(Long chatId, String text) {
        return new BotResponse(chatId, text, null);
    }

    public static BotResponse textWithKeyboard(Long chatId, String text, InlineKeyboardMarkup keyboard) {
        return new BotResponse(chatId, text, keyboard);
    }

    public PartialBotApiMethod<?> toSendMessage() {
        SendMessage sendMessage = new SendMessage(String.valueOf(chatId), text);
        sendMessage.setReplyMarkup(keyboard);
        return sendMessage;
    }
}
