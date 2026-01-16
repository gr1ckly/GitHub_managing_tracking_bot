package org.example.client.bot.handler;

import org.example.client.bot.BotCommandContext;
import org.example.client.bot.BotCommandHandler;
import org.example.client.bot.BotResponse;
import org.example.client.bot.UserStateService;
import org.springframework.stereotype.Component;

@Component
public class CancelCommandHandler implements BotCommandHandler {
    private final UserStateService userStateService;

    public CancelCommandHandler(UserStateService userStateService) {
        this.userStateService = userStateService;
    }

    @Override
    public boolean supports(String command) {
        return "/cancel".equalsIgnoreCase(command);
    }

    @Override
    public BotResponse handle(BotCommandContext ctx) {
        userStateService.consume(ctx.chatId());
        return BotResponse.text(ctx.chatId(), "Текущий ввод отменен.");
    }
}
