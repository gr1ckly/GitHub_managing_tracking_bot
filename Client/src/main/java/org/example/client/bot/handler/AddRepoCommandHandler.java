package org.example.client.bot.handler;

import org.example.client.bot.*;
import org.springframework.stereotype.Component;

@Component
public class AddRepoCommandHandler implements BotCommandHandler {
    private final UserStateService userStateService;

    public AddRepoCommandHandler(UserStateService userStateService) {
        this.userStateService = userStateService;
    }

    @Override
    public boolean supports(String command) {
        return "/addrepo".equalsIgnoreCase(command);
    }

    @Override
    public BotResponse handle(BotCommandContext ctx) {
        userStateService.setPending(ctx.chatId(), new PendingAction(PendingActionType.WAIT_REPO_URL, null));
        return BotResponse.text(ctx.chatId(), "Пришли ссылку на репозиторий для подключения.");
    }
}
