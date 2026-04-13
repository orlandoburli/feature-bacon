package io.featurebacon;

import java.util.List;
import java.util.ArrayList;

/**
 * Immutable result of a single flag evaluation.
 */
public class EvaluationResult {
    private final String tenantId;
    private final String flagKey;
    private final boolean enabled;
    private final String variant;
    private final String reason;

    public EvaluationResult(String tenantId, String flagKey, boolean enabled, String variant, String reason) {
        this.tenantId = tenantId;
        this.flagKey = flagKey;
        this.enabled = enabled;
        this.variant = variant;
        this.reason = reason;
    }

    public String getTenantId() { return tenantId; }
    public String getFlagKey() { return flagKey; }
    public boolean isEnabled() { return enabled; }
    public String getVariant() { return variant; }
    public String getReason() { return reason; }

    static EvaluationResult fromJson(String json) {
        JsonHelper.JsonObject obj = JsonHelper.parseObject(json);
        return fromJsonObject(obj);
    }

    static EvaluationResult fromJsonObject(JsonHelper.JsonObject obj) {
        return new EvaluationResult(
                obj.getString("tenant_id", ""),
                obj.getString("flag_key", ""),
                obj.getBoolean("enabled", false),
                obj.getString("variant", ""),
                obj.getString("reason", "")
        );
    }

    static List<EvaluationResult> listFromJson(String json) {
        JsonHelper.JsonArray arr = JsonHelper.parseArray(json);
        List<EvaluationResult> results = new ArrayList<>();
        for (int i = 0; i < arr.size(); i++) {
            results.add(fromJsonObject(arr.getObject(i)));
        }
        return results;
    }

    @Override
    public String toString() {
        return "EvaluationResult{flagKey='" + flagKey + "', enabled=" + enabled
                + ", variant='" + variant + "', reason='" + reason + "'}";
    }
}
