package org.example.server.services;

import com.fasterxml.jackson.databind.ObjectMapper;
import org.example.server.model.dto.KafkaNotificationMessage;
import org.example.server.repos.UserRepository;
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
    private final UserRepository userRepository;

    public KafkaNotificationConsumer(@Value("${bot.notification.url}") String botNotificationUrl,
                                   UserRepository userRepository) {
        this.botNotificationUrl = botNotificationUrl;
        this.userRepository = userRepository;
        log.info("KafkaNotificationConsumer initialized with bot URL: {}", botNotificationUrl);
    }

    @KafkaListener(topics = "${reptracker.kafka.topic}", groupId = "rep-tracker-consumer")
    public void consume(String payload) {
        log.info("Received Kafka message: {}", payload);
        try {
            KafkaNotificationMessage msg = objectMapper.readValue(payload, KafkaNotificationMessage.class);
            log.info("Получено уведомление от Kafka: link={}, author={}, title={}", 
                    msg.link(), msg.author(), msg.title());
            
            // Find users who track this repository
            String repoUrl = msg.link();
            if (repoUrl != null) {
                // Extract repo owner and name from URL to find in database
                String[] parts = extractOwnerAndName(repoUrl);
                if (parts != null) {
                    String owner = parts[0];
                    String name = parts[1];
                    
                    // Find users tracking this repository
                    userRepository.findUsersByRepositoryUrl(repoUrl).forEach(user -> {
                        try {
                            String message = msg.getFormattedMessage();
                            log.info("Отправка уведомления пользователю chatId={}: {}", user.getChatId(), message);
                            
                            // Send to bot
                            BotNotificationRequest botRequest = new BotNotificationRequest(
                                user.getChatId().toString(), 
                                message
                            );
                            
                            ResponseEntity<Void> response = restTemplate.postForEntity(
                                botNotificationUrl, botRequest, Void.class);
                            
                            if (response.getStatusCode().is2xxSuccessful()) {
                                log.info("Уведомление успешно отправлено пользователю chatId={}", user.getChatId());
                            } else {
                                log.warn("Не удалось отправить уведомление боту, статус: {}", response.getStatusCode());
                            }
                        } catch (Exception e) {
                            log.info("Ошибка отправки уведомления пользователю chatId={}: {}",
                                    user.getChatId(), e.getMessage(), e);
                        }
                    });
                }
            }
        } catch (Exception e) {
            log.info("Ошибка обработки Kafka сообщения: {}", e.getMessage(), e);
        }
    }
    
    private String[] extractOwnerAndName(String githubUrl) {
        try {
            // Extract from URL like https://github.com/owner/repo
            String[] parts = githubUrl.split("/");
            if (parts.length >= 5 && parts[2].equals("github.com")) {
                return new String[]{parts[3], parts[4]};
            }
        } catch (Exception e) {
            log.warn("Ошибка парсинга URL репозитория: {}", githubUrl, e);
        }
        return null;
    }
    
    public record BotNotificationRequest(String chatId, String message) {}
}
