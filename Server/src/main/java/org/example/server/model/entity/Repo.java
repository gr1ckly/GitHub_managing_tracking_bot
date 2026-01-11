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
@Table(name = "repos")
public class Repo {
    @Id
    @GeneratedValue(strategy = GenerationType.SEQUENCE, generator = "repos_id_gen")
    @SequenceGenerator(name = "repos_id_gen", sequenceName = "repos_id_seq", allocationSize = 1)
    @Column(name = "id", nullable = false)
    private Integer id;

    @Column(name = "url", nullable = false, length = Integer.MAX_VALUE)
    private String url;

    @Column(name = "owner", length = Integer.MAX_VALUE)
    private String owner;

    @Column(name = "name", length = Integer.MAX_VALUE)
    private String name;

    @ColumnDefault("now()")
    @Column(name = "added_at")
    private OffsetDateTime addedAt;

    @OneToMany(mappedBy = "repo")
    private Set<Branch> branches = new LinkedHashSet<>();

    @OneToMany(mappedBy = "repo")
    private Set<Commit> commits = new LinkedHashSet<>();

    @OneToMany(mappedBy = "repo")
    private Set<File> files = new LinkedHashSet<>();

    @OneToMany(mappedBy = "repo")
    private Set<Notification> notifications = new LinkedHashSet<>();

    @OneToMany(mappedBy = "repo")
    private Set<UserRepo> userRepos = new LinkedHashSet<>();

}