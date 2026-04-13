package io.featurebacon;

import com.sun.net.httpserver.HttpServer;
import com.sun.net.httpserver.HttpExchange;
import org.junit.After;
import org.junit.Before;
import org.junit.Test;

import java.io.IOException;
import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.List;

import static org.junit.Assert.*;

public class BaconClientTest {

    private HttpServer server;
    private BaconClient client;

    @Before
    public void setUp() throws Exception {
        server = HttpServer.create(new InetSocketAddress(0), 0);
        int port = server.getAddress().getPort();
        client = BaconClient.builder("http://localhost:" + port)
                .apiKey("test-key")
                .build();
    }

    @After
    public void tearDown() {
        server.stop(0);
    }

    // ── evaluate ────────────────────────────────────────────────────

    @Test
    public void testEvaluateReturnsResult() throws Exception {
        String responseBody = "{\"tenant_id\":\"t1\",\"flag_key\":\"dark-mode\","
                + "\"enabled\":true,\"variant\":\"on\",\"reason\":\"match\"}";

        server.createContext("/api/v1/evaluate", exchange -> {
            assertApiKey(exchange);
            assertMethod(exchange, "POST");
            respond(exchange, 200, responseBody);
        });
        server.start();

        EvaluationContext ctx = EvaluationContext.builder("user-1")
                .environment("prod")
                .attribute("plan", "pro")
                .build();

        EvaluationResult result = client.evaluate("dark-mode", ctx);

        assertEquals("t1", result.getTenantId());
        assertEquals("dark-mode", result.getFlagKey());
        assertTrue(result.isEnabled());
        assertEquals("on", result.getVariant());
        assertEquals("match", result.getReason());
    }

    @Test
    public void testEvaluateThrowsBaconErrorOnApiError() {
        String errorBody = "{\"type\":\"about:blank\",\"title\":\"Unauthorized\","
                + "\"detail\":\"Invalid API key\",\"instance\":\"/api/v1/evaluate\"}";

        server.createContext("/api/v1/evaluate", exchange -> respond(exchange, 401, errorBody));
        server.start();

        EvaluationContext ctx = EvaluationContext.builder("user-1").build();

        try {
            client.evaluate("some-flag", ctx);
            fail("Expected BaconError");
        } catch (BaconError e) {
            assertEquals(401, e.getStatusCode());
            assertEquals("Unauthorized", e.getTitle());
            assertEquals("Invalid API key", e.getDetail());
        }
    }

    // ── evaluateBatch ───────────────────────────────────────────────

    @Test
    public void testEvaluateBatchReturnsMultipleResults() throws Exception {
        String responseBody = "["
                + "{\"tenant_id\":\"t1\",\"flag_key\":\"flag-a\",\"enabled\":true,\"variant\":\"v1\",\"reason\":\"r1\"},"
                + "{\"tenant_id\":\"t1\",\"flag_key\":\"flag-b\",\"enabled\":false,\"variant\":\"\",\"reason\":\"r2\"}"
                + "]";

        server.createContext("/api/v1/evaluate/batch", exchange -> {
            assertApiKey(exchange);
            assertMethod(exchange, "POST");
            String body = new String(exchange.getRequestBody().readAllBytes(), StandardCharsets.UTF_8);
            assertTrue(body.contains("flag-a"));
            assertTrue(body.contains("flag-b"));
            respond(exchange, 200, responseBody);
        });
        server.start();

        EvaluationContext ctx = EvaluationContext.builder("user-1").build();
        List<EvaluationResult> results = client.evaluateBatch(Arrays.asList("flag-a", "flag-b"), ctx);

        assertEquals(2, results.size());
        assertTrue(results.get(0).isEnabled());
        assertFalse(results.get(1).isEnabled());
        assertEquals("flag-a", results.get(0).getFlagKey());
        assertEquals("flag-b", results.get(1).getFlagKey());
    }

    // ── isEnabled / getVariant convenience ──────────────────────────

    @Test
    public void testIsEnabledReturnsFalseOnError() {
        server.createContext("/api/v1/evaluate", exchange -> respond(exchange, 500, "{}"));
        server.start();

        EvaluationContext ctx = EvaluationContext.builder("user-1").build();
        assertFalse(client.isEnabled("flag", ctx));
    }

    @Test
    public void testGetVariantReturnsEmptyOnError() {
        server.createContext("/api/v1/evaluate", exchange -> respond(exchange, 500, "{}"));
        server.start();

        EvaluationContext ctx = EvaluationContext.builder("user-1").build();
        assertEquals("", client.getVariant("flag", ctx));
    }

    // ── healthz / readyz ────────────────────────────────────────────

    @Test
    public void testHealthyReturnsTrue() {
        server.createContext("/healthz", exchange -> respond(exchange, 200, "{\"status\":\"healthy\"}"));
        server.start();

        assertTrue(client.healthy());
    }

