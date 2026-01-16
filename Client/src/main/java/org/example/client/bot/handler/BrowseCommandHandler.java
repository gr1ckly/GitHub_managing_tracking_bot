package org.example.client.bot.handler;

import org.example.client.bot.BotCommandContext;
import org.example.client.bot.BotCommandHandler;
import org.example.client.bot.BotResponse;
import org.example.client.dto.TreeEntryDto;
import org.example.client.service.ServerClientService;
import org.springframework.stereotype.Component;
import org.telegram.telegrambots.meta.api.objects.replykeyboard.InlineKeyboardMarkup;
import org.telegram.telegrambots.meta.api.objects.replykeyboard.buttons.InlineKeyboardButton;
import org.telegram.telegrambots.meta.api.objects.replykeyboard.buttons.InlineKeyboardRow;

import java.util.ArrayList;
import java.util.List;

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
        List<TreeEntryDto> entries = serverClientService.fetchTreeEntries(ctx.chatId(), "");
        InlineKeyboardMarkup kb = buildKeyboard(entries, "");
        return BotResponse.textWithKeyboard(ctx.chatId(), "Выбери папку или файл:", kb);
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
            String parent = parentPath(currentPath);
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
