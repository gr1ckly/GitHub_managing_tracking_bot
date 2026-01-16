package org.example.client.bot;

import org.example.client.config.BotProperties;
import org.springframework.stereotype.Component;
import org.telegram.telegrambots.longpolling.interfaces.LongPollingUpdateConsumer;
import org.telegram.telegrambots.longpolling.starter.SpringLongPollingBot;

@Component
public class TelegramBot implements SpringLongPollingBot {

    private final BotProperties botProperties;
    private final BotUpdateHandler updateHandler;

    public TelegramBot(BotProperties botProperties, BotUpdateHandler updateHandler) {
        this.botProperties = botProperties;
        this.updateHandler = updateHandler;
    }

    @Override
    public String getBotToken() {
        return botProperties.getToken();
    }

    @Override
    public LongPollingUpdateConsumer getUpdatesConsumer() {
        return updateHandler;
    }
}
