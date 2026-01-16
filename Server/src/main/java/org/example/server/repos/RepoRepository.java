package org.example.server.repos;

import org.example.server.model.entity.Repo;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.Optional;

public interface RepoRepository extends JpaRepository<Repo, Integer> {
    Optional<Repo> findByUrl(String url);
}
