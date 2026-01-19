package org.example.server.services;

import org.example.server.model.entity.Token;
import org.example.server.model.entity.User;
import org.example.server.repos.TokensRepository;
import org.example.server.repos.UserRepository;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;

import java.time.OffsetDateTime;
import java.util.Optional;

@Service
public class UserService {
    private final UserRepository userRepository;
    private final GitHubClientImpl gitHubClient;
    private final TokensRepository tokensRepository;
    private static final Logger log = LoggerFactory.getLogger(UserService.class);

    public UserService(UserRepository userRepository, GitHubClientImpl gitHubClient, TokensRepository tokensRepository) {
        this.userRepository = userRepository;
        this.gitHubClient = gitHubClient;
        this.tokensRepository = tokensRepository;
    }

    public String register(Long id, String name){
        log.info("Registering user id={} name={}", id, name);
        User user = new User();
        user.setUsername(name);
        user.setChatId(id);
        user.setCreatedAt(OffsetDateTime.now());
        try {
            userRepository.save(user);
            log.info("User id={} saved", id);
            return "true";
        } catch(Exception e){
            log.error("Failed to save user id={} name={}", id, name, e);
            throw e;
        }
    }


    public boolean token(Long id, String token){
        Token entity = new Token();
        Optional<User> user = userRepository.findByChatId(id);
        if(user.isPresent()){
            entity.setUser(user.get());
            gitHubClient.validateToken(token);
            entity.setToken(token);
            entity.setCreatedAt(OffsetDateTime.now());
            tokensRepository.save(entity);
        }else{
            return false;
        }

        return true;
    }


}
