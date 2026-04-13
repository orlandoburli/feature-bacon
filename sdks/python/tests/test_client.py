import io
import json
import unittest
import urllib.error
from unittest.mock import MagicMock, patch

from feature_bacon import (
    BaconClient,
    BaconError,
    EvaluationContext,
    EvaluationResult,
    HealthResponse,
)


def _mock_response(body: dict, status: int = 200) -> MagicMock:
    resp = MagicMock()
    resp.read.return_value = json.dumps(body).encode()
    resp.status = status
    resp.__enter__ = lambda s: s
    resp.__exit__ = MagicMock(return_value=False)
    return resp


class TestEvaluationContext(unittest.TestCase):
    def test_to_dict_minimal(self):
        ctx = EvaluationContext(subject_id="u1")
        assert ctx.to_dict() == {"subjectId": "u1"}

    def test_to_dict_full(self):
        ctx = EvaluationContext(
            subject_id="u1",
            environment="prod",
            attributes={"plan": "pro"},
        )
        assert ctx.to_dict() == {
            "subjectId": "u1",
            "environment": "prod",
            "attributes": {"plan": "pro"},
        }

    def test_to_dict_empty_environment_excluded(self):
        ctx = EvaluationContext(subject_id="u1", attributes={"k": "v"})
        d = ctx.to_dict()
        assert "environment" not in d
        assert d["attributes"] == {"k": "v"}


class TestEvaluationResult(unittest.TestCase):
    def test_from_dict(self):
        r = EvaluationResult.from_dict(
            {
                "tenantId": "t1",
                "flagKey": "flag-a",
                "enabled": True,
                "variant": "blue",
                "reason": "match",
            }
        )
        assert r.tenant_id == "t1"
        assert r.flag_key == "flag-a"
        assert r.enabled is True
        assert r.variant == "blue"
        assert r.reason == "match"

    def test_from_dict_defaults(self):
        r = EvaluationResult.from_dict({})
        assert r.tenant_id == ""
        assert r.enabled is False
        assert r.variant == ""


