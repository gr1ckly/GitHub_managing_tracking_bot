package org.example.server.repos;

import org.example.server.model.entity.Token;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.Optional;

public interface TokensRepository extends JpaRepository<Token, Integer> {
    Optional<Token> findByUser(org.example.server.model.entity.User user);
}
