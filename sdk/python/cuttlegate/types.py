"""Public types for the Cuttlegate Python SDK.

CuttlegateConfig, EvalContext, EvalResult are plain dataclasses.
CuttlegateClientProtocol is a PEP 544 structural protocol — consumers
can type-hint against it and write test doubles without subclassing.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Protocol


@dataclass
class CuttlegateConfig:
    """Configuration for a CuttlegateClient.

    api_key is marked repr=False so it never appears in log output or
    tracebacks. It is the caller's responsibility to source the key securely
    (environment variable, secrets manager, etc.) — this class does no
    config file I/O.
    """

    api_key: str = field(repr=False)
    server_url: str
    project: str
    environment: str
    timeout_ms: int = 10_000


@dataclass
class EvalContext:
    """Context for a flag evaluation request."""

    user_id: str
    attributes: dict[str, Any] = field(default_factory=dict)


@dataclass
class EvalResult:
    """Result of evaluating a single flag."""

    key: str
    enabled: bool
    value: str        # value_key from the wire — variant key for string flags, "true"/"false" for bool
    reason: str
    evaluated_at: str


class CuttlegateClientProtocol(Protocol):
    """PEP 544 structural protocol for Cuttlegate clients.

    Type-hint variables as CuttlegateClientProtocol rather than the concrete
    CuttlegateClient so test doubles and mocks are accepted without subclassing.
    """

    def bool(self, key: str, ctx: EvalContext) -> bool:
        """Return True if the flag variant is 'true'. Raises NotFoundError if absent."""
        ...

    def string(self, key: str, ctx: EvalContext) -> str:
        """Return the flag's string variant value. Raises NotFoundError if absent."""
        ...

    def evaluate(self, key: str, ctx: EvalContext) -> EvalResult:
        """Evaluate a single flag by key. Raises NotFoundError if absent."""
        ...

    def evaluate_all(self, ctx: EvalContext) -> dict[str, EvalResult]:
        """Evaluate all flags in one HTTP round trip."""
        ...
