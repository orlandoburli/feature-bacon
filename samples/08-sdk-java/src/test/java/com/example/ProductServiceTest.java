package com.example;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpServer;
import io.featurebacon.BaconClient;
import org.junit.jupiter.api.*;

import java.io.IOException;
import java.io.OutputStream;
import java.lang.reflect.Method;
import java.net.InetSocketAddress;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;

import static org.junit.jupiter.api.Assertions.*;

class ProductServiceTest {

    private HttpServer mockBacon;
    private HttpServer appServer;
    private HttpClient http;
    private String appBase;

    @BeforeEach
    void setUp() {
        http = HttpClient.newHttpClient();
    }

    @AfterEach
    void tearDown() {
        if (appServer != null) { appServer.stop(0); appServer = null; }
        if (mockBacon != null) { mockBacon.stop(0); mockBacon = null; }
    }

    @Test
    void testHealthWhenBaconIsUp() throws Exception {
        boot(true, false, false);

        HttpResponse<String> resp = get("/health");
        assertEquals(200, resp.statusCode());
        assertTrue(resp.body().contains("\"status\":\"ok\""));
        assertTrue(resp.body().contains("\"baconHealthy\":true"));
    }

    @Test
    void testHealthWhenBaconIsDown() throws Exception {
        boot(false, false, false);

        HttpResponse<String> resp = get("/health");
        assertEquals(503, resp.statusCode());
        assertTrue(resp.body().contains("\"status\":\"degraded\""));
        assertTrue(resp.body().contains("\"baconHealthy\":false"));
    }

    @Test
    void testHomeReturnsFeatures() throws Exception {
        boot(true, false, true);

        HttpResponse<String> resp = get("/");
        assertEquals(200, resp.statusCode());
        String body = resp.body();
        assertTrue(body.contains("\"service\":\"product-service-java\""));
        assertTrue(body.contains("\"dark_mode\":{"));
        assertTrue(body.contains("\"new_pricing\":{"));
        assertTrue(body.contains("\"beta_features\":{"));
        assertTrue(body.contains("\"checkout_redesign\":{"));
    }

    @Test
    void testHomeWithUser() throws Exception {
        boot(true, false, false);

        HttpResponse<String> resp = get("/?user=alice");
        assertEquals(200, resp.statusCode());
        assertTrue(resp.body().contains("\"user\":\"alice\""));
    }

    @Test
    void testHomeDefaultUser() throws Exception {
        boot(true, false, false);

        HttpResponse<String> resp = get("/");
        assertEquals(200, resp.statusCode());
        assertTrue(resp.body().contains("\"user\":\"anonymous\""));
    }

    @Test
    void testHomeHandlesError() throws Exception {
        boot(true, true, false);

        HttpResponse<String> resp = get("/");
        assertEquals(500, resp.statusCode());
        assertTrue(resp.body().contains("\"error\":"));
    }

    @Test
    void testProductsNewPricingEnabled() throws Exception {
        boot(true, false, true);

        HttpResponse<String> resp = get("/products");
        assertEquals(200, resp.statusCode());
        String body = resp.body();
        assertTrue(body.contains("\"newPricingActive\":true"));
        assertTrue(body.contains("\"checkoutVariant\":\"modern\""));
        assertTrue(body.contains("\"price\":26.99"));
        assertTrue(body.contains("\"price\":8.99"));
        assertTrue(body.contains("\"price\":89.99"));
    }

    @Test
    void testProductsNewPricingDisabled() throws Exception {
        boot(true, false, false);

        HttpResponse<String> resp = get("/products");
        assertEquals(200, resp.statusCode());
        String body = resp.body();
        assertTrue(body.contains("\"newPricingActive\":false"));
        assertTrue(body.contains("\"price\":29.99"));
        assertTrue(body.contains("\"price\":9.99"));
        assertTrue(body.contains("\"price\":99.99"));
    }

    // ── helpers ─────────────────────────────────────────────────────

