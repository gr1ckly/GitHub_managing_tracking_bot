package org.example.server.repos;

import org.example.server.model.entity.Notification;
import org.example.server.model.entity.Repo;
import org.example.server.model.entity.User;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.Optional;

public interface NotificationRepository extends JpaRepository<Notification, Integer> {
    Optional<Notification> findByUserAndRepo(User user, Repo repo);
}
