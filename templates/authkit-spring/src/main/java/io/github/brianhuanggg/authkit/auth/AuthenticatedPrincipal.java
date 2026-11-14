package io.github.brianhuanggg.authkit.auth;

import java.security.Principal;
import java.util.List;
import java.util.UUID;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.authentication.AbstractAuthenticationToken;
import org.springframework.security.core.authority.SimpleGrantedAuthority;

/**
 * Who is calling, in both modes. Roles are plain names ({@code "user"});
 * authorities add the {@code ROLE_} prefix Spring expects.
 */
public record AuthenticatedPrincipal(UUID userId, List<String> roles) implements Principal {

    public AuthenticatedPrincipal {
        roles = List.copyOf(roles);
    }

    @Override
    public String getName() {
        return userId.toString();
    }

    public AbstractAuthenticationToken toAuthentication() {
        return UsernamePasswordAuthenticationToken.authenticated(this, null,
                roles.stream().map(r -> new SimpleGrantedAuthority("ROLE_" + r)).toList());
    }
}
