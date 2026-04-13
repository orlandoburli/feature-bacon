package io.featurebacon;

import java.util.Collections;
import java.util.HashMap;
import java.util.LinkedHashMap;
import java.util.Map;

/**
 * Describes the subject and environment for a flag evaluation request.
 * Instances are immutable and created via the {@link Builder}.
 */
public class EvaluationContext {
    private final String subjectId;
    private final String environment;
    private final Map<String, Object> attributes;

    private EvaluationContext(Builder builder) {
        this.subjectId = builder.subjectId;
        this.environment = builder.environment;
        this.attributes = Collections.unmodifiableMap(new HashMap<>(builder.attributes));
    }

    public static Builder builder(String subjectId) {
        return new Builder(subjectId);
    }

    public String getSubjectId() { return subjectId; }
    public String getEnvironment() { return environment; }
    public Map<String, Object> getAttributes() { return attributes; }

    String toJson() {
        Map<String, Object> map = new LinkedHashMap<>();
        map.put("subject_id", subjectId);
        if (environment != null && !environment.isEmpty()) {
            map.put("environment", environment);
        }
        if (!attributes.isEmpty()) {
            map.put("attributes", attributes);
        }
        return JsonHelper.toJson(map);
    }

    public static class Builder {
        private final String subjectId;
        private String environment = "";
        private final Map<String, Object> attributes = new HashMap<>();

        private Builder(String subjectId) {
            this.subjectId = subjectId;
        }

        public Builder environment(String env) {
            this.environment = env;
            return this;
        }

        public Builder attribute(String key, Object value) {
            this.attributes.put(key, value);
            return this;
        }

        public EvaluationContext build() {
            return new EvaluationContext(this);
        }
    }
}
