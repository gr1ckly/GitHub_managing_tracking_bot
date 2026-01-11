package org.example.client.bot.pending;

import org.example.client.bot.*;
import org.example.client.service.ServerClientService;
import org.springframework.stereotype.Component;
import org.telegram.telegrambots.meta.api.methods.GetFile;
import org.telegram.telegrambots.meta.api.objects.Document;
import org.telegram.telegrambots.meta.api.objects.File;
import org.telegram.telegrambots.meta.exceptions.TelegramApiException;

import java.io.IOException;
import java.io.InputStream;

@Component
public class FileUploadPendingHandler implements PendingActionHandler {
    private final UserStateService userStateService;
    private final ServerClientService serverClientService;
    private final TelegramSender sender;

    public FileUploadPendingHandler(UserStateService userStateService,
                                    ServerClientService serverClientService,
                                    TelegramSender sender) {
        this.userStateService = userStateService;
        this.serverClientService = serverClientService;
        this.sender = sender;
    }

    @Override
    public boolean supports(PendingActionType type) {
        return PendingActionType.WAIT_FILE_UPLOAD == type;
    }

    @Override
    public BotResponse handle(PendingAction pendingAction, BotCommandContext ctx) {
        if (pendingAction.data() == null || pendingAction.data().isBlank()) {
            // first step: path
            userStateService.setPending(ctx.chatId(), new PendingAction(PendingActionType.WAIT_FILE_UPLOAD, ctx.payload()));
            return BotResponse.text(ctx.chatId(), "Путь принят. Теперь отправь файл документом.");
        }
        Document document = ctx.document().orElse(null);
        if (document == null) {
            userStateService.setPending(ctx.chatId(), pendingAction);
            return BotResponse.text(ctx.chatId(), "Не найден файл. Отправь документ для загрузки.");
        }
        byte[] bytes = downloadDocument(document);
        String response = serverClientService.uploadFile(ctx.chatId(), pendingAction.data(), document.getFileName(), bytes);
        return BotResponse.text(ctx.chatId(), response);
    }

    private byte[] downloadDocument(Document document) {
        try {
            File file = sender.rawClient().execute(new GetFile(document.getFileId()));
            try (InputStream stream = sender.rawClient().downloadFileAsStream(file)) {
                return stream.readAllBytes();
            }
        } catch (TelegramApiException | IOException e) {
            throw new IllegalStateException("Не удалось скачать файл", e);
        }
    }
}
