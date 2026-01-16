package org.example.client.bot.pending;

import org.example.client.bot.*;
import org.example.client.service.ServerClientService;
import org.springframework.stereotype.Component;

@Component
public class RepoUrlPendingHandler implements PendingActionHandler {
    private final ServerClientService serverClientService;

    public RepoUrlPendingHandler(ServerClientService serverClientService) {
        this.serverClientService = serverClientService;
    }

    @Override
    public boolean supports(PendingActionType type) {
        return PendingActionType.WAIT_REPO_URL == type;
    }

    @Override
    public BotResponse handle(PendingAction pendingAction, BotCommandContext ctx) {
        String result = serverClientService.addRepository(ctx.chatId(), ctx.payload());
        return BotResponse.text(ctx.chatId(), result);
    }
}
