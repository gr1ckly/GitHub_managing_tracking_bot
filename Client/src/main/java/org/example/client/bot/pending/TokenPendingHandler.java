package org.example.client.bot.pending;

import org.example.client.bot.*;
import org.example.client.service.ServerClientService;
import org.springframework.stereotype.Component;

@Component
public class TokenPendingHandler implements PendingActionHandler {
    private final ServerClientService serverClientService;

    public TokenPendingHandler(ServerClientService serverClientService) {
        this.serverClientService = serverClientService;
    }

    @Override
    public boolean supports(PendingActionType type) {
        return PendingActionType.WAIT_PAT_TOKEN == type;
    }

    @Override
    public BotResponse handle(PendingAction pendingAction, BotCommandContext ctx) {
        serverClientService.updateToken(ctx.chatId(), ctx.payload());
        return BotResponse.text(ctx.chatId(), "Токен сохранен.");
    }
}
