package org.example.client.bot.handler;

import org.example.client.bot.BotCommandContext;
import org.example.client.bot.BotCommandHandler;
import org.example.client.bot.BotResponse;
import org.springframework.stereotype.Component;

@Component
public class HelpCommandHandler implements BotCommandHandler {
    @Override
    public boolean supports(String command) {
        return "/help".equalsIgnoreCase(command);
    }

    @Override
    public BotResponse handle(BotCommandContext ctx) {
        String help = """
                Доступные команды:
                /start - регистрация
                /token - задать PAT токен
                /addrepo - добавить репозиторий
                /add - загрузить файл
                /edit - запросить ссылку для редактирования файла
                /rm - удалить файл
                /push - отправить изменения в удаленный репозиторий
                /watch - подписаться на уведомления о коммитах
                /ls - показать дерево репозитория
                /cancel - отменить текущий ввод
                """;
        return BotResponse.text(ctx.chatId(), help);
    }
}
