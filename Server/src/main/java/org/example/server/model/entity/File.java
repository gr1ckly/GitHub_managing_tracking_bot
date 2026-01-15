package org.example.server.model.entity;

import jakarta.persistence.*;
import lombok.Getter;
import lombok.Setter;
import org.hibernate.annotations.ColumnDefault;
import org.hibernate.annotations.OnDelete;
import org.hibernate.annotations.OnDeleteAction;
import org.hibernate.annotations.JdbcTypeCode;
import org.hibernate.type.SqlTypes;
import org.example.server.model.enums.FileState;

import java.time.OffsetDateTime;
import java.util.LinkedHashSet;
import java.util.Set;

@Getter
@Setter
@Entity
@Table(name = "files")
public class File {
    @Id
    @GeneratedValue(strategy = GenerationType.SEQUENCE, generator = "files_id_gen")
    @SequenceGenerator(name = "files_id_gen", sequenceName = "files_id_seq", allocationSize = 1)
    @Column(name = "id", nullable = false)
    private Integer id;

    @ManyToOne(fetch = FetchType.LAZY, optional = false)
    @OnDelete(action = OnDeleteAction.CASCADE)
    @JoinColumn(name = "repo_id", nullable = false)
    private Repo repo;

    @Column(name = "path", nullable = false, length = Integer.MAX_VALUE)
    private String path;

    @Column(name = "storage_key", length = Integer.MAX_VALUE)
    private String storageKey;

    @ColumnDefault("now()")
    @Column(name = "created_at")
    private OffsetDateTime createdAt;

    @ColumnDefault("now()")
    @Column(name = "updated_at")
    private OffsetDateTime updatedAt;

    @ManyToMany(mappedBy = "files")
    private Set<Commit> commits = new LinkedHashSet<>();
    @OneToMany(mappedBy = "file")
    private Set<EditorSession> editorSessions = new LinkedHashSet<>();

    @Enumerated(EnumType.STRING)
    @JdbcTypeCode(SqlTypes.NAMED_ENUM)
    @Column(name = "state", columnDefinition = "file_state", nullable = false)
    private FileState state;
}
