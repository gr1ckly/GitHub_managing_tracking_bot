package org.example.server.integrations;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import org.example.server.proto.RepTrackerServiceGrpc;
import org.example.server.proto.TrackingRepo;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.DisposableBean;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

@Component
public class RepTrackerClient implements DisposableBean {
    private static final Logger log = LoggerFactory.getLogger(RepTrackerClient.class);
    private final ManagedChannel channel;
    private final RepTrackerServiceGrpc.RepTrackerServiceBlockingStub stub;

    public RepTrackerClient(@Value("${reptracker.host:localhost}") String host,
                            @Value("${reptracker.port:50051}") int port) {
        this.channel = ManagedChannelBuilder.forAddress(host, port)
                .usePlaintext()
                .build();
        this.stub = RepTrackerServiceGrpc.newBlockingStub(channel);
    }

    public void addTrackingRepo(String link, Long chatId) {
        log.info("Starting RepTracker gRPC call: link={}, chatId={}", link, chatId);
        try {
            log.info("Creating gRPC request: link={}, chatId={}", link, chatId);
            TrackingRepo request = TrackingRepo.newBuilder()
                    .setLink(link)
                    .setChatId(chatId.toString())
                    .build();
            
            log.info("Sending gRPC request to RepTracker service");
            stub.addTrackingRepo(request);
            
            log.info("RepTracker gRPC call completed successfully: link={}, chatId={}", link, chatId);
        } catch (Exception e) {
            log.info("RepTracker gRPC call failed: link={}, chatId={}, error={}", link, chatId, e.getMessage(), e);
            throw e;
        }
    }

    @Override
    public void destroy() {
        if (channel != null && !channel.isShutdown()) {
            channel.shutdownNow();
        }
    }
}
