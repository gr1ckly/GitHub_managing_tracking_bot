package org.example.server.integrations;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import org.example.server.proto.CoderManagerServiceGrpc;
import org.example.server.proto.CreateEditorSessionRequest;
import org.example.server.proto.CreateEditorSessionResponse;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.DisposableBean;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

@Component
public class CoderManagerClient implements DisposableBean {
    private static final Logger log = LoggerFactory.getLogger(CoderManagerClient.class);

    private final ManagedChannel channel;
    private final CoderManagerServiceGrpc.CoderManagerServiceBlockingStub stub;

    public CoderManagerClient(@Value("${coder.manager.host:coder-manager-server}") String host,
                              @Value("${coder.manager.port:9090}") int port) {
        this.channel = ManagedChannelBuilder.forAddress(host, port)
                .usePlaintext()
                .build();
        this.stub = CoderManagerServiceGrpc.newBlockingStub(channel);
    }

    public String createEditorSession(String s3Key, String path, Long chatId, long ttlSeconds) {
        log.info("Requesting editor session: s3Key={}, path={}, chatId={}, ttlSeconds={}", s3Key, path, chatId, ttlSeconds);
        CreateEditorSessionRequest request = CreateEditorSessionRequest.newBuilder()
                .setS3Key(s3Key)
                .setPath(path)
                .setChatId(chatId.toString())
                .setTtlSeconds(ttlSeconds)
                .build();
        CreateEditorSessionResponse response = stub.createEditorSession(request);
        return response.getOneTimeUrl();
    }

    @Override
    public void destroy() {
        if (channel != null && !channel.isShutdown()) {
            channel.shutdownNow();
        }
    }
}
