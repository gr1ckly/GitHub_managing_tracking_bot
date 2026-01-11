package org.example.client.bot.handler;

import org.example.client.bot.*;
import org.springframework.stereotype.Component;

@Component
public class PushCommandHandler implements BotCommandHandler {
    private final UserStateService userStateService;

    public PushCommandHandler(UserStateService userStateService) {
        this.userStateService = userStateService;
    }

    @Override
    public boolean supports(String command) {
        return "/push".equalsIgnoreCase(command);
    }

    @Override
    public BotResponse handle(BotCommandContext ctx) {
        userStateService.setPending(ctx.chatId(), new PendingAction(PendingActionType.WAIT_PUSH_REPO, null));
        return BotResponse.text(ctx.chatId(), "Укажи репозиторий, в который отправлять изменения.");
    }
}
