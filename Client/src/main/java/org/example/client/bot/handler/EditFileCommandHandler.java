package org.example.client.bot.handler;

import org.example.client.bot.*;
import org.springframework.stereotype.Component;

@Component
public class EditFileCommandHandler implements BotCommandHandler {
    private final UserStateService userStateService;

    public EditFileCommandHandler(UserStateService userStateService) {
        this.userStateService = userStateService;
    }

    @Override
    public boolean supports(String command) {
        return "/edit".equalsIgnoreCase(command);
    }

    @Override
    public BotResponse handle(BotCommandContext ctx) {
        userStateService.setPending(ctx.chatId(), new PendingAction(PendingActionType.WAIT_EDIT_PATH, null));
        return BotResponse.text(ctx.chatId(), "Пришли путь к файлу для редактирования.");
    }
}
