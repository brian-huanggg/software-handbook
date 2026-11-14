package io.github.brianhuanggg.authkit.auth;

import java.util.Collection;
import java.util.List;
import java.util.UUID;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.security.core.userdetails.UserDetails;

/** A row of {@code users} as Spring Security sees it. The username is the email. */
public record AccountUserDetails(UUID id, String email, String passwordHash, String role) implements UserDetails {

    @Override
    public Collection<? extends GrantedAuthority> getAuthorities() {
        return List.of(new SimpleGrantedAuthority("ROLE_" + role));
    }

    @Override
    public String getPassword() {
        return passwordHash;
    }

    @Override
    public String getUsername() {
        return email;
    }

    public AuthenticatedPrincipal principal() {
        return new AuthenticatedPrincipal(id, List.of(role));
    }
}
