package io.featurebacon;

/**
 * Thrown when the Feature Bacon API returns an error response.
 * Follows RFC 7807 problem detail fields.
 */
public class BaconError extends Exception {
    private final int statusCode;
    private final String type;
    private final String title;
    private final String detail;
    private final String instance;

    public BaconError(int statusCode, String type, String title, String detail, String instance) {
        super(title != null && !title.isEmpty() ? title : "HTTP " + statusCode);
        this.statusCode = statusCode;
        this.type = type;
        this.title = title;
        this.detail = detail;
        this.instance = instance;
    }

    public BaconError(String message, Throwable cause) {
        super(message, cause);
        this.statusCode = 0;
        this.type = "";
        this.title = message;
        this.detail = cause != null ? cause.getMessage() : "";
        this.instance = "";
    }

    public int getStatusCode() { return statusCode; }
    public String getType() { return type; }
    public String getTitle() { return title; }
    public String getDetail() { return detail; }
    public String getInstance() { return instance; }

    static BaconError fromJson(int statusCode, String json) {
        JsonHelper.JsonObject obj = JsonHelper.parseObject(json);
        return new BaconError(
                statusCode,
                obj.getString("type", ""),
                obj.getString("title", ""),
                obj.getString("detail", ""),
                obj.getString("instance", "")
        );
    }

    @Override
    public String toString() {
        StringBuilder sb = new StringBuilder("BaconError{statusCode=");
        sb.append(statusCode);
        if (!title.isEmpty()) sb.append(", title='").append(title).append('\'');
        if (!detail.isEmpty()) sb.append(", detail='").append(detail).append('\'');
        sb.append('}');
        return sb.toString();
    }
}
