package org.example.client.bot.handler;

import org.example.client.bot.BotCommandContext;
import org.example.client.bot.BotCommandHandler;
import org.example.client.bot.BotResponse;
import org.example.client.service.ServerClientService;
import org.springframework.stereotype.Component;

@Component
public class BrowseCommandHandler implements BotCommandHandler {
    private final ServerClientService serverClientService;

    public BrowseCommandHandler(ServerClientService serverClientService) {
        this.serverClientService = serverClientService;
    }

    @Override
    public boolean supports(String command) {
        return "/ls".equalsIgnoreCase(command) || "/tree".equalsIgnoreCase(command);
    }

    @Override
    public BotResponse handle(BotCommandContext ctx) {
        String tree = serverClientService.fetchRepositoryTree(ctx.chatId());
        return BotResponse.text(ctx.chatId(), tree);
    }
}
