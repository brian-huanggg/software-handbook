package io.github.brianhuanggg.authkit.config;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import io.github.brianhuanggg.authkit.PostgresTestBase;
import java.io.IOException;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Statement;
import java.util.UUID;
import org.flywaydb.core.Flyway;
import org.flywaydb.core.api.FlywayException;
import org.junit.jupiter.api.Test;
import org.testcontainers.postgresql.PostgreSQLContainer;

/**
 * {@link FlywayConfig#migrate}: which schemas Flyway adopts. Each test gets a
 * fresh database in the shared Postgres container.
 */
class FlywayAdoptionTest {

    private static final PostgreSQLContainer PG = PostgresTestBase.POSTGRES;

    private static String newDatabase() throws SQLException {
        String name = "flyway_" + UUID.randomUUID().toString().replace("-", "");
        try (Connection c = DriverManager.getConnection(PG.getJdbcUrl(), PG.getUsername(), PG.getPassword());
                Statement s = c.createStatement()) {
            s.execute("CREATE DATABASE " + name);
        }
        return "jdbc:postgresql://%s:%d/%s".formatted(PG.getHost(), PG.getMappedPort(5432), name);
    }

    private static void exec(String url, String sql) throws SQLException {
        try (Connection c = DriverManager.getConnection(url, PG.getUsername(), PG.getPassword());
                Statement s = c.createStatement()) {
            s.execute(sql);
        }
    }

    private static boolean tableExists(String url, String table) throws SQLException {
        try (Connection c = DriverManager.getConnection(url, PG.getUsername(), PG.getPassword());
                Statement s = c.createStatement();
                ResultSet rs = s.executeQuery("SELECT to_regclass('" + table + "') IS NOT NULL")) {
            rs.next();
            return rs.getBoolean(1);
        }
    }

    private static String migration(String file) throws IOException {
        try (InputStream in = FlywayAdoptionTest.class.getResourceAsStream("/db/migration/" + file)) {
            return new String(in.readAllBytes(), StandardCharsets.UTF_8);
        }
    }

    /** What the Go stack's migrate service leaves behind: the schema plus golang-migrate's table. */
    private static void goSchema(String url, long version, boolean dirty) throws Exception {
        exec(url, migration("V1__init_auth.sql"));
        exec(url, migration("V2__refresh_family_expiry.sql"));
        exec(url, "CREATE TABLE schema_migrations (version bigint NOT NULL PRIMARY KEY, dirty boolean NOT NULL)");
        exec(url, "INSERT INTO schema_migrations VALUES (%d, %b)".formatted(version, dirty));
    }

    private static Flyway flyway(String url) {
        // The same settings application.yml gives Boot's Flyway.
        return Flyway.configure()
                .dataSource(url, PG.getUsername(), PG.getPassword())
                .locations("classpath:db/migration")
                .baselineVersion("2")
                .load();
    }

    @Test
    void emptyDatabaseRunsEveryMigration() throws Exception {
        String url = newDatabase();
        FlywayConfig.migrate(flyway(url));
        assertThat(tableExists(url, "users")).isTrue();
        assertThat(flyway(url).info().current().getVersion().getVersion()).isEqualTo("2");
    }

    @Test
    void cleanGoSchemaIsAdoptedAtTheBaseline() throws Exception {
        String url = newDatabase();
        goSchema(url, FlywayConfig.GO_SCHEMA_VERSION, false);
        FlywayConfig.migrate(flyway(url));
        var current = flyway(url).info().current();
        assertThat(current.getVersion().getVersion()).isEqualTo("2");
        assertThat(current.getType().isBaseline()).as("baselined, not re-run").isTrue();
        // Idempotent: a second start with a history just migrates (nothing to do).
        FlywayConfig.migrate(flyway(url));
    }

    @Test
    void unrelatedNonEmptySchemaIsNotAdopted() throws Exception {
        String url = newDatabase();
        exec(url, "CREATE TABLE someone_elses (id int)");
        assertThatThrownBy(() -> FlywayConfig.migrate(flyway(url))).isInstanceOf(FlywayException.class);
        assertThat(tableExists(url, "users")).as("no tables were assumed to exist").isFalse();
        assertThat(tableExists(url, "flyway_schema_history")).as("no baseline was written").isFalse();
    }

    @Test
    void dirtyGoSchemaIsRefused() throws Exception {
        String url = newDatabase();
        goSchema(url, FlywayConfig.GO_SCHEMA_VERSION, true);
        assertThatThrownBy(() -> FlywayConfig.migrate(flyway(url)))
                .isInstanceOf(FlywayException.class).hasMessageContaining("dirty");
    }

    @Test
    void goSchemaAtAnotherVersionIsRefused() throws Exception {
        String url = newDatabase();
        goSchema(url, FlywayConfig.GO_SCHEMA_VERSION + 1, false);
        assertThatThrownBy(() -> FlywayConfig.migrate(flyway(url)))
                .isInstanceOf(FlywayException.class).hasMessageContaining("version 3");
        assertThat(tableExists(url, "flyway_schema_history")).isFalse();
    }
}