    private void boot(boolean healthOK, boolean evaluateError, boolean newPricingEnabled) throws Exception {
        mockBacon = startMockBacon(healthOK, evaluateError, newPricingEnabled);
        BaconClient client = BaconClient.builder(
                "http://localhost:" + mockBacon.getAddress().getPort()
        ).build();
        ProductService svc = new ProductService(client);

        appServer = HttpServer.create(new InetSocketAddress(0), 0);
        for (String[] m : new String[][]{{"/", "handleHome"}, {"/products", "handleProducts"}, {"/health", "handleHealth"}}) {
            Method method = ProductService.class.getDeclaredMethod(m[1], HttpExchange.class);
            method.setAccessible(true);
            appServer.createContext(m[0], ex -> {
                try {
                    method.invoke(svc, ex);
                } catch (java.lang.reflect.InvocationTargetException ite) {
                    Throwable cause = ite.getCause();
                    if (cause instanceof IOException) throw (IOException) cause;
                    throw new IOException(cause);
                } catch (Exception e) {
                    throw new IOException(e);
                }
            });
        }
        appServer.start();
        appBase = "http://localhost:" + appServer.getAddress().getPort();
    }

    private HttpServer startMockBacon(boolean healthOK, boolean evaluateError, boolean newPricingEnabled) throws IOException {
        HttpServer mock = HttpServer.create(new InetSocketAddress(0), 0);

        mock.createContext("/healthz", ex -> {
            if (healthOK) {
                mockRespond(ex, 200, "{\"status\":\"healthy\"}");
            } else {
                mockRespond(ex, 503, "{\"status\":\"unhealthy\"}");
            }
        });

        mock.createContext("/api/v1/evaluate/batch", ex -> {
            if (evaluateError) {
                mockRespond(ex, 500, "{\"title\":\"Internal Server Error\"}");
                return;
            }
            ex.getRequestBody().readAllBytes();
            String result = "[" +
                    "{\"tenant_id\":\"t\",\"flag_key\":\"dark_mode\",\"enabled\":false,\"variant\":\"off\",\"reason\":\"default\"}," +
                    String.format("{\"tenant_id\":\"t\",\"flag_key\":\"new_pricing\",\"enabled\":%b,\"variant\":\"%s\",\"reason\":\"rule_match\"},",
                            newPricingEnabled, newPricingEnabled ? "on" : "off") +
                    "{\"tenant_id\":\"t\",\"flag_key\":\"beta_features\",\"enabled\":false,\"variant\":\"off\",\"reason\":\"default\"}," +
                    "{\"tenant_id\":\"t\",\"flag_key\":\"checkout_redesign\",\"enabled\":true,\"variant\":\"modern\",\"reason\":\"rule_match\"}" +
                    "]";
            mockRespond(ex, 200, result);
        });

        mock.createContext("/api/v1/evaluate", ex -> {
            if (evaluateError) {
                mockRespond(ex, 500, "{\"title\":\"Internal Server Error\"}");
                return;
            }
            String body = new String(ex.getRequestBody().readAllBytes(), StandardCharsets.UTF_8);
            String flagKey = extractJsonString(body, "flag_key");
            boolean enabled = "new_pricing".equals(flagKey) && newPricingEnabled;
            String variant = "checkout_redesign".equals(flagKey) ? "modern" : (enabled ? "on" : "off");
            mockRespond(ex, 200, String.format(
                    "{\"tenant_id\":\"t\",\"flag_key\":\"%s\",\"enabled\":%b,\"variant\":\"%s\",\"reason\":\"rule_match\"}",
                    flagKey, enabled, variant
            ));
        });

        mock.start();
        return mock;
    }

    private HttpResponse<String> get(String path) throws Exception {
        return http.send(
                HttpRequest.newBuilder().uri(URI.create(appBase + path)).GET().build(),
                HttpResponse.BodyHandlers.ofString()
        );
    }

    private static void mockRespond(HttpExchange ex, int status, String body) throws IOException {
        byte[] bytes = body.getBytes(StandardCharsets.UTF_8);
        ex.getResponseHeaders().set("Content-Type", "application/json");
        ex.sendResponseHeaders(status, bytes.length);
        try (OutputStream os = ex.getResponseBody()) {
            os.write(bytes);
        }
    }

    private static String extractJsonString(String json, String key) {
        String needle = "\"" + key + "\":\"";
        int start = json.indexOf(needle);
        if (start == -1) return "";
        start += needle.length();
        int end = json.indexOf('"', start);
        return json.substring(start, end);
    }
}
