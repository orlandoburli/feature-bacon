package io.featurebacon;

import org.junit.Test;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

import static org.junit.Assert.*;

public class JsonHelperTest {

    // ── toJson ───────────────────────────────────────────────────────

    @Test
    public void testToJsonSimpleMap() {
        Map<String, Object> map = new LinkedHashMap<>();
        map.put("key", "value");
        map.put("num", 42);
        map.put("flag", true);

        String json = JsonHelper.toJson(map);
        assertEquals("{\"key\":\"value\",\"num\":42,\"flag\":true}", json);
    }

    @Test
    public void testToJsonNull() {
        Map<String, Object> map = new LinkedHashMap<>();
        map.put("n", null);

        assertEquals("{\"n\":null}", JsonHelper.toJson(map));
    }

    @Test
    public void testToJsonNestedMap() {
        Map<String, Object> inner = new LinkedHashMap<>();
        inner.put("a", 1);

        Map<String, Object> outer = new LinkedHashMap<>();
        outer.put("nested", inner);

        assertEquals("{\"nested\":{\"a\":1}}", JsonHelper.toJson(outer));
    }

    @Test
    public void testToJsonList() {
        List<Object> list = new ArrayList<>(Arrays.asList("x", "y"));
        Map<String, Object> map = new LinkedHashMap<>();
        map.put("items", list);

        assertEquals("{\"items\":[\"x\",\"y\"]}", JsonHelper.toJson(map));
    }

    @Test
    public void testToJsonStringEscaping() {
        Map<String, Object> map = new LinkedHashMap<>();
        map.put("s", "line1\nline2\ttab\"quote\\slash");

        String json = JsonHelper.toJson(map);
        assertTrue(json.contains("\\n"));
        assertTrue(json.contains("\\t"));
        assertTrue(json.contains("\\\""));
        assertTrue(json.contains("\\\\"));
    }

    @Test
    public void testToJsonControlCharEscaping() {
        Map<String, Object> map = new LinkedHashMap<>();
        map.put("s", "\b\f");

        String json = JsonHelper.toJson(map);
        assertTrue(json.contains("\\b"));
        assertTrue(json.contains("\\f"));
    }

    @Test
    public void testToJsonLowControlChar() {
        Map<String, Object> map = new LinkedHashMap<>();
        map.put("s", "\u0001");

        String json = JsonHelper.toJson(map);
        assertTrue(json.contains("\\u0001"));
    }

    @Test
    public void testToJsonNonStringObject() {
        Map<String, Object> map = new LinkedHashMap<>();
        map.put("obj", new Object() {
            @Override
            public String toString() {
                return "custom";
            }
        });

        assertTrue(JsonHelper.toJson(map).contains("\"custom\""));
    }

    @Test
    public void testToJsonEmptyList() {
        Map<String, Object> map = new LinkedHashMap<>();
        map.put("empty", new ArrayList<>());

        assertEquals("{\"empty\":[]}", JsonHelper.toJson(map));
    }

    @Test
    public void testToJsonDouble() {
        Map<String, Object> map = new LinkedHashMap<>();
        map.put("d", 3.14);

        assertTrue(JsonHelper.toJson(map).contains("3.14"));
    }

    // ── parseObject ──────────────────────────────────────────────────

    @Test
    public void testParseObjectSimple() {
        JsonHelper.JsonObject obj = JsonHelper.parseObject("{\"a\":\"b\",\"c\":true,\"d\":42}");

        assertEquals("b", obj.getString("a", ""));
        assertTrue(obj.getBoolean("c", false));
        assertEquals(42, obj.getLong("d", 0));
    }

    @Test
    public void testParseObjectNested() {
        JsonHelper.JsonObject obj = JsonHelper.parseObject("{\"inner\":{\"x\":1}}");

        JsonHelper.JsonObject inner = obj.getObject("inner");
        assertEquals(1, inner.getLong("x", 0));
    }

    @Test
    public void testParseObjectWithArray() {
        JsonHelper.JsonObject obj = JsonHelper.parseObject("{\"list\":[1,2,3]}");

        JsonHelper.JsonArray arr = obj.getArray("list");
        assertEquals(3, arr.size());
    }

    @Test
    public void testParseObjectNull() {
        JsonHelper.JsonObject obj = JsonHelper.parseObject(null);
        assertFalse(obj.has("anything"));
    }

    @Test
    public void testParseObjectEmpty() {
        JsonHelper.JsonObject obj = JsonHelper.parseObject("");
        assertFalse(obj.has("anything"));
    }

    @Test
    public void testParseObjectNonObject() {
        JsonHelper.JsonObject obj = JsonHelper.parseObject("[1,2]");
        assertFalse(obj.has("anything"));
    }

    @Test
    public void testParseObjectDefaults() {
        JsonHelper.JsonObject obj = JsonHelper.parseObject("{}");

        assertEquals("def", obj.getString("missing", "def"));
        assertFalse(obj.getBoolean("missing", false));
        assertEquals(99, obj.getLong("missing", 99));
        assertNotNull(obj.getObject("missing"));
        assertNotNull(obj.getArray("missing"));
    }