    @Test
    public void testHealthyReturnsFalseOnError() {
        server.createContext("/healthz", exchange -> respond(exchange, 503, "{\"status\":\"unhealthy\"}"));
        server.start();

        assertFalse(client.healthy());
    }

    @Test
    public void testReadyReturnsHealthResponse() throws Exception {
        String responseBody = "{\"status\":\"ready\",\"modules\":{"
                + "\"database\":{\"status\":\"healthy\",\"latency_ms\":5,\"message\":\"ok\"},"
                + "\"cache\":{\"status\":\"healthy\",\"latency_ms\":1,\"message\":\"\"}"
                + "}}";

        server.createContext("/readyz", exchange -> respond(exchange, 200, responseBody));
        server.start();

        HealthResponse resp = client.ready();
        assertEquals("ready", resp.getStatus());
        assertEquals(2, resp.getModules().size());
        assertEquals("healthy", resp.getModules().get("database").getStatus());
        assertEquals(5, resp.getModules().get("database").getLatencyMs());
        assertEquals("ok", resp.getModules().get("database").getMessage());
    }

    @Test
    public void testReadyThrowsOnServerError() {
        String errorBody = "{\"type\":\"about:blank\",\"title\":\"Service Unavailable\","
                + "\"detail\":\"Not ready\",\"instance\":\"/readyz\"}";

        server.createContext("/readyz", exchange -> respond(exchange, 503, errorBody));
        server.start();

        try {
            client.ready();
            fail("Expected BaconError");
        } catch (BaconError e) {
            assertEquals(503, e.getStatusCode());
            assertEquals("Service Unavailable", e.getTitle());
        }
    }

    // ── request body validation ─────────────────────────────────────

    @Test
    public void testEvaluateSendsCorrectBody() throws Exception {
        server.createContext("/api/v1/evaluate", exchange -> {
            String body = new String(exchange.getRequestBody().readAllBytes(), StandardCharsets.UTF_8);
            assertTrue("Body should contain flag_key", body.contains("\"flag_key\":\"my-flag\""));
            assertTrue("Body should contain subject_id", body.contains("\"subject_id\":\"u1\""));
            assertTrue("Body should contain environment", body.contains("\"environment\":\"staging\""));
            respond(exchange, 200,
                    "{\"tenant_id\":\"t1\",\"flag_key\":\"my-flag\",\"enabled\":false,\"variant\":\"\",\"reason\":\"off\"}");
        });
        server.start();

        EvaluationContext ctx = EvaluationContext.builder("u1")
                .environment("staging")
                .build();
        client.evaluate("my-flag", ctx);
    }

    // ── builder defaults ────────────────────────────────────────────

    @Test
    public void testBuilderStripsTrailingSlash() throws Exception {
        int port = server.getAddress().getPort();
        BaconClient c = BaconClient.builder("http://localhost:" + port + "///")
                .apiKey("k")
                .build();

        server.createContext("/healthz", exchange -> respond(exchange, 200, "{\"status\":\"healthy\"}"));
        server.start();

        assertTrue(c.healthy());
    }

    // ── interrupt handling ──────────────────────────────────────────

    @Test
    public void testEvaluateReInterruptsOnInterruptedException() {
        server.start();
        EvaluationContext ctx = EvaluationContext.builder("user-1").build();

        Thread.currentThread().interrupt();
        try {
            client.evaluate("flag", ctx);
            fail("Expected BaconError");
        } catch (BaconError e) {
            assertTrue("Thread interrupt flag must be preserved", Thread.interrupted());
        }
    }

    @Test
    public void testHealthyReturnsFalseOnInterrupt() {
        server.start();
        Thread.currentThread().interrupt();
        assertFalse(client.healthy());
        Thread.interrupted();
    }

    // ── no API key ───────────────────────────────────────────────────

    @Test
    public void testRequestsWorkWithoutApiKey() throws Exception {
        int port = server.getAddress().getPort();
        BaconClient noKeyClient = BaconClient.builder("http://localhost:" + port).build();

        server.createContext("/healthz", exchange -> {
            assertNull(exchange.getRequestHeaders().getFirst("X-API-Key"));
            respond(exchange, 200, "{\"status\":\"healthy\"}");
        });
        server.start();

        assertTrue(noKeyClient.healthy());
    }

    // ── helpers ─────────────────────────────────────────────────────

    private static void respond(HttpExchange exchange, int status, String body) throws IOException {
        byte[] bytes = body.getBytes(StandardCharsets.UTF_8);
        exchange.getResponseHeaders().set("Content-Type", "application/json");
        exchange.sendResponseHeaders(status, bytes.length);
        try (OutputStream os = exchange.getResponseBody()) {
            os.write(bytes);
        }
    }

    private static void assertApiKey(HttpExchange exchange) {
        assertEquals("test-key", exchange.getRequestHeaders().getFirst("X-API-Key"));
    }

    private static void assertMethod(HttpExchange exchange, String method) {
        assertEquals(method, exchange.getRequestMethod());
    }
}
