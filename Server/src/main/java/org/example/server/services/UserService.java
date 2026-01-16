package org.example.server.services;

import org.example.server.model.entity.Token;
import org.example.server.model.entity.User;
import org.example.server.repos.TokensRepository;
import org.example.server.repos.UserRepository;
import org.springframework.stereotype.Service;

import java.time.OffsetDateTime;
import java.util.Optional;

@Service
public class UserService {
    private final UserRepository userRepository;
    private final GitHubClientImpl gitHubClient;
    private final TokensRepository tokensRepository;

    public UserService(UserRepository userRepository, GitHubClientImpl gitHubClient, TokensRepository tokensRepository) {
        this.userRepository = userRepository;
        this.gitHubClient = gitHubClient;
        this.tokensRepository = tokensRepository;
    }

    public String register(Long id, String name){
        User user = new User();
        user.setUsername(name);
        user.setChatId(id);
        user.setCreatedAt(OffsetDateTime.now());
        userRepository.save(user);
        return "true";
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
