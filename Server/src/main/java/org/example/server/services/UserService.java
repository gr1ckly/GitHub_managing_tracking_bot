package org.example.server.services;

import org.example.server.model.entity.User;
import org.example.server.repos.UserRepository;
import org.springframework.stereotype.Service;

import java.time.OffsetDateTime;

@Service
public class UserService {
    private final UserRepository userRepository;

    public UserService(UserRepository userRepository) {
        this.userRepository = userRepository;
    }

    public String register(Long id, String name){
        User user = new User();
        user.setUsername(name);
        user.setChatId(id);
        user.setCreatedAt(OffsetDateTime.now());
        userRepository.save(user);
        return "true";
    }

}
