package org.example.client.bot.handler;

import org.example.client.bot.BotCommandContext;
import org.example.client.bot.BotCommandHandler;
import org.example.client.bot.BotResponse;
import org.example.client.service.ServerClientService;
import org.springframework.stereotype.Component;

@Component
public class StartCommandHandler implements BotCommandHandler {
    private final ServerClientService serverClientService;

    public StartCommandHandler(ServerClientService serverClientService) {
        this.serverClientService = serverClientService;
    }

    @Override
    public boolean supports(String command) {
        return "/start".equalsIgnoreCase(command);
    }

    @Override
    public BotResponse handle(BotCommandContext ctx) {
        serverClientService.registerUser(ctx.chatId(), ctx.message().getFrom().getUserName());
        String text = """
                Привет! Ты зарегистрирован.
                Задай PAT токен через /token
                Посмотреть команды: /help
                """;
        return BotResponse.text(ctx.chatId(), text);
    }
}
