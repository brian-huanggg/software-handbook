package io.github.brianhuanggg.authkit.config;

import org.flywaydb.core.Flyway;
import org.flywaydb.core.api.FlywayException;
import org.springframework.boot.flyway.autoconfigure.FlywayMigrationStrategy;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.jdbc.core.simple.JdbcClient;

/**
 * Lets Flyway adopt a schema that the Go stack's golang-migrate created, and
 * nothing else.
 *
 * <p>A blanket {@code baseline-on-migrate} would mark any non-empty schema
 * without a Flyway history as already migrated, including one that only holds
 * unrelated tables; the app would then start without its own tables. Instead
 * the baseline is set only when golang-migrate's {@code schema_migrations}
 * table says the Go migrations are applied, clean, at exactly the version our
 * Flyway migrations end at. Every other non-empty schema without a history
 * fails at startup, as plain Flyway does.
 */
@Configuration(proxyBeanMethods = false)
class FlywayConfig {

    /**
     * The golang-migrate version that matches {@code V1__} to {@code V2__}.
     * Raise it, and {@code spring.flyway.baseline-version}, whenever a migration
     * is added to both projects.
     */
    static final long GO_SCHEMA_VERSION = 2;

    @Bean
    FlywayMigrationStrategy adoptGoSchema() {
        return FlywayConfig::migrate;
    }

    static void migrate(Flyway flyway) {
        if (flyway.info().current() == null) { // no Flyway history yet
            JdbcClient jdbc = JdbcClient.create(flyway.getConfiguration().getDataSource());
            if (goMigrateTableExists(jdbc)) {
                GoVersion go = jdbc.sql("SELECT version, dirty FROM schema_migrations")
                        .query((rs, row) -> new GoVersion(rs.getLong("version"), rs.getBoolean("dirty")))
                        .optional()
                        .orElseThrow(() -> new FlywayException(
                                "schema_migrations exists but is empty: the Go migrations never ran to completion"));
                if (go.dirty()) {
                    throw new FlywayException("golang-migrate left the schema dirty at version " + go.version()
                            + "; fix it with `migrate force` before starting this API");
                }
                if (go.version() != GO_SCHEMA_VERSION) {
                    throw new FlywayException("golang-migrate is at version " + go.version()
                            + " but this API's migrations end at " + GO_SCHEMA_VERSION
                            + "; add the missing Flyway migration first");
                }
                String baseline = flyway.getConfiguration().getBaselineVersion().getVersion();
                if (!baseline.equals(Long.toString(GO_SCHEMA_VERSION))) {
                    throw new FlywayException("spring.flyway.baseline-version is " + baseline
                            + " but must be " + GO_SCHEMA_VERSION + " to adopt the Go schema");
                }
                flyway.baseline();
            }
        }
        flyway.migrate();
    }

    private static boolean goMigrateTableExists(JdbcClient jdbc) {
        // Unqualified, so it follows search_path like golang-migrate and Flyway do.
        return jdbc.sql("SELECT to_regclass('schema_migrations') IS NOT NULL").query(Boolean.class).single();
    }

    private record GoVersion(long version, boolean dirty) {
    }
}
