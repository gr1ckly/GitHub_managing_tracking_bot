package org.example.client.bot.handler;

import org.example.client.bot.*;
import org.springframework.stereotype.Component;

@Component
public class DeleteFileCommandHandler implements BotCommandHandler {
    private final UserStateService userStateService;

    public DeleteFileCommandHandler(UserStateService userStateService) {
        this.userStateService = userStateService;
    }

    @Override
    public boolean supports(String command) {
        return "/rm".equalsIgnoreCase(command) || "/delete".equalsIgnoreCase(command);
    }

    @Override
    public BotResponse handle(BotCommandContext ctx) {
        userStateService.setPending(ctx.chatId(), new PendingAction(PendingActionType.WAIT_DELETE_PATH, null));
        return BotResponse.text(ctx.chatId(), "Пришли путь к файлу, который нужно удалить.");
    }
}
