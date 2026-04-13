from dataclasses import dataclass, field


@dataclass
class EvaluationContext:
    subject_id: str
    environment: str = ""
    attributes: dict | None = None

    def to_dict(self) -> dict:
        d: dict = {"subjectId": self.subject_id}
        if self.environment:
            d["environment"] = self.environment
        if self.attributes:
            d["attributes"] = self.attributes
        return d


@dataclass
class EvaluationResult:
    tenant_id: str
    flag_key: str
    enabled: bool
    variant: str
    reason: str

    @classmethod
    def from_dict(cls, data: dict) -> "EvaluationResult":
        return cls(
            tenant_id=data.get("tenantId", ""),
            flag_key=data.get("flagKey", ""),
            enabled=data.get("enabled", False),
            variant=data.get("variant", ""),
            reason=data.get("reason", ""),
        )


@dataclass
class HealthResponse:
    status: str
    modules: dict = field(default_factory=dict)
