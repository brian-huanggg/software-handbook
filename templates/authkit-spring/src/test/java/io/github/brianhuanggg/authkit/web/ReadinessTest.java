package io.github.brianhuanggg.authkit.web;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.content;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import io.github.brianhuanggg.authkit.PostgresTestBase;
import java.sql.SQLException;
import javax.sql.DataSource;
import org.junit.jupiter.api.Test;
import org.mockito.Mockito;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.setup.MockMvcBuilders;

@AutoConfigureMockMvc
class ReadinessTest extends PostgresTestBase {

    @Autowired
    MockMvc mvc;

    @Test
    void readyWhenTheDatabaseAnswers() throws Exception {
        mvc.perform(get("/readyz")).andExpect(status().isOk()).andExpect(content().string("ok"));
    }

    @Test
    void notReadyWhenTheDatabaseIsDown() throws Exception {
        DataSource down = Mockito.mock(DataSource.class);
        Mockito.when(down.getConnection()).thenThrow(new SQLException("connection refused: secret-host"));
        MockMvc standalone = MockMvcBuilders.standaloneSetup(new ReadinessController(down)).build();

        standalone.perform(get("/readyz"))
                .andExpect(status().isServiceUnavailable())
                .andExpect(jsonPath("$.error").value("not ready")); // no database detail leaks
    }
}
