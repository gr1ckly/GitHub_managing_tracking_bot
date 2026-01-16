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
public class CommitFileId implements java.io.Serializable {
    private static final long serialVersionUID = -7631241709309494679L;
    @Column(name = "commit_id", nullable = false)
    private Long commitId;

    @Column(name = "file_id", nullable = false)
    private Integer fileId;

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || Hibernate.getClass(this) != Hibernate.getClass(o)) return false;
        CommitFileId entity = (CommitFileId) o;
        return Objects.equals(this.commitId, entity.commitId) &&
                Objects.equals(this.fileId, entity.fileId);
    }

    @Override
    public int hashCode() {
        return Objects.hash(commitId, fileId);
    }

}