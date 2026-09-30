package com.gauchoracing.warden.bridge;

import com.google.gson.Gson;
import com.google.gson.JsonObject;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.WebSocket;
import java.nio.ByteBuffer;
import java.time.Duration;
import java.util.ArrayDeque;
import java.util.Deque;
import java.util.Map;
import java.util.concurrent.CompletionStage;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;
import java.util.function.Consumer;
import java.util.logging.Logger;

/**
 * The plugin's end of the Discord bridge: one WebSocket to Warden, which owns
 * everything Discord-side. The plugin only sends game events and receives
 * already-resolved names and text.
 *
 * <p>Everything that touches the socket or the buffer runs on one thread, so
 * sends never overlap (java.net.http.WebSocket allows one outstanding send)
 * and no locking is needed.
 */
public final class BridgeClient {

    private static final Gson GSON = new Gson();
    private static final Duration PING_INTERVAL = Duration.ofSeconds(30);
    private static final Duration MAX_BACKOFF = Duration.ofSeconds(60);
    private static final Duration SEND_TIMEOUT = Duration.ofSeconds(10);

    /** A short outage should not lose chat, but a long one should not replay stale chat. */
    private static final int BUFFER_LIMIT = 100;
    private static final Duration BUFFER_MAX_AGE = Duration.ofSeconds(60);

    private final URI uri;
    private final String token;
    private final HttpClient http;
    private final Logger logger;
    private final Consumer<JsonObject> onMessage;
    private final ScheduledExecutorService thread = Executors.newSingleThreadScheduledExecutor(
            Thread.ofPlatform().name("warden-bridge").daemon().factory());
    private final Deque<Pending> buffer = new ArrayDeque<>();

    private WebSocket socket;
    private Duration backoff = Duration.ofSeconds(1);
    private boolean closed;

    private record Pending(String json, long queuedAt) {}

    public BridgeClient(String baseUrl, String token, Duration connectTimeout, Logger logger,
            Consumer<JsonObject> onMessage) {
        this.uri = URI.create(baseUrl.replaceAll("/+$", "").replaceFirst("^http", "ws") + "/api/plugin/bridge");
        this.token = token;
        this.http = HttpClient.newBuilder().connectTimeout(connectTimeout).build();
        this.logger = logger;
        this.onMessage = onMessage;
    }

    public void start() {
        thread.execute(this::connect);
        thread.scheduleAtFixedRate(this::ping, PING_INTERVAL.toSeconds(), PING_INTERVAL.toSeconds(),
                TimeUnit.SECONDS);
    }

    /** Safe from any thread. Buffered while disconnected. */
    public void send(Map<String, String> event) {
        String json = GSON.toJson(event);
        thread.execute(() -> {
            buffer.addLast(new Pending(json, System.currentTimeMillis()));
            while (buffer.size() > BUFFER_LIMIT) {
                buffer.removeFirst();
            }
            flush();
        });
    }

    /** Sends whatever is still queued, then closes. Blocks for at most a few seconds. */
    public void close(Map<String, String> lastEvent) {
        send(lastEvent);
        thread.execute(() -> {
            closed = true;
            if (socket != null) {
                socket.sendClose(WebSocket.NORMAL_CLOSURE, "server stopping");
            }
        });
        thread.shutdown();
        try {
            thread.awaitTermination(3, TimeUnit.SECONDS);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        }
        thread.shutdownNow();
    }

    private void connect() {
        if (closed) {
            return;
        }
        http.newWebSocketBuilder()
                .header("Authorization", "Bearer " + token)
                .buildAsync(uri, new Listener())
                .whenComplete((ws, error) -> thread.execute(() -> {
                    if (error != null) {
                        logger.fine("Warden: bridge connect failed: " + error.getMessage());
                        scheduleReconnect();
                        return;
                    }
                    socket = ws;
                    backoff = Duration.ofSeconds(1);
                    logger.info("Warden: Discord bridge connected");
                    flush();
                }));
    }

    private void scheduleReconnect() {
        socket = null;
        if (closed || thread.isShutdown()) {
            return;
        }
        Duration delay = backoff;
        backoff = backoff.multipliedBy(2).compareTo(MAX_BACKOFF) > 0 ? MAX_BACKOFF : backoff.multipliedBy(2);
        thread.schedule(this::connect, delay.toMillis(), TimeUnit.MILLISECONDS);
    }

    private void flush() {
        long cutoff = System.currentTimeMillis() - BUFFER_MAX_AGE.toMillis();
        while (!buffer.isEmpty() && buffer.peekFirst().queuedAt() < cutoff) {
            buffer.removeFirst();
        }
        while (socket != null && !buffer.isEmpty()) {
            try {
                socket.sendText(buffer.peekFirst().json(), true)
                        .get(SEND_TIMEOUT.toMillis(), TimeUnit.MILLISECONDS);
                buffer.removeFirst();
            } catch (Exception e) {
                // Leave the event queued for the next connection.
                logger.fine("Warden: bridge send failed: " + e.getMessage());
                socket.abort();
                scheduleReconnect();
                return;
            }
        }
    }

    private void ping() {
        if (socket != null) {
            socket.sendPing(ByteBuffer.allocate(0));
        }
    }

    private final class Listener implements WebSocket.Listener {

        private final StringBuilder partial = new StringBuilder();

        @Override
        public CompletionStage<?> onText(WebSocket ws, CharSequence data, boolean last) {
            partial.append(data);
            if (last) {
                String text = partial.toString();
                partial.setLength(0);
                try {
                    onMessage.accept(GSON.fromJson(text, JsonObject.class));
                } catch (RuntimeException e) {
                    logger.warning("Warden: bad bridge message: " + e.getMessage());
                }
            }
            ws.request(1);
            return null;
        }

        @Override
        public CompletionStage<?> onClose(WebSocket ws, int statusCode, String reason) {
            thread.execute(BridgeClient.this::scheduleReconnect);
            return null;
        }

        @Override
        public void onError(WebSocket ws, Throwable error) {
            logger.fine("Warden: bridge connection error: " + error.getMessage());
            thread.execute(BridgeClient.this::scheduleReconnect);
        }
    }
}
