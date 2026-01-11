package org.example.client.bot;

import org.springframework.stereotype.Service;

import java.util.Map;
import java.util.Optional;
import java.util.concurrent.ConcurrentHashMap;

@Service
public class UserStateService {
    private final Map<Long, PendingAction> pendingByChat = new ConcurrentHashMap<>();

    public void setPending(Long chatId, PendingAction pendingAction) {
        pendingByChat.put(chatId, pendingAction);
    }

    public Optional<PendingAction> consume(Long chatId) {
        PendingAction action = pendingByChat.remove(chatId);
        return Optional.ofNullable(action);
    }

    public boolean hasPending(Long chatId) {
        return pendingByChat.containsKey(chatId);
    }
}
