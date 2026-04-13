package com.example;

import com.sun.net.httpserver.HttpServer;
import com.sun.net.httpserver.HttpExchange;
import io.featurebacon.BaconClient;
import io.featurebacon.EvaluationContext;
import io.featurebacon.EvaluationResult;

import java.io.IOException;
import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.net.URLDecoder;
import java.nio.charset.StandardCharsets;
import java.util.*;
import java.util.logging.Logger;

public class ProductService {
    private static final Logger LOG = Logger.getLogger(ProductService.class.getName());
    private final BaconClient client;

    public ProductService(BaconClient client) {
        this.client = client;
    }

    public static void main(String[] args) throws IOException {
        String baconUrl = env("BACON_URL", "http://localhost:8080");
        String apiKey = env("BACON_API_KEY", "");
        int port = Integer.parseInt(env("PORT", "3000"));

        BaconClient client = BaconClient.builder(baconUrl).apiKey(apiKey).build();
        ProductService svc = new ProductService(client);

        HttpServer server = HttpServer.create(new InetSocketAddress(port), 0);
        server.createContext("/", svc::handleHome);
        server.createContext("/products", svc::handleProducts);
        server.createContext("/health", svc::handleHealth);
        server.start();
        LOG.info(() -> "Product service on :" + port);
    }

    void handleHome(HttpExchange ex) throws IOException {
        EvaluationContext ctx = contextFromQuery(ex);
        try {
            List<EvaluationResult> results = client.evaluateBatch(
                List.of("dark_mode", "new_pricing", "beta_features", "checkout_redesign"),
                ctx
            );
            StringBuilder json = new StringBuilder("{\"service\":\"product-service-java\",\"user\":\"" + ctx.getSubjectId() + "\",\"features\":{");
            for (int i = 0; i < results.size(); i++) {
                EvaluationResult r = results.get(i);
                if (i > 0) json.append(",");
                json.append("\"").append(r.getFlagKey()).append("\":{")
                    .append("\"enabled\":").append(r.isEnabled())
                    .append(",\"variant\":\"").append(r.getVariant()).append("\"")
                    .append(",\"reason\":\"").append(r.getReason()).append("\"}");
            }
            json.append("}}");
            respond(ex, 200, json.toString());
        } catch (Exception e) {
            respond(ex, 500, "{\"error\":\"" + e.getMessage() + "\"}");
        }
    }

    void handleProducts(HttpExchange ex) throws IOException {
        EvaluationContext ctx = contextFromQuery(ex);
        boolean newPricing = client.isEnabled("new_pricing", ctx);
        String variant = client.getVariant("checkout_redesign", ctx);
        double discount = newPricing ? 0.9 : 1.0;

        String json = String.format(
            "{\"products\":[" +
            "{\"id\":1,\"name\":\"Widget Pro\",\"price\":%.2f}," +
            "{\"id\":2,\"name\":\"Widget Basic\",\"price\":%.2f}," +
            "{\"id\":3,\"name\":\"Widget Enterprise\",\"price\":%.2f}" +
            "],\"checkoutVariant\":\"%s\",\"newPricingActive\":%b}",
            29.99 * discount, 9.99 * discount, 99.99 * discount, variant, newPricing
        );
        respond(ex, 200, json);
    }

    void handleHealth(HttpExchange ex) throws IOException {
        boolean healthy = client.healthy();
        int code = healthy ? 200 : 503;
        respond(ex, code, String.format(
            "{\"status\":\"%s\",\"baconHealthy\":%b}",
            healthy ? "ok" : "degraded", healthy
        ));
    }

    private EvaluationContext contextFromQuery(HttpExchange ex) {
        Map<String, String> params = parseQuery(ex.getRequestURI().getQuery());
        return EvaluationContext.builder(params.getOrDefault("user", "anonymous"))
            .environment(env("ENVIRONMENT", "production"))
            .attribute("plan", params.getOrDefault("plan", "free"))
            .build();
    }

    private static Map<String, String> parseQuery(String query) {
        Map<String, String> map = new HashMap<>();
        if (query == null) return map;
        for (String pair : query.split("&")) {
            String[] kv = pair.split("=", 2);
            if (kv.length == 2) map.put(kv[0], URLDecoder.decode(kv[1], StandardCharsets.UTF_8));
        }
        return map;
    }

    private static void respond(HttpExchange ex, int status, String body) throws IOException {
        byte[] bytes = body.getBytes(StandardCharsets.UTF_8);
        ex.getResponseHeaders().set("Content-Type", "application/json");
        ex.sendResponseHeaders(status, bytes.length);
        try (OutputStream os = ex.getResponseBody()) {
            os.write(bytes);
        }
    }

    private static String env(String key, String fallback) {
        String v = System.getenv(key);
        return v != null && !v.isEmpty() ? v : fallback;
    }
}
