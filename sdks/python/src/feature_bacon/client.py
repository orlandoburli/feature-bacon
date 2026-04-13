import json
import urllib.request
import urllib.error

from .errors import BaconError
from .types import EvaluationContext, EvaluationResult, HealthResponse


class BaconClient:
    def __init__(
        self,
        base_url: str,
        *,
        api_key: str | None = None,
        timeout: float = 5.0,
    ):
        self._base_url = base_url.rstrip("/")
        self._api_key = api_key
        self._timeout = timeout

    def evaluate(
        self, flag_key: str, context: EvaluationContext
    ) -> EvaluationResult:
        data = self._post(
            "/api/v1/evaluate",
            {"flagKey": flag_key, "context": context.to_dict()},
        )
        return EvaluationResult.from_dict(data)

    def evaluate_batch(
        self, flag_keys: list[str], context: EvaluationContext
    ) -> list[EvaluationResult]:
        data = self._post(
            "/api/v1/evaluate/batch",
            {"flagKeys": flag_keys, "context": context.to_dict()},
        )
        return [EvaluationResult.from_dict(r) for r in data.get("results", [])]

    def is_enabled(
        self, flag_key: str, context: EvaluationContext
    ) -> bool:
        try:
            return self.evaluate(flag_key, context).enabled
        except Exception:
            return False

    def get_variant(
        self, flag_key: str, context: EvaluationContext
    ) -> str:
        try:
            return self.evaluate(flag_key, context).variant
        except Exception:
            return ""

    def healthy(self) -> bool:
        try:
            data = self._get("/healthz")
            return data.get("status") == "ok"
        except Exception:
            return False

    def ready(self) -> HealthResponse:
        data = self._get("/readyz")
        return HealthResponse(
            status=data.get("status", ""),
            modules=data.get("modules", {}),
        )

    def _post(self, path: str, body: dict) -> dict:
        return self._request(path, method="POST", body=body)

    def _get(self, path: str) -> dict:
        return self._request(path, method="GET")

    def _request(
        self, path: str, *, method: str, body: dict | None = None
    ) -> dict:
        url = f"{self._base_url}{path}"
        headers = {"Content-Type": "application/json"}
        if self._api_key:
            headers["X-API-Key"] = self._api_key

        data = json.dumps(body).encode() if body else None
        req = urllib.request.Request(
            url, data=data, headers=headers, method=method
        )

        try:
            with urllib.request.urlopen(req, timeout=self._timeout) as resp:
                return json.loads(resp.read())
        except urllib.error.HTTPError as e:
            try:
                error_body = json.loads(e.read())
            except Exception:
                error_body = {}
            raise BaconError(
                status_code=e.code,
                type_=error_body.get("type", ""),
                title=error_body.get("title", f"HTTP {e.code}"),
                detail=error_body.get("detail", ""),
                instance=error_body.get("instance", path),
            ) from e
