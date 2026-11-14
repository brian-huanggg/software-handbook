package io.github.brianhuanggg.authkit.config;

import io.github.brianhuanggg.authkit.ratelimit.RateLimitFilter;
import io.github.brianhuanggg.authkit.ratelimit.RateLimits;
import io.github.brianhuanggg.authkit.session.SessionAuthenticationFilter;
import io.github.brianhuanggg.authkit.session.SessionService;
import io.github.brianhuanggg.authkit.token.JwtService;
import io.github.brianhuanggg.authkit.web.CsrfHeaderFilter;
import io.github.brianhuanggg.authkit.web.JsonAccessDeniedHandler;
import io.github.brianhuanggg.authkit.web.JsonAuthenticationEntryPoint;
import io.github.brianhuanggg.authkit.web.JsonCorsProcessor;
import java.util.List;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.annotation.Order;
import org.springframework.http.HttpMethod;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.annotation.web.configuration.EnableWebSecurity;
import org.springframework.security.config.annotation.web.configurers.AbstractHttpConfigurer;
import org.springframework.security.config.http.SessionCreationPolicy;
import org.springframework.security.web.SecurityFilterChain;
import org.springframework.security.web.access.intercept.AuthorizationFilter;
import org.springframework.web.cors.CorsConfiguration;
import org.springframework.web.cors.CorsConfigurationSource;
import org.springframework.web.cors.UrlBasedCorsConfigurationSource;
import org.springframework.web.filter.CorsFilter;

/**
 * Three chains, all with the same base: {@code /session/**}, {@code /token/**}
 * and everything else. Servlet filter order, first to last:
 * <ol>
 * <li>{@code BodySizeLimitFilter} and {@code NoStoreFilter} (servlet filters,
 * ahead of Spring Security, so nothing buffers unbounded input);</li>
 * <li>CORS, so preflights are answered before the CSRF check;</li>
 * <li>{@code CsrfHeaderFilter};</li>
 * <li>{@code RateLimitFilter} (429 before any hashing), then the authentication filters.</li>
 * </ol>
 * Spring's own CSRF, form login, HTTP Basic, logout and HttpSession are off.
 */
@Configuration(proxyBeanMethods = false)
@EnableWebSecurity
class SecurityConfig {

    private final RateLimits rateLimits;

    SecurityConfig(RateLimits rateLimits) {
        this.rateLimits = rateLimits;
    }

    static final String BEARER_CHALLENGE = "Bearer realm=\"authkit\"";

    @Bean
    @Order(1)
    SecurityFilterChain sessionChain(HttpSecurity http, AuthkitProperties props, SessionService sessions)
            throws Exception {
        base(http, props, null)
                .securityMatcher("/session/**")
                .authorizeHttpRequests(a -> a
                        .requestMatchers("/session/me").authenticated()
                        .anyRequest().permitAll())
                .addFilterBefore(new SessionAuthenticationFilter(sessions), AuthorizationFilter.class);
        return http.build();
    }

    @Bean
    @Order(2)
    SecurityFilterChain tokenChain(HttpSecurity http, AuthkitProperties props, JwtService jwt) throws Exception {
        base(http, props, BEARER_CHALLENGE)
                .securityMatcher("/token/**")
                .authorizeHttpRequests(a -> a
                        .requestMatchers("/token/me").authenticated()
                        .anyRequest().permitAll())
                // Every bearer failure, whatever the cause, is the same 401 as everywhere else.
                .oauth2ResourceServer(o -> o
                        .jwt(j -> j
                                .decoder(jwt.decoder())
                                .jwtAuthenticationConverter(JwtService::toAuthentication))
                        .authenticationEntryPoint(new JsonAuthenticationEntryPoint(BEARER_CHALLENGE))
                        .accessDeniedHandler(new JsonAccessDeniedHandler()));
        return http.build();
    }

    @Bean
    @Order(3)
    SecurityFilterChain publicChain(HttpSecurity http, AuthkitProperties props) throws Exception {
        base(http, props, null)
                .authorizeHttpRequests(a -> a.anyRequest().permitAll());
        return http.build();
    }

    private HttpSecurity base(HttpSecurity http, AuthkitProperties props, String challenge) throws Exception {
        http
                .csrf(AbstractHttpConfigurer::disable)
                .formLogin(AbstractHttpConfigurer::disable)
                .httpBasic(AbstractHttpConfigurer::disable)
                .logout(AbstractHttpConfigurer::disable)
                .requestCache(c -> c.disable())
                // No HttpSession, ever: the session cookie is our own.
                .sessionManagement(s -> s.sessionCreationPolicy(SessionCreationPolicy.STATELESS))
                // NoStoreFilter sets Cache-Control on every response.
                .headers(h -> h.cacheControl(c -> c.disable()))
                .exceptionHandling(e -> e
                        .authenticationEntryPoint(new JsonAuthenticationEntryPoint(challenge))
                        .accessDeniedHandler(new JsonAccessDeniedHandler()))
                .addFilterAfter(new CsrfHeaderFilter(props.corsOrigins()), CorsFilter.class)
                // After CSRF, before any authentication or password work.
                .addFilterAfter(new RateLimitFilter(rateLimits), CsrfHeaderFilter.class);
        // An empty list disables CORS.
        if (!props.corsOrigins().isEmpty()) {
            CorsFilter cors = new CorsFilter(corsSource(props.corsOrigins()));
            cors.setCorsProcessor(new JsonCorsProcessor());
            http.addFilter(cors);
        }
        return http;
    }

    /**
     * Explicit origins only: a wildcard cannot be combined with cookies, and
     * echoing arbitrary origins would be just as bad.
     */
    static CorsConfigurationSource corsSource(List<String> origins) {
        CorsConfiguration config = new CorsConfiguration();
        config.setAllowedOrigins(origins);
        config.setAllowCredentials(true);
        config.setAllowedMethods(List.of(HttpMethod.GET.name(), HttpMethod.POST.name(), HttpMethod.OPTIONS.name()));
        config.setAllowedHeaders(List.of("Content-Type", "Authorization", "X-Requested-With"));
        config.setMaxAge(600L);
        UrlBasedCorsConfigurationSource source = new UrlBasedCorsConfigurationSource();
        source.registerCorsConfiguration("/**", config);
        return source;
    }
}
