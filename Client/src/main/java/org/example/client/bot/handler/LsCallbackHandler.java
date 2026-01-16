package org.example.client.bot.handler;

import org.example.client.bot.BotCommandContext;
import org.example.client.bot.BotCommandHandler;
import org.example.client.bot.BotResponse;
import org.example.client.bot.TelegramSender;
import org.example.client.dto.TreeEntryDto;
import org.example.client.service.ServerClientService;
import org.springframework.stereotype.Component;
import org.telegram.telegrambots.meta.api.objects.replykeyboard.InlineKeyboardMarkup;
import org.telegram.telegrambots.meta.api.objects.replykeyboard.buttons.InlineKeyboardButton;
import org.telegram.telegrambots.meta.api.objects.replykeyboard.buttons.InlineKeyboardRow;

import java.util.ArrayList;
import java.util.List;

@Component
public class LsCallbackHandler implements BotCommandHandler {
    private final ServerClientService serverClientService;
    private final TelegramSender sender;

    public LsCallbackHandler(ServerClientService serverClientService, TelegramSender sender) {
        this.serverClientService = serverClientService;
        this.sender = sender;
    }

    @Override
    public boolean supports(String command) {
        return "CALLBACK".equalsIgnoreCase(command);
    }

    @Override
    public BotResponse handle(BotCommandContext ctx) {
        String data = ctx.payload();
        if (data == null || !data.startsWith("ls:")) {
            return BotResponse.text(ctx.chatId(), "Не удалось обработать действие.");
        }
        String[] parts = data.split(":", 3);
        if (parts.length < 3) {
            return BotResponse.text(ctx.chatId(), "Некорректное действие.");
        }
        String type = parts[1];
        String path = parts[2];
        if ("dir".equals(type)) {
            List<TreeEntryDto> entries = serverClientService.fetchTreeEntries(ctx.chatId(), path);
            InlineKeyboardMarkup kb = buildKeyboard(entries, path);
            String title = path.isBlank() ? "Корень" : path;
            return BotResponse.textWithKeyboard(ctx.chatId(), "Путь: " + title, kb);
        } else if ("file".equals(type)) {
            byte[] bytes = serverClientService.downloadFile(ctx.chatId(), path);
            String fileName = path.contains("/") ? path.substring(path.lastIndexOf('/') + 1) : path;
            sender.sendDocument(ctx.chatId(), fileName, bytes);
            return BotResponse.text(ctx.chatId(), "Файл отправлен.");
        }
        return BotResponse.text(ctx.chatId(), "Неизвестное действие.");
    }

    private InlineKeyboardMarkup buildKeyboard(List<TreeEntryDto> entries, String currentPath) {
        List<InlineKeyboardRow> rows = new ArrayList<>();
        for (TreeEntryDto entry : entries) {
            InlineKeyboardButton btn = InlineKeyboardButton.builder()
                    .text((entry.directory() ? "📁 " : "📄 ") + entry.name())
                    .callbackData("ls:" + (entry.directory() ? "dir:" : "file:") + entry.path())
                    .build();
            rows.add(new InlineKeyboardRow(btn));
        }
        if (currentPath != null && !currentPath.isBlank()) {
            InlineKeyboardButton back = InlineKeyboardButton.builder()
                    .text("⬅️ Назад")
                    .callbackData("ls:dir:" + parentPath(currentPath))
                    .build();
            rows.add(new InlineKeyboardRow(back));
        }
        return InlineKeyboardMarkup.builder().keyboard(rows).build();
    }

    private String parentPath(String path) {
        String p = path;
        if (p.endsWith("/")) p = p.substring(0, p.length() - 1);
        int idx = p.lastIndexOf('/');
        if (idx <= 0) return "";
        return p.substring(0, idx);
    }
}
