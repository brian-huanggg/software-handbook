package io.github.brianhuanggg.authkit;

import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.DynamicPropertyRegistry;
import org.springframework.test.context.DynamicPropertySource;
import org.testcontainers.postgresql.PostgreSQLContainer;

/**
 * Real Postgres (same major as compose) with Flyway applied, shared by every
 * subclass in the JVM. The tests use unique emails instead of truncating.
 */
@SpringBootTest(properties = "authkit.jwt-secret=kkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkk")
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
