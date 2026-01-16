package org.example.server.model.entity;

import jakarta.persistence.*;
import lombok.Getter;
import lombok.Setter;
import org.hibernate.annotations.ColumnDefault;
import org.hibernate.annotations.OnDelete;
import org.hibernate.annotations.OnDeleteAction;

import java.time.OffsetDateTime;
import java.util.LinkedHashSet;
import java.util.Set;

@Getter
@Setter
@Entity
@Table(name = "commits")
public class Commit {
    @Id
    @GeneratedValue(strategy = GenerationType.SEQUENCE, generator = "commits_id_gen")
    @SequenceGenerator(name = "commits_id_gen", sequenceName = "commits_id_seq", allocationSize = 1)
    @Column(name = "id", nullable = false)
    private Long id;

    @ManyToOne(fetch = FetchType.LAZY, optional = false)
    @OnDelete(action = OnDeleteAction.CASCADE)
    @JoinColumn(name = "repo_id", nullable = false)
    private Repo repo;

    @ManyToOne(fetch = FetchType.LAZY)
    @OnDelete(action = OnDeleteAction.SET_NULL)
    @JoinColumn(name = "branch_id")
    private Branch branch;

    @Column(name = "commit_hash", length = Integer.MAX_VALUE)
    private String commitHash;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "author_id")
    private User author;

    @Column(name = "message", length = Integer.MAX_VALUE)
    private String message;

    @Column(name = "pushing")
    private Boolean pushing;

    @ColumnDefault("now()")
    @Column(name = "created_at")
    private OffsetDateTime createdAt;

    @ManyToMany
    @JoinTable(
            name = "commit_files",
            joinColumns = @JoinColumn(name = "commit_id"),
            inverseJoinColumns = @JoinColumn(name = "file_id")
    )
    private Set<File> files = new LinkedHashSet<>();

    @OneToMany(mappedBy = "lastCommit")
    private Set<Notification> notifications = new LinkedHashSet<>();

}
