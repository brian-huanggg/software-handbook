package io.github.brianhuanggg.authkit;

import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.context.DynamicPropertySource;
import org.testcontainers.postgresql.PostgreSQLContainer;

/**
 * Real Postgres (same major as compose) with Flyway applied, shared by every
 * subclass in the JVM. The tests use unique emails instead of truncating.
 */
// Bursts are high so unrelated tests, all calling from one address, never hit
// a limit. The rate limit tests lower them.
// A subclass's own @TestPropertySource overrides the bursts; @SpringBootTest properties would win over it.
@SpringBootTest(properties = "authkit.jwt-secret=kkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkk")
@TestPropertySource(properties = {
        "authkit.rate-limit-ip-burst=100000",
        "authkit.rate-limit-email-burst=100000"})
public abstract class PostgresTestBase {

    static final PostgreSQLContainer POSTGRES = new PostgreSQLContainer("postgres:16");

    static {
        POSTGRES.start();
    }

    @DynamicPropertySource
    static void database(DynamicPropertyRegistry registry) {
        registry.add("authkit.database-url", () -> "postgres://%s:%s@%s:%d/%s".formatted(
                POSTGRES.getUsername(), POSTGRES.getPassword(), POSTGRES.getHost(),
                POSTGRES.getMappedPort(5432), POSTGRES.getDatabaseName()));
    }
}
