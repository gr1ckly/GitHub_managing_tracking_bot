package org.example.client.bot;

public interface PendingActionHandler {
    boolean supports(PendingActionType type);

    BotResponse handle(PendingAction pendingAction, BotCommandContext ctx);
}
