package org.example.client.bot.pending;

import org.example.client.bot.*;
import org.example.client.service.ServerClientService;
import org.springframework.stereotype.Component;

@Component
public class EditPathPendingHandler implements PendingActionHandler {
    private final ServerClientService serverClientService;

    public EditPathPendingHandler(ServerClientService serverClientService) {
        this.serverClientService = serverClientService;
    }

    @Override
    public boolean supports(PendingActionType type) {
        return PendingActionType.WAIT_EDIT_PATH == type;
    }

    @Override
    public BotResponse handle(PendingAction pendingAction, BotCommandContext ctx) {
        String link = serverClientService.requestEditLink(ctx.chatId(), ctx.payload());
        return BotResponse.text(ctx.chatId(), "Ссылка для редактирования: " + link);
    }
}
