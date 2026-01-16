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
        TrackingRepo request = TrackingRepo.newBuilder()
                .setLink(link)
                .setChatId(chatId.toString())
                .build();
        stub.addTrackingRepo(request);
        log.info("RepTracker notified for repo {} and chat {}", link, chatId);
    }

    @Override
    public void destroy() {
        if (channel != null && !channel.isShutdown()) {
            channel.shutdownNow();
        }
    }
}
