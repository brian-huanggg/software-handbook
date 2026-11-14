package io.github.brianhuanggg.authkit.web;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.content;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc;
import org.springframework.context.annotation.Import;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@SpringBootTest(properties = {
        "authkit.database-url=postgres://u:p@localhost:5432/authkit",
        "authkit.jwt-secret=kkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkk",
        "spring.flyway.enabled=false"})
@AutoConfigureMockMvc
@Import(ScaffoldTest.Echo.class)
class ScaffoldTest {

    @RestController
    static class Echo {
        record Body(String name) {
        }

        @PostMapping("/test-echo")
        String echo(@RequestBody Body body) {
            return body.name();
        }
    }

    @Autowired
    MockMvc mvc;

    @Test
    void healthz() throws Exception {
        mvc.perform(get("/healthz")).andExpect(status().isOk()).andExpect(content().string("ok"));
    }

    @Test
    void rejectsUnknownJsonFields() throws Exception {
        mvc.perform(post("/test-echo").header("X-Requested-With", "XMLHttpRequest").contentType(MediaType.APPLICATION_JSON)
                        .content("{\"name\":\"a\",\"extra\":1}"))
                .andExpect(status().isBadRequest());
    }

    @Test
    void acceptsKnownJsonFields() throws Exception {
        mvc.perform(post("/test-echo").header("X-Requested-With", "XMLHttpRequest").contentType(MediaType.APPLICATION_JSON).content("{\"name\":\"a\"}"))
                .andExpect(status().isOk()).andExpect(content().string("a"));
    }

    @Test
    void bodyOfExactlyLimitPasses() throws Exception {
        String body = "{\"name\":\"" + "a".repeat(BodySizeLimitFilter.MAX_BODY_BYTES - 11) + "\"}";
        mvc.perform(post("/test-echo").header("X-Requested-With", "XMLHttpRequest").contentType(MediaType.APPLICATION_JSON).content(body))
                .andExpect(status().isOk());
    }

    @Test
    void bodyOverLimitIs413() throws Exception {
        String body = "{\"name\":\"" + "a".repeat(BodySizeLimitFilter.MAX_BODY_BYTES - 10) + "\"}";
        mvc.perform(post("/test-echo").header("X-Requested-With", "XMLHttpRequest").contentType(MediaType.APPLICATION_JSON).content(body))
                .andExpect(status().isPayloadTooLarge())
                .andExpect(header().string("Cache-Control", "no-store"))
                .andExpect(content().json("{\"error\":\"request body too large\"}"));
    }
}
