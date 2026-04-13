package io.featurebacon;

import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * Minimal JSON builder and parser — no external dependencies.
 * Handles the subset of JSON the Feature Bacon API produces/consumes.
 */
final class JsonHelper {

    private JsonHelper() {}

    // ── JSON writing ────────────────────────────────────────────────

    static String toJson(Map<String, Object> map) {
        StringBuilder sb = new StringBuilder();
        writeMap(sb, map);
        return sb.toString();
    }

    @SuppressWarnings("unchecked")
    private static void writeValue(StringBuilder sb, Object value) {
        if (value == null) {
            sb.append("null");
        } else if (value instanceof String) {
            writeString(sb, (String) value);
        } else if (value instanceof Boolean || value instanceof Number) {
            sb.append(value);
        } else if (value instanceof Map) {
            writeMap(sb, (Map<String, Object>) value);
        } else if (value instanceof List) {
            writeList(sb, (List<Object>) value);
        } else {
            writeString(sb, value.toString());
        }
    }

    private static void writeString(StringBuilder sb, String s) {
        sb.append('"');
        for (int i = 0; i < s.length(); i++) {
            char c = s.charAt(i);
            switch (c) {
                case '"':  sb.append("\\\""); break;
                case '\\': sb.append("\\\\"); break;
                case '\b': sb.append("\\b");  break;
                case '\f': sb.append("\\f");  break;
                case '\n': sb.append("\\n");  break;
                case '\r': sb.append("\\r");  break;
                case '\t': sb.append("\\t");  break;
                default:
                    if (c < 0x20) {
                        sb.append(String.format("\\u%04x", (int) c));
                    } else {
                        sb.append(c);
                    }
            }
        }
        sb.append('"');
    }

    private static void writeMap(StringBuilder sb, Map<String, Object> map) {
        sb.append('{');
        boolean first = true;
        for (Map.Entry<String, Object> e : map.entrySet()) {
            if (!first) sb.append(',');
            first = false;
            writeString(sb, e.getKey());
            sb.append(':');
            writeValue(sb, e.getValue());
        }
        sb.append('}');
    }

    private static void writeList(StringBuilder sb, List<Object> list) {
        sb.append('[');
        for (int i = 0; i < list.size(); i++) {
            if (i > 0) sb.append(',');
            writeValue(sb, list.get(i));
        }
        sb.append(']');
    }

    // ── JSON reading ────────────────────────────────────────────────

    static JsonObject parseObject(String json) {
        if (json == null || json.trim().isEmpty()) {
            return new JsonObject(new LinkedHashMap<>());
        }
        Parser p = new Parser(json.trim());
        Object val = p.parseValue();
        if (val instanceof JsonObject) return (JsonObject) val;
        return new JsonObject(new LinkedHashMap<>());
    }

    static JsonArray parseArray(String json) {
        if (json == null || json.trim().isEmpty()) {
            return new JsonArray(new ArrayList<>());
        }
        Parser p = new Parser(json.trim());
        Object val = p.parseValue();
        if (val instanceof JsonArray) return (JsonArray) val;
        return new JsonArray(new ArrayList<>());
    }

    // ── Lightweight wrappers ────────────────────────────────────────

    static final class JsonObject {
        private final Map<String, Object> map;

        JsonObject(Map<String, Object> map) { this.map = map; }

        String getString(String key, String defaultValue) {
            Object v = map.get(key);
            return v instanceof String ? (String) v : defaultValue;
        }

        boolean getBoolean(String key, boolean defaultValue) {
            Object v = map.get(key);
            return v instanceof Boolean ? (Boolean) v : defaultValue;
        }

        long getLong(String key, long defaultValue) {
            Object v = map.get(key);
            if (v instanceof Number) return ((Number) v).longValue();
            return defaultValue;
        }

        JsonObject getObject(String key) {
            Object v = map.get(key);
            return v instanceof JsonObject ? (JsonObject) v : new JsonObject(new LinkedHashMap<>());
        }

        JsonArray getArray(String key) {
            Object v = map.get(key);
            return v instanceof JsonArray ? (JsonArray) v : new JsonArray(new ArrayList<>());
        }

        boolean has(String key) { return map.containsKey(key); }

        Map<String, Object> rawMap() { return map; }
    }

    static final class JsonArray {
        private final List<Object> list;

        JsonArray(List<Object> list) { this.list = list; }

        int size() { return list.size(); }

        Object get(int i) { return list.get(i); }

