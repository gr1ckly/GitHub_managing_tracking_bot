package org.example.server.repos;

import org.example.server.model.entity.Token;
import org.springframework.data.jpa.repository.JpaRepository;

public interface TokensRepository extends JpaRepository<Token, Integer> {
}
