package org.example.client.config;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.web.client.RestTemplate;

@Configuration
public class RestTemplateConfig {

    @Bean
    public RestTemplate restTemplate() {
        // Простая конфигурация RestTemplate; при необходимости таймауты можно добавить позже.
        return new RestTemplate();
    }
}
