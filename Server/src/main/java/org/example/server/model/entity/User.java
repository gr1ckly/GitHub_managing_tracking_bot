package org.example.server.model.entity;

import jakarta.persistence.*;
import lombok.Getter;
import lombok.Setter;
import org.hibernate.annotations.ColumnDefault;

import java.time.OffsetDateTime;
import java.util.LinkedHashSet;
import java.util.Set;

@Getter
@Setter
@Entity
@Table(name = "users")
public class User {
    @Id
    @GeneratedValue(strategy = GenerationType.SEQUENCE, generator = "users_id_gen")
    @SequenceGenerator(name = "users_id_gen", sequenceName = "users_id_seq", allocationSize = 1)
    @Column(name = "id", nullable = false)
    private Integer id;

    @Column(name = "chat_id", nullable = false, length = Integer.MAX_VALUE)
    private Long chatId;

    @Column(name = "username", length = Integer.MAX_VALUE)
    private String username;

    @ColumnDefault("now()")
    @Column(name = "created_at")
    private OffsetDateTime createdAt;

    @OneToMany(mappedBy = "author")
    private Set<Commit> commits = new LinkedHashSet<>();

    @OneToMany(mappedBy = "forUser")
    private Set<EditorSession> editorSessions = new LinkedHashSet<>();

    @OneToMany(mappedBy = "user")
    private Set<Notification> notifications = new LinkedHashSet<>();

    @OneToOne(mappedBy = "user")
    private Token token;

    @OneToMany(mappedBy = "user")
    private Set<UserRepo> userRepos = new LinkedHashSet<>();

}