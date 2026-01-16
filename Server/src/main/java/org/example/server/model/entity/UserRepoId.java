package org.example.server.model.entity;

import jakarta.persistence.Column;
import jakarta.persistence.Embeddable;
import lombok.Getter;
import lombok.Setter;
import org.hibernate.Hibernate;

import java.util.Objects;

@Getter
@Setter
@Embeddable
public class UserRepoId implements java.io.Serializable {
    private static final long serialVersionUID = -4229765431059186493L;
    @Column(name = "user_id", nullable = false)
    private Integer userId;

    @Column(name = "repo_id", nullable = false)
    private Integer repoId;

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || Hibernate.getClass(this) != Hibernate.getClass(o)) return false;
        UserRepoId entity = (UserRepoId) o;
        return Objects.equals(this.repoId, entity.repoId) &&
                Objects.equals(this.userId, entity.userId);
    }

    @Override
    public int hashCode() {
        return Objects.hash(repoId, userId);
    }

}