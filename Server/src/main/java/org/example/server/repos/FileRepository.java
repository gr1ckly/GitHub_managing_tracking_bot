package org.example.server.repos;

import org.example.server.model.entity.File;
import org.example.server.model.entity.Repo;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.List;
import java.util.Optional;

public interface FileRepository extends JpaRepository<File, Integer> {
    Optional<File> findByRepoAndPath(Repo repo, String path);
    List<File> findByRepo(Repo repo);
}