        JsonObject getObject(int i) {
            Object v = list.get(i);
            return v instanceof JsonObject ? (JsonObject) v : new JsonObject(new LinkedHashMap<>());
        }
    }

    // ── Recursive-descent parser ────────────────────────────────────

    private static class Parser {
        private final String src;
        private int pos;

        Parser(String src) {
            this.src = src;
            this.pos = 0;
        }

        Object parseValue() {
            skipWhitespace();
            if (pos >= src.length()) return null;
            char c = src.charAt(pos);
            switch (c) {
                case '{': return parseObj();
                case '[': return parseArr();
                case '"': return parseStr();
                case 't': case 'f': return parseBool();
                case 'n': return parseNull();
                default:  return parseNumber();
            }
        }

        private JsonObject parseObj() {
            expect('{');
            Map<String, Object> map = new LinkedHashMap<>();
            skipWhitespace();
            if (peek() == '}') { pos++; return new JsonObject(map); }
            while (true) {
                skipWhitespace();
                String key = parseStr();
                skipWhitespace();
                expect(':');
                Object val = parseValue();
                map.put(key, val);
                skipWhitespace();
                if (peek() == ',') { pos++; } else { break; }
            }
            expect('}');
            return new JsonObject(map);
        }

        private JsonArray parseArr() {
            expect('[');
            List<Object> list = new ArrayList<>();
            skipWhitespace();
            if (peek() == ']') { pos++; return new JsonArray(list); }
            while (true) {
                list.add(parseValue());
                skipWhitespace();
                if (peek() == ',') { pos++; } else { break; }
            }
            expect(']');
            return new JsonArray(list);
        }

        private String parseStr() {
            expect('"');
            StringBuilder sb = new StringBuilder();
            while (pos < src.length()) {
                char c = src.charAt(pos++);
                if (c == '"') return sb.toString();
                if (c == '\\') {
                    if (pos >= src.length()) break;
                    char esc = src.charAt(pos++);
                    switch (esc) {
                        case '"':  sb.append('"');  break;
                        case '\\': sb.append('\\'); break;
                        case '/':  sb.append('/');  break;
                        case 'b':  sb.append('\b'); break;
                        case 'f':  sb.append('\f'); break;
                        case 'n':  sb.append('\n'); break;
                        case 'r':  sb.append('\r'); break;
                        case 't':  sb.append('\t'); break;
                        case 'u':
                            String hex = src.substring(pos, pos + 4);
                            sb.append((char) Integer.parseInt(hex, 16));
                            pos += 4;
                            break;
                        default: sb.append(esc);
                    }
                } else {
                    sb.append(c);
                }
            }
            return sb.toString();
        }

        private Boolean parseBool() {
            if (src.startsWith("true", pos))  { pos += 4; return Boolean.TRUE; }
            if (src.startsWith("false", pos)) { pos += 5; return Boolean.FALSE; }
            throw new IllegalStateException("Expected boolean at pos " + pos);
        }

        private Object parseNull() {
            if (src.startsWith("null", pos)) { pos += 4; return null; }
            throw new IllegalStateException("Expected null at pos " + pos);
        }

        private Number parseNumber() {
            int start = pos;
            if (peek() == '-') pos++;
            while (pos < src.length() && Character.isDigit(src.charAt(pos))) pos++;
            boolean isFloat = false;
            if (pos < src.length() && src.charAt(pos) == '.') {
                isFloat = true;
                pos++;
                while (pos < src.length() && Character.isDigit(src.charAt(pos))) pos++;
            }
            if (pos < src.length() && (src.charAt(pos) == 'e' || src.charAt(pos) == 'E')) {
                isFloat = true;
                pos++;
                if (pos < src.length() && (src.charAt(pos) == '+' || src.charAt(pos) == '-')) pos++;
                while (pos < src.length() && Character.isDigit(src.charAt(pos))) pos++;
            }
            String numStr = src.substring(start, pos);
            if (isFloat) return Double.parseDouble(numStr);
            long l = Long.parseLong(numStr);
            if (l >= Integer.MIN_VALUE && l <= Integer.MAX_VALUE) return (int) l;
            return l;
        }

        private void skipWhitespace() {
            while (pos < src.length() && Character.isWhitespace(src.charAt(pos))) pos++;
        }

        private char peek() {
            return pos < src.length() ? src.charAt(pos) : 0;
        }

        private void expect(char c) {
            skipWhitespace();
            if (pos >= src.length() || src.charAt(pos) != c) {
                throw new IllegalStateException("Expected '" + c + "' at pos " + pos);
            }
            pos++;
        }
    }
}
