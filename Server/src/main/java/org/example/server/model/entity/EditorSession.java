package org.example.server.model.entity;

import jakarta.persistence.*;
import lombok.Getter;
import lombok.Setter;
import org.hibernate.annotations.ColumnDefault;
import org.hibernate.annotations.OnDelete;
import org.hibernate.annotations.OnDeleteAction;

import java.time.OffsetDateTime;

@Getter
@Setter
@Entity
@Table(name = "editor_sessions")
public class EditorSession {
    @Id
    @GeneratedValue(strategy = GenerationType.SEQUENCE, generator = "editor_sessions_id_gen")
    @SequenceGenerator(name = "editor_sessions_id_gen", sequenceName = "editor_sessions_id_seq", allocationSize = 1)
    @Column(name = "id", nullable = false)
    private Long id;

    @ManyToOne(fetch = FetchType.LAZY, optional = false)
    @OnDelete(action = OnDeleteAction.CASCADE)
    @JoinColumn(name = "file_id", nullable = false)
    private File file;

    @Column(name = "session_url", nullable = false, length = Integer.MAX_VALUE)
    private String sessionUrl;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "for_user")
    private User forUser;

    @ColumnDefault("now()")
    @Column(name = "created_at")
    private OffsetDateTime createdAt;

    @Column(name = "expires_at")
    private OffsetDateTime expiresAt;

}