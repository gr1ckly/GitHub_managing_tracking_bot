package org.example.client.bot;

public interface BotCommandHandler {
    boolean supports(String command);

    BotResponse handle(BotCommandContext ctx);
}