class TestBaconClient(unittest.TestCase):
    def setUp(self):
        self.client = BaconClient(
            "http://localhost:8080", api_key="test-key"
        )
        self.ctx = EvaluationContext(
            subject_id="user_1", environment="production"
        )

    @patch("feature_bacon.client.urllib.request.urlopen")
    def test_evaluate_success(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response(
            {
                "tenantId": "t1",
                "flagKey": "dark-mode",
                "enabled": True,
                "variant": "on",
                "reason": "targeted",
            }
        )

        result = self.client.evaluate("dark-mode", self.ctx)

        assert isinstance(result, EvaluationResult)
        assert result.flag_key == "dark-mode"
        assert result.enabled is True
        assert result.variant == "on"
        assert result.reason == "targeted"

    @patch("feature_bacon.client.urllib.request.urlopen")
    def test_evaluate_sends_correct_payload(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response(
            {"flagKey": "f", "enabled": False, "variant": "", "reason": ""}
        )
        self.client.evaluate("my-flag", self.ctx)

        req = mock_urlopen.call_args[0][0]
        body = json.loads(req.data)
        assert body["flagKey"] == "my-flag"
        assert body["context"]["subjectId"] == "user_1"
        assert body["context"]["environment"] == "production"
        assert req.get_header("X-api-key") == "test-key"
        assert req.get_header("Content-type") == "application/json"

    @patch("feature_bacon.client.urllib.request.urlopen")
    def test_evaluate_batch(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response(
            {
                "results": [
                    {"flagKey": "a", "enabled": True, "variant": "v1", "reason": "r"},
                    {"flagKey": "b", "enabled": False, "variant": "", "reason": "off"},
                ]
            }
        )

        results = self.client.evaluate_batch(["a", "b"], self.ctx)

        assert len(results) == 2
        assert results[0].flag_key == "a"
        assert results[0].enabled is True
        assert results[1].flag_key == "b"
        assert results[1].enabled is False

    @patch("feature_bacon.client.urllib.request.urlopen")
    def test_evaluate_batch_empty(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"results": []})
        results = self.client.evaluate_batch([], self.ctx)
        assert results == []

    @patch("feature_bacon.client.urllib.request.urlopen")
    def test_is_enabled_true(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response(
            {"flagKey": "f", "enabled": True, "variant": "", "reason": ""}
        )
        assert self.client.is_enabled("f", self.ctx) is True

    @patch("feature_bacon.client.urllib.request.urlopen")
    def test_is_enabled_false_on_error(self, mock_urlopen):
        mock_urlopen.side_effect = urllib.error.URLError("connection refused")
        assert self.client.is_enabled("f", self.ctx) is False

    @patch("feature_bacon.client.urllib.request.urlopen")
    def test_get_variant(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response(
            {"flagKey": "f", "enabled": True, "variant": "blue", "reason": ""}
        )
        assert self.client.get_variant("f", self.ctx) == "blue"

    @patch("feature_bacon.client.urllib.request.urlopen")
    def test_get_variant_empty_on_error(self, mock_urlopen):
        mock_urlopen.side_effect = urllib.error.URLError("timeout")
        assert self.client.get_variant("f", self.ctx) == ""

    @patch("feature_bacon.client.urllib.request.urlopen")
    def test_healthy_true(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"status": "ok"})
        assert self.client.healthy() is True

    @patch("feature_bacon.client.urllib.request.urlopen")
    def test_healthy_false_on_bad_status(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"status": "degraded"})
        assert self.client.healthy() is False

    @patch("feature_bacon.client.urllib.request.urlopen")
    def test_healthy_false_on_error(self, mock_urlopen):
        mock_urlopen.side_effect = urllib.error.URLError("down")
        assert self.client.healthy() is False

    @patch("feature_bacon.client.urllib.request.urlopen")
    def test_ready(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response(
            {"status": "ready", "modules": {"db": "ok", "cache": "ok"}}
        )
        resp = self.client.ready()
        assert isinstance(resp, HealthResponse)
        assert resp.status == "ready"
        assert resp.modules == {"db": "ok", "cache": "ok"}

    @patch("feature_bacon.client.urllib.request.urlopen")
    def test_http_error_raises_bacon_error(self, mock_urlopen):
        error_body = json.dumps(
            {
                "type": "about:blank",
                "title": "Unauthorized",
                "detail": "Invalid API key",
                "instance": "/api/v1/evaluate",
            }
        ).encode()
        mock_urlopen.side_effect = urllib.error.HTTPError(
            url="http://localhost:8080/api/v1/evaluate",
            code=401,
            msg="Unauthorized",
            hdrs={},
            fp=io.BytesIO(error_body),
        )

        with self.assertRaises(BaconError) as cm:
            self.client.evaluate("f", self.ctx)

        err = cm.exception
        assert err.status_code == 401
        assert err.type == "about:blank"
        assert err.title == "Unauthorized"
        assert err.detail == "Invalid API key"
        assert err.instance == "/api/v1/evaluate"

    @patch("feature_bacon.client.urllib.request.urlopen")
    def test_http_error_with_unreadable_body(self, mock_urlopen):
        mock_urlopen.side_effect = urllib.error.HTTPError(
            url="http://localhost:8080/api/v1/evaluate",
            code=500,
            msg="Internal Server Error",
            hdrs={},
            fp=io.BytesIO(b"not json"),
        )

        with self.assertRaises(BaconError) as cm:
            self.client.evaluate("f", self.ctx)

        err = cm.exception
        assert err.status_code == 500
        assert err.title == "HTTP 500"
        assert err.detail == ""

    def test_base_url_trailing_slash_stripped(self):
        client = BaconClient("http://example.com/")
        assert client._base_url == "http://example.com"

    @patch("feature_bacon.client.urllib.request.urlopen")
    def test_no_api_key_omits_header(self, mock_urlopen):
        mock_urlopen.return_value = _mock_response({"status": "ok"})
        client = BaconClient("http://localhost:8080")
        client.healthy()

        req = mock_urlopen.call_args[0][0]
        assert not req.has_header("X-api-key")


if __name__ == "__main__":
    unittest.main()
