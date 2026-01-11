package org.example.client.bot.pending;

import org.example.client.bot.*;
import org.example.client.service.ServerClientService;
import org.springframework.stereotype.Component;

@Component
public class PushRepoPendingHandler implements PendingActionHandler {
    private final ServerClientService serverClientService;

    public PushRepoPendingHandler(ServerClientService serverClientService) {
        this.serverClientService = serverClientService;
    }

    @Override
    public boolean supports(PendingActionType type) {
        return PendingActionType.WAIT_PUSH_REPO == type;
    }

    @Override
    public BotResponse handle(PendingAction pendingAction, BotCommandContext ctx) {
        String result = serverClientService.pushRepository(ctx.chatId(), ctx.payload());
        return BotResponse.text(ctx.chatId(), result);
    }
}
