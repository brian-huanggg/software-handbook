package com.example.cookies;

import java.security.SecureRandom;
import java.time.Duration;
import java.time.Instant;
import java.util.Base64;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

import org.springframework.stereotype.Component;

@Component 
public class SessionStore {
    public static final Duration TTL = Duration.ofSeconds(3600);

    public record Session(String status, Instant createdAt, Instant expiredAt) {}

    private final SecureRandom random = new SecureRandom();
    private final Map<String, Session> sessions = new ConcurrentHashMap<>();

    public String create() {
        byte[] b = new byte[32];
        random.nextBytes(b);
        String id = Base64.getUrlEncoder().withoutPadding().encodeToString(b);

        Instant now = Instant.now();
        sessions.put(id, new Session("authenticaterd", now, now.plus(TTL)));
        
        return id;
    }
}