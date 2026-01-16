package org.example.client.bot.handler;

import org.example.client.bot.*;
import org.springframework.stereotype.Component;

@Component
public class WatchCommandHandler implements BotCommandHandler {
    private final UserStateService userStateService;

    public WatchCommandHandler(UserStateService userStateService) {
        this.userStateService = userStateService;
    }

    @Override
    public boolean supports(String command) {
        return "/watch".equalsIgnoreCase(command);
    }

    @Override
    public BotResponse handle(BotCommandContext ctx) {
        userStateService.setPending(ctx.chatId(), new PendingAction(PendingActionType.WAIT_WATCH_REPO, null));
        return BotResponse.text(ctx.chatId(), "Пришли ссылку на репозиторий для подписки на коммиты.");
    }
}
