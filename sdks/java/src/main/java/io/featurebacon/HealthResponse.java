package io.featurebacon;

import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.Map;

/**
 * Response from the /healthz and /readyz endpoints.
 */
public class HealthResponse {
    private final String status;
    private final Map<String, ModuleHealth> modules;

    public HealthResponse(String status, Map<String, ModuleHealth> modules) {
        this.status = status;
        this.modules = Collections.unmodifiableMap(modules);
    }

    public String getStatus() { return status; }
    public Map<String, ModuleHealth> getModules() { return modules; }

    public boolean isHealthy() {
        return "healthy".equalsIgnoreCase(status);
    }

    static HealthResponse fromJson(String json) {
        JsonHelper.JsonObject obj = JsonHelper.parseObject(json);
        String status = obj.getString("status", "unknown");
        Map<String, ModuleHealth> modules = new LinkedHashMap<>();
        JsonHelper.JsonObject modsObj = obj.getObject("modules");
        for (Map.Entry<String, Object> entry : modsObj.rawMap().entrySet()) {
            if (entry.getValue() instanceof JsonHelper.JsonObject) {
                modules.put(entry.getKey(), ModuleHealth.fromJsonObject((JsonHelper.JsonObject) entry.getValue()));
            }
        }
        return new HealthResponse(status, modules);
    }

    @Override
    public String toString() {
        return "HealthResponse{status='" + status + "', modules=" + modules.keySet() + '}';
    }

    public static class ModuleHealth {
        private final String status;
        private final long latencyMs;
        private final String message;

        public ModuleHealth(String status, long latencyMs, String message) {
            this.status = status;
            this.latencyMs = latencyMs;
            this.message = message;
        }

        public String getStatus() { return status; }
        public long getLatencyMs() { return latencyMs; }
        public String getMessage() { return message; }

        static ModuleHealth fromJsonObject(JsonHelper.JsonObject obj) {
            return new ModuleHealth(
                    obj.getString("status", "unknown"),
                    obj.getLong("latency_ms", 0),
                    obj.getString("message", "")
            );
        }

        @Override
        public String toString() {
            return "ModuleHealth{status='" + status + "', latencyMs=" + latencyMs + '}';
        }
    }
}