    @Test
    public void testParseObjectWithNull() {
        JsonHelper.JsonObject obj = JsonHelper.parseObject("{\"n\":null}");
        assertTrue(obj.has("n"));
        assertEquals("def", obj.getString("n", "def"));
    }

    // ── parseArray ───────────────────────────────────────────────────

    @Test
    public void testParseArray() {
        JsonHelper.JsonArray arr = JsonHelper.parseArray("[{\"a\":1},{\"a\":2}]");

        assertEquals(2, arr.size());
        assertEquals(1, arr.getObject(0).getLong("a", 0));
        assertEquals(2, arr.getObject(1).getLong("a", 0));
    }

    @Test
    public void testParseArrayNull() {
        JsonHelper.JsonArray arr = JsonHelper.parseArray(null);
        assertEquals(0, arr.size());
    }

    @Test
    public void testParseArrayEmpty() {
        JsonHelper.JsonArray arr = JsonHelper.parseArray("");
        assertEquals(0, arr.size());
    }

    @Test
    public void testParseArrayNonArray() {
        JsonHelper.JsonArray arr = JsonHelper.parseArray("{\"a\":1}");
        assertEquals(0, arr.size());
    }

    @Test
    public void testParseArrayGetRawElement() {
        JsonHelper.JsonArray arr = JsonHelper.parseArray("[42]");
        assertEquals(42, arr.get(0));
    }

    // ── number parsing ───────────────────────────────────────────────

    @Test
    public void testParseNegativeNumber() {
        JsonHelper.JsonObject obj = JsonHelper.parseObject("{\"n\":-10}");
        assertEquals(-10, obj.getLong("n", 0));
    }

    @Test
    public void testParseFloat() {
        JsonHelper.JsonObject obj = JsonHelper.parseObject("{\"f\":3.14}");
        assertEquals(3, obj.getLong("f", 0));
    }

    @Test
    public void testParseExponent() {
        JsonHelper.JsonObject obj = JsonHelper.parseObject("{\"e\":1e2}");
        assertEquals(100, obj.getLong("e", 0));
    }

    @Test
    public void testParseNegativeExponent() {
        JsonHelper.JsonObject obj = JsonHelper.parseObject("{\"e\":5E-1}");
        assertEquals(0, obj.getLong("e", 0));
    }

    @Test
    public void testParsePositiveExponent() {
        JsonHelper.JsonObject obj = JsonHelper.parseObject("{\"e\":1e+3}");
        assertEquals(1000, obj.getLong("e", 0));
    }

    @Test
    public void testParseLongNumber() {
        long big = 3_000_000_000L;
        JsonHelper.JsonObject obj = JsonHelper.parseObject("{\"l\":" + big + "}");
        assertEquals(big, obj.getLong("l", 0));
    }

    // ── string escape parsing ────────────────────────────────────────

    @Test
    public void testParseStringEscapes() {
        JsonHelper.JsonObject obj = JsonHelper.parseObject(
                "{\"s\":\"a\\\"b\\\\c\\/d\\be\\ff\\ng\\rh\\ti\"}");

        String s = obj.getString("s", "");
        assertTrue(s.contains("a\"b"));
        assertTrue(s.contains("\\c"));
        assertTrue(s.contains("/d"));
        assertTrue(s.contains("\b"));
        assertTrue(s.contains("\f"));
        assertTrue(s.contains("\n"));
        assertTrue(s.contains("\r"));
        assertTrue(s.contains("\t"));
    }

    @Test
    public void testParseUnicodeEscape() {
        JsonHelper.JsonObject obj = JsonHelper.parseObject("{\"s\":\"\\u0041\"}");
        assertEquals("A", obj.getString("s", ""));
    }

    // ── boolean / null parsing ───────────────────────────────────────

    @Test
    public void testParseBoolTrue() {
        JsonHelper.JsonObject obj = JsonHelper.parseObject("{\"b\":true}");
        assertTrue(obj.getBoolean("b", false));
    }

    @Test
    public void testParseBoolFalse() {
        JsonHelper.JsonObject obj = JsonHelper.parseObject("{\"b\":false}");
        assertFalse(obj.getBoolean("b", true));
    }

    // ── has ──────────────────────────────────────────────────────────

    @Test
    public void testHas() {
        JsonHelper.JsonObject obj = JsonHelper.parseObject("{\"present\":1}");
        assertTrue(obj.has("present"));
        assertFalse(obj.has("absent"));
    }

    // ── rawMap ───────────────────────────────────────────────────────

    @Test
    public void testRawMap() {
        JsonHelper.JsonObject obj = JsonHelper.parseObject("{\"k\":\"v\"}");
        Map<String, Object> raw = obj.rawMap();
        assertEquals("v", raw.get("k"));
    }
}
