package io.featurebacon;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * Thread-safe client for the Feature Bacon flag evaluation API.
 *
 * <pre>{@code
 * BaconClient client = BaconClient.builder("http://localhost:8080")
 *         .apiKey("my-api-key")
 *         .build();
 *
 * EvaluationContext ctx = EvaluationContext.builder("user-123")
 *         .environment("production")
 *         .attribute("plan", "pro")
 *         .build();
 *
 * EvaluationResult result = client.evaluate("dark-mode", ctx);
 * }</pre>
 */
public class BaconClient {
    private final String baseUrl;
    private final String apiKey;
    private final HttpClient httpClient;
    private final Duration timeout;

    private BaconClient(Builder builder) {
        String url = builder.baseUrl;
        while (url.endsWith("/")) {
            url = url.substring(0, url.length() - 1);
        }
        this.baseUrl = url;
        this.apiKey = builder.apiKey;
        this.timeout = builder.timeout;
        this.httpClient = builder.httpClient != null
                ? builder.httpClient
                : HttpClient.newBuilder().connectTimeout(timeout).build();
    }

    public static Builder builder(String baseUrl) {
        return new Builder(baseUrl);
    }

    /**
     * Evaluate a single feature flag.
     */
    public EvaluationResult evaluate(String flagKey, EvaluationContext context) throws BaconError {
        Map<String, Object> body = new LinkedHashMap<>();
        body.put("flag_key", flagKey);
        body.put("context", contextMap(context));

        String json = post("/api/v1/evaluate", JsonHelper.toJson(body));
        return EvaluationResult.fromJson(json);
    }

    /**
     * Evaluate multiple feature flags in a single request.
     */
    public List<EvaluationResult> evaluateBatch(List<String> flagKeys, EvaluationContext context) throws BaconError {
        List<Object> keys = new ArrayList<>(flagKeys);
        Map<String, Object> body = new LinkedHashMap<>();
        body.put("flag_keys", keys);
        body.put("context", contextMap(context));

        String json = post("/api/v1/evaluate/batch", JsonHelper.toJson(body));
        return EvaluationResult.listFromJson(json);
    }

    /**
     * Convenience: returns whether a flag is enabled, defaulting to {@code false} on any error.
     */
    public boolean isEnabled(String flagKey, EvaluationContext context) {
        try {
            return evaluate(flagKey, context).isEnabled();
        } catch (Exception e) {
            return false;
        }
    }

    /**
     * Convenience: returns the variant string, defaulting to {@code ""} on any error.
     */
    public String getVariant(String flagKey, EvaluationContext context) {
        try {
            return evaluate(flagKey, context).getVariant();
        } catch (Exception e) {
            return "";
        }
    }

    /**
     * Returns {@code true} if the server's /healthz endpoint reports healthy.
     */
    public boolean healthy() {
        try {
            String json = get("/healthz");
            JsonHelper.JsonObject obj = JsonHelper.parseObject(json);
            return "healthy".equalsIgnoreCase(obj.getString("status", ""));
        } catch (Exception e) {
            return false;
        }
    }

    /**
     * Returns the full readiness response from /readyz.
     */
    public HealthResponse ready() throws BaconError {
        String json = get("/readyz");
        return HealthResponse.fromJson(json);
    }

    // ── HTTP helpers ────────────────────────────────────────────────

    private String post(String path, String jsonBody) throws BaconError {
        return send(HttpRequest.newBuilder()
                .uri(URI.create(baseUrl + path))
                .header("Content-Type", "application/json")
                .timeout(timeout)
                .POST(HttpRequest.BodyPublishers.ofString(jsonBody)));
    }

    private String get(String path) throws BaconError {
        return send(HttpRequest.newBuilder()
                .uri(URI.create(baseUrl + path))
                .timeout(timeout)
                .GET());
    }

    private String send(HttpRequest.Builder reqBuilder) throws BaconError {
        if (apiKey != null && !apiKey.isEmpty()) {
            reqBuilder.header("X-API-Key", apiKey);
        }
        try {
            HttpResponse<String> resp = httpClient.send(reqBuilder.build(), HttpResponse.BodyHandlers.ofString());
            return handleResponse(resp);
        } catch (BaconError e) {
            throw e;
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new BaconError("Request interrupted", e);
        } catch (Exception e) {
            throw new BaconError("Request failed: " + e.getMessage(), e);
        }
    }

    private static String handleResponse(HttpResponse<String> resp) throws BaconError {
        int status = resp.statusCode();
        if (status >= 200 && status < 300) {
            return resp.body();
        }
        throw BaconError.fromJson(status, resp.body());
    }

    @SuppressWarnings("unchecked")
    private static Map<String, Object> contextMap(EvaluationContext ctx) {
        Map<String, Object> map = new LinkedHashMap<>();
        map.put("subject_id", ctx.getSubjectId());
        String env = ctx.getEnvironment();
        if (env != null && !env.isEmpty()) {
            map.put("environment", env);
        }
        if (!ctx.getAttributes().isEmpty()) {
            map.put("attributes", ctx.getAttributes());
        }
        return map;
    }

    // ── Builder ─────────────────────────────────────────────────────

    public static class Builder {
        private final String baseUrl;
        private String apiKey;
        private Duration timeout = Duration.ofSeconds(5);
        private HttpClient httpClient;

        private Builder(String baseUrl) {
            this.baseUrl = baseUrl;
        }

        public Builder apiKey(String apiKey) {
            this.apiKey = apiKey;
            return this;
        }

        public Builder timeout(Duration timeout) {
            this.timeout = timeout;
            return this;
        }

        /** Supply a pre-configured HttpClient (e.g. with custom SSL or proxy). */
        public Builder httpClient(HttpClient httpClient) {
            this.httpClient = httpClient;
            return this;
        }

        public BaconClient build() {
            return new BaconClient(this);
        }
    }
}
