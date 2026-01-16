package org.example.client.bot.handler;

import org.example.client.bot.*;
import org.springframework.stereotype.Component;

@Component
public class AddFileCommandHandler implements BotCommandHandler {
    private final UserStateService userStateService;

    public AddFileCommandHandler(UserStateService userStateService) {
        this.userStateService = userStateService;
    }

    @Override
    public boolean supports(String command) {
        return "/add".equalsIgnoreCase(command);
    }

    @Override
    public BotResponse handle(BotCommandContext ctx) {
        userStateService.setPending(ctx.chatId(), new PendingAction(PendingActionType.WAIT_FILE_UPLOAD, null));
        return BotResponse.text(ctx.chatId(), "Пришли путь до файла в репозитории, затем отправь сам файл.");
    }
}
