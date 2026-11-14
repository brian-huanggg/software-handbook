package io.github.brianhuanggg.authkit.token;

import java.time.Instant;
import java.util.HashMap;
import java.util.HexFormat;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;
import java.util.function.Function;

/** For tests and single-process demos; tokens vanish on restart. Not a bean. */
public final class InMemoryRefreshStore implements RefreshStore {

    private final Map<String, Refresh> tokens = new HashMap<>();
    private final Function<UUID, Optional<String>> roleOf;

    /** Every user exists and has the role {@code user}. */
    public InMemoryRefreshStore() {
        this(id -> Optional.of("user"));
    }

    /**
     * @param roleOf the current role of a user, empty when the user does not
     *               exist. There is no users table here, so the caller supplies it.
     */
    public InMemoryRefreshStore(Function<UUID, Optional<String>> roleOf) {
        this.roleOf = roleOf;
    }

    private static String key(byte[] idHash) {
        return HexFormat.of().formatHex(idHash);
    }

    // One lock for everything: rotate must see "unused" and write "used" as one step.

    @Override
    public synchronized void create(Refresh token) {
        if (tokens.putIfAbsent(key(token.idHash()), token.copy()) != null) {
            throw new DuplicateRefreshException();
        }
    }

    @Override
    public synchronized Optional<Refresh> find(byte[] idHash) {
        return Optional.ofNullable(tokens.get(key(idHash))).map(Refresh::copy);
    }

    @Override
    public synchronized Optional<String> rotate(byte[] oldHash, Instant usedAt, Refresh next) {
        Refresh old = tokens.get(key(oldHash));
        if (old == null || old.usedAt() != null) {
            return Optional.empty();
        }
        Optional<String> role = roleOf.apply(old.userId());
        if (role.isEmpty()) {
            return role;
        }
        if (tokens.containsKey(key(next.idHash()))) {
            throw new DuplicateRefreshException();
        }
        tokens.put(key(oldHash), new Refresh(old.idHash(), old.userId(), old.familyId(), usedAt,
                old.expiresAt(), old.familyExpiresAt()));
        tokens.put(key(next.idHash()), next.copy());
        return role;
    }

    @Override
    public synchronized void deleteFamily(UUID familyId) {
        tokens.values().removeIf(t -> t.familyId().equals(familyId));
    }

    @Override
    public synchronized long deleteExpired(Instant now) {
        int before = tokens.size();
        tokens.values().removeIf(t -> t.expiresAt().isBefore(now));
        return before - tokens.size();
    }
}
