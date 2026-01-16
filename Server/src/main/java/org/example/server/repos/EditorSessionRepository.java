package org.example.server.repos;

import org.example.server.model.entity.EditorSession;
import org.springframework.data.jpa.repository.JpaRepository;

public interface EditorSessionRepository extends JpaRepository<EditorSession, Long> {
}
