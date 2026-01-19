package org.example.server.services;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.example.server.model.dto.KafkaNotificationMessage;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.ResponseEntity;
import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.stereotype.Service;
import org.springframework.web.client.RestTemplate;

@Service
public class KafkaNotificationConsumer {
    private static final Logger log = LoggerFactory.getLogger(KafkaNotificationConsumer.class);

    private final ObjectMapper objectMapper = new ObjectMapper();
    private final RestTemplate restTemplate = new RestTemplate();
    private final String botNotificationUrl;

    public KafkaNotificationConsumer(@Value("${bot.notification.url}") String botNotificationUrl) {
        this.botNotificationUrl = botNotificationUrl;
    }

    @KafkaListener(topics = "${reptracker.kafka.topic}", groupId = "rep-tracker-consumer")
    public void consume(String payload) {
        try {
            KafkaNotificationMessage msg = objectMapper.readValue(payload, KafkaNotificationMessage.class);
            log.info("Получено уведомление от Kafka для chatId {}: {}", msg.chatId(), msg.message());
            ResponseEntity<Void> response = restTemplate.postForEntity(botNotificationUrl, msg, Void.class);
            if (!response.getStatusCode().is2xxSuccessful()) {
                log.warn("Не удалось отправить уведомление боту, статус: {}", response.getStatusCode());
            }
        } catch (Exception e) {
            log.error("Ошибка обработки Kafka сообщения: {}", e.getMessage());
        }
    }
}
