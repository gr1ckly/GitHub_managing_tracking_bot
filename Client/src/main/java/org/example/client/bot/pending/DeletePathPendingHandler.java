package org.example.client.bot.pending;

import org.example.client.bot.*;
import org.example.client.service.ServerClientService;
import org.springframework.stereotype.Component;

@Component
public class DeletePathPendingHandler implements PendingActionHandler {
    private final ServerClientService serverClientService;

    public DeletePathPendingHandler(ServerClientService serverClientService) {
        this.serverClientService = serverClientService;
    }

    @Override
    public boolean supports(PendingActionType type) {
        return PendingActionType.WAIT_DELETE_PATH == type;
    }

    @Override
    public BotResponse handle(PendingAction pendingAction, BotCommandContext ctx) {
        String result = serverClientService.deleteFile(ctx.chatId(), ctx.payload());
        return BotResponse.text(ctx.chatId(), result);
    }
}
