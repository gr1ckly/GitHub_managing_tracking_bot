package org.example.client.bot.pending;

import org.example.client.bot.*;
import org.example.client.service.ServerClientService;
import org.springframework.stereotype.Component;

@Component
public class WatchRepoPendingHandler implements PendingActionHandler {
    private final ServerClientService serverClientService;

    public WatchRepoPendingHandler(ServerClientService serverClientService) {
        this.serverClientService = serverClientService;
    }

    @Override
    public boolean supports(PendingActionType type) {
        return PendingActionType.WAIT_WATCH_REPO == type;
    }

    @Override
    public BotResponse handle(PendingAction pendingAction, BotCommandContext ctx) {
        String result = serverClientService.watchRepository(ctx.chatId(), ctx.payload());
        return BotResponse.text(ctx.chatId(), result);
    }
}
