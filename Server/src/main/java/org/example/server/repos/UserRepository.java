package org.example.server.repos;

import org.example.server.model.entity.User;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.util.List;
import java.util.Optional;

@Repository
public interface UserRepository extends JpaRepository<User, Long> {
    Optional<User> findByChatId(Long chat_id);
    
    @Query("SELECT DISTINCT u FROM User u " +
           "JOIN u.userRepos ur " +
           "JOIN ur.repo r " +
           "WHERE r.url = :repoUrl")
    List<User> findUsersByRepositoryUrl(@Param("repoUrl") String repoUrl);
}
