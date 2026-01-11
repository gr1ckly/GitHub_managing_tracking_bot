package org.example.client.bot.handler;

import org.example.client.bot.*;
import org.springframework.stereotype.Component;

@Component
public class TokenCommandHandler implements BotCommandHandler {
    private final UserStateService userStateService;

    public TokenCommandHandler(UserStateService userStateService) {
        this.userStateService = userStateService;
    }

    @Override
    public boolean supports(String command) {
        return "/token".equalsIgnoreCase(command);
    }

    @Override
    public BotResponse handle(BotCommandContext ctx) {
        userStateService.setPending(ctx.chatId(), new PendingAction(PendingActionType.WAIT_PAT_TOKEN, null));
        return BotResponse.text(ctx.chatId(), "Отправь PAT токен сообщением.");
    }
}
