package org.example.client.bot;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;
import org.telegram.telegrambots.longpolling.interfaces.LongPollingUpdateConsumer;
import org.telegram.telegrambots.meta.api.objects.Document;
import org.telegram.telegrambots.meta.api.objects.message.Message;
import org.telegram.telegrambots.meta.api.objects.Update;
import org.telegram.telegrambots.meta.api.objects.CallbackQuery;

import java.util.List;

@Component
public class BotUpdateHandler implements LongPollingUpdateConsumer {
    private static final Logger log = LoggerFactory.getLogger(BotUpdateHandler.class);

    private final BotCommandDispatcher dispatcher;
    private final UserStateService userStateService;
    private final TelegramSender sender;

    public BotUpdateHandler(BotCommandDispatcher dispatcher,
                            UserStateService userStateService,
                            TelegramSender sender) {
        this.dispatcher = dispatcher;
        this.userStateService = userStateService;
        this.sender = sender;
    }

    @Override
    public void consume(List<Update> updates) {
        updates.forEach(this::handleUpdateSafely);
    }

    private void handleUpdateSafely(Update update) {
        try {
            handleUpdate(update);
        } catch (Exception e) {
            log.error("Failed to handle update {}", update.getUpdateId(), e);
        }
    }

    private void handleUpdate(Update update) {
        if (update.hasCallbackQuery()) {
            handleCallback(update.getCallbackQuery());
            return;
        }
        if (update.hasMessage()) {
            Message message = update.getMessage();
            Long chatId = message.getChatId();
            String text = message.hasText() ? message.getText().trim() : "";
            Document document = message.hasDocument() ? message.getDocument() : null;

            if (text.startsWith("/")) {
                BotCommandContext ctx = new BotCommandContext(chatId, extractCommand(text), extractPayload(text), message, document);
                try {
                    send(dispatcher.dispatch(ctx));
                } catch (IllegalArgumentException e) {
                    send(BotResponse.text(chatId, "Неизвестная команда. Используй /help."));
                }
                return;
            }

            if (userStateService.hasPending(chatId)) {
                userStateService.consume(chatId).ifPresent(pending -> {
                    BotCommandContext ctx = new BotCommandContext(chatId, pending.type().name(), text, message, document);
                    try {
                        send(dispatcher.dispatchPending(pending, ctx));
                    } catch (IllegalArgumentException e) {
                        send(BotResponse.text(chatId, "Не получилось обработать ввод. Попробуй снова или /cancel."));
                    }
                });
                return;
            }

            send(BotResponse.text(chatId, "Не понял запрос. Используй /help для списка команд."));
        }
    }

    private void send(BotResponse response) {
        sender.send(response);
    }

    private void handleCallback(CallbackQuery query) {
        String data = query.getData();
        Long chatId = query.getMessage().getChatId();
        BotCommandContext ctx = new BotCommandContext(chatId, "CALLBACK", data, (Message) query.getMessage(), null);
        try {
            send(dispatcher.dispatch(ctx));
        } catch (IllegalArgumentException e) {
            send(BotResponse.text(chatId, "Не удалось обработать действие."));
        }
    }

    private String extractCommand(String text) {
        int spaceIdx = text.indexOf(' ');
        return spaceIdx > 0 ? text.substring(0, spaceIdx) : text;
    }

    private String extractPayload(String text) {
        int spaceIdx = text.indexOf(' ');
        return spaceIdx > 0 ? text.substring(spaceIdx + 1).trim() : "";
    }
}
