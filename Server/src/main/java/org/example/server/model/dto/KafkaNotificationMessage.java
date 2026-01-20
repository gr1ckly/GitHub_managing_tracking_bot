package org.example.server.model.dto;

import java.time.LocalDateTime;

public record KafkaNotificationMessage(
    String link,
    String author, 
    String title,
    String updated_at
) {
    
    public String getFormattedMessage() {
        return String.format("🔔 *New commit in %s*\n\n👤 *Author:* %s\n📝 *Message:* %s\n\n🔗 [View commit](%s)", 
            link, author, title, link);
    }
}
