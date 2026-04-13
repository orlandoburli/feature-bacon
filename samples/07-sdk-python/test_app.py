import pytest
from unittest.mock import patch, MagicMock
from app import app


@pytest.fixture
def test_client():
    app.config["TESTING"] = True
    app.config["WTF_CSRF_ENABLED"] = False
    with app.test_client() as c:
        yield c


def _mock_result(flag_key, enabled=True, variant=None, reason="RULE_MATCH"):
    r = MagicMock()
    r.flag_key = flag_key
    r.enabled = enabled
    r.variant = variant
    r.reason = reason
    return r


def test_health_healthy(test_client):
    with patch("app.client") as mock_client:
        mock_client.healthy.return_value = True
        resp = test_client.get("/health")
        assert resp.status_code == 200
        data = resp.get_json()
        assert data["status"] == "ok"
        assert data["baconHealthy"] is True


def test_health_unhealthy(test_client):
    with patch("app.client") as mock_client:
        mock_client.healthy.return_value = False
        resp = test_client.get("/health")
        assert resp.status_code == 503
        data = resp.get_json()
        assert data["status"] == "degraded"
        assert data["baconHealthy"] is False


def test_home_returns_features(test_client):
    results = [
        _mock_result("dark_mode", enabled=True, variant="on"),
        _mock_result("new_pricing", enabled=False),
        _mock_result("beta_features", enabled=True, variant="group_a"),
        _mock_result("checkout_redesign", enabled=True, variant="v2"),
        _mock_result("maintenance_mode", enabled=False),
    ]
    with patch("app.client") as mock_client:
        mock_client.evaluate_batch.return_value = results
        resp = test_client.get("/")
        assert resp.status_code == 200
        data = resp.get_json()
        assert data["service"] == "catalog-api"
        features = data["features"]
        assert "dark_mode" in features
        assert features["dark_mode"]["enabled"] is True
        assert features["dark_mode"]["variant"] == "on"
        assert features["new_pricing"]["enabled"] is False
        assert len(features) == 5


def test_home_default_user(test_client):
    with patch("app.client") as mock_client:
        mock_client.evaluate_batch.return_value = []
        resp = test_client.get("/")
        data = resp.get_json()
        assert data["user"] == "anonymous"


def test_home_with_user(test_client):
    with patch("app.client") as mock_client:
        mock_client.evaluate_batch.return_value = []
        resp = test_client.get("/?user=alice")
        data = resp.get_json()
        assert data["user"] == "alice"


def test_products_new_pricing_enabled(test_client):
    with patch("app.client") as mock_client:
        mock_client.is_enabled.return_value = True
        mock_client.get_variant.return_value = "v2"
        resp = test_client.get("/products")
        assert resp.status_code == 200
        data = resp.get_json()
        assert data["newPricingActive"] is True
        prices = {p["name"]: p["price"] for p in data["products"]}
        assert prices["Widget Pro"] == round(29.99 * 0.9, 2)
        assert prices["Widget Basic"] == round(9.99 * 0.9, 2)
        assert prices["Widget Enterprise"] == round(99.99 * 0.9, 2)


def test_products_new_pricing_disabled(test_client):
    with patch("app.client") as mock_client:
        mock_client.is_enabled.return_value = False
        mock_client.get_variant.return_value = None
        resp = test_client.get("/products")
        assert resp.status_code == 200
        data = resp.get_json()
        assert data["newPricingActive"] is False
        prices = {p["name"]: p["price"] for p in data["products"]}
        assert prices["Widget Pro"] == 29.99
        assert prices["Widget Basic"] == 9.99
        assert prices["Widget Enterprise"] == 99.99


def test_products_variant(test_client):
    with patch("app.client") as mock_client:
        mock_client.is_enabled.return_value = False
        mock_client.get_variant.return_value = "v3"
        resp = test_client.get("/products")
        data = resp.get_json()
        assert data["checkoutVariant"] == "v3"
