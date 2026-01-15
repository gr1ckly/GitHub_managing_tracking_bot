package org.example.server.repos;

import org.example.server.model.entity.User;
import org.example.server.model.entity.UserRepo;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.List;
import java.util.Optional;

public interface UserRepoRepository extends JpaRepository<UserRepo, Integer> {
    List<UserRepo> findByUser(User user);
    Optional<UserRepo> findFirstByUser(User user);
    Optional<UserRepo> findByUserAndRepo_Id(User user, Integer repoId);
}
