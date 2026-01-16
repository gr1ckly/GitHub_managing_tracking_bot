package org.example.client.bot;

import org.springframework.stereotype.Service;

import java.util.List;

@Service
public class BotCommandDispatcher {

    private final List<BotCommandHandler> handlers;
    private final List<PendingActionHandler> pendingHandlers;

    public BotCommandDispatcher(List<BotCommandHandler> handlers, List<PendingActionHandler> pendingHandlers) {
        this.handlers = handlers;
        this.pendingHandlers = pendingHandlers;
    }

    public BotResponse dispatch(BotCommandContext ctx) {
        return handlers.stream()
                .filter(h -> h.supports(ctx.command()))
                .findFirst()
                .orElseThrow(() -> new IllegalArgumentException("Unknown command: " + ctx.command()))
                .handle(ctx);
    }

    public BotResponse dispatchPending(PendingAction pendingAction, BotCommandContext ctx) {
        return pendingHandlers.stream()
                .filter(handler -> handler.supports(pendingAction.type()))
                .findFirst()
                .orElseThrow(() -> new IllegalArgumentException("No handler for pending action " + pendingAction.type()))
                .handle(pendingAction, ctx);
    }
}
