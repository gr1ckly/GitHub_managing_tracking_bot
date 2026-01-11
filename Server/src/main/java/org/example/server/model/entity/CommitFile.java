package org.example.server.model.entity;

import jakarta.persistence.*;
import lombok.Getter;
import lombok.Setter;
import org.hibernate.annotations.OnDelete;
import org.hibernate.annotations.OnDeleteAction;

@Getter
@Setter
@Entity
@Table(name = "commit_files")
public class CommitFile {
    @SequenceGenerator(name = "commit_files_id_gen", sequenceName = "branches_id_seq", allocationSize = 1)
    @EmbeddedId
    private CommitFileId id;

    @MapsId("commitId")
    @ManyToOne(fetch = FetchType.LAZY, optional = false)
    @OnDelete(action = OnDeleteAction.CASCADE)
    @JoinColumn(name = "commit_id", nullable = false)
    private Commit commit;

    @MapsId("fileId")
    @ManyToOne(fetch = FetchType.LAZY, optional = false)
    @OnDelete(action = OnDeleteAction.CASCADE)
    @JoinColumn(name = "file_id", nullable = false)
    private File file;

}