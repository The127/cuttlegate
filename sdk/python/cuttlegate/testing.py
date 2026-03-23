"""In-process mock test helper for the Cuttlegate Python SDK.

MockCuttlegateClient is a pure in-memory implementation of
CuttlegateClientProtocol. It ships as part of the cuttlegate distribution so
consumers can import it in their test suites without a live server.

Usage::

    from cuttlegate.testing import MockCuttlegateClient
    from cuttlegate.types import EvalContext

    client = MockCuttlegateClient(flags={"dark-mode": True, "theme": "blue"})
    assert client.bool("dark-mode", EvalContext(user_id="u1")) is True
    assert client.string("theme", EvalContext(user_id="u1")) == "blue"
    client.assert_evaluated("dark-mode")

No pytest, unittest, or other test-framework imports are used in this module.
"""

from __future__ import annotations

from typing import Any

from .errors import NotFoundError
from .types import EvalContext, EvalResult

_MOCK_EVALUATED_AT = "1970-01-01T00:00:00Z"


class MockCuttlegateClient:
    """In-memory implementation of CuttlegateClientProtocol for unit tests.

    Structurally satisfies CuttlegateClientProtocol via PEP 544 duck typing —
    no explicit subclassing required.
    """

    def __init__(self, flags: dict[str, Any] | None = None) -> None:
        """Initialise with an optional pre-set flags dict.

        Keys are flag keys; values are bool or str (e.g. True, False, "blue").
        The dict is copied so callers cannot mutate mock state via the original
        reference.
        """
        self._flags: dict[str, Any] = dict(flags) if flags is not None else {}
        self._evaluated: set[str] = set()

    # ------------------------------------------------------------------
    # CuttlegateClientProtocol methods
    # ------------------------------------------------------------------

    def bool(self, key: str, ctx: EvalContext) -> bool:
        """Return True if stored value is True or "true". Raises NotFoundError for unknown keys."""
        result = self._require(key)
        self._evaluated.add(key)
        return result.enabled

    def string(self, key: str, ctx: EvalContext) -> str:
        """Return str(stored_value). Raises NotFoundError for unknown keys."""
        result = self._require(key)
        self._evaluated.add(key)
        return result.value

    def evaluate(self, key: str, ctx: EvalContext) -> EvalResult:
        """Return EvalResult with reason='mock'. Raises NotFoundError for unknown keys."""
        result = self._require(key)
        self._evaluated.add(key)
        return result

    def evaluate_all(self, ctx: EvalContext) -> dict[str, EvalResult]:
        """Return EvalResult for every configured flag. Returns {} if no flags configured."""
        results: dict[str, EvalResult] = {}
        for key in self._flags:
            results[key] = _to_eval_result(key, self._flags[key])
            self._evaluated.add(key)
        return results

    # ------------------------------------------------------------------
    # Test-time mutation
    # ------------------------------------------------------------------

    def set_flag(self, key: str, value: Any) -> None:
        """Set or update a flag value. value is typically bool or str."""
        self._flags[key] = value

    # ------------------------------------------------------------------
    # Assertion helpers
    # ------------------------------------------------------------------

    def assert_evaluated(self, key: str) -> None:
        """Raise AssertionError if the flag was not evaluated during the test."""
        if key not in self._evaluated:
            raise AssertionError(
                f"expected flag {key!r} to have been evaluated, but it was not"
            )

    def assert_not_evaluated(self, key: str) -> None:
        """Raise AssertionError if the flag was evaluated during the test."""
        if key in self._evaluated:
            raise AssertionError(
                f"expected flag {key!r} not to have been evaluated, but it was"
            )

    def reset(self) -> None:
        """Clear all flag state and evaluation history. Use between test cases."""
        self._flags.clear()
        self._evaluated.clear()

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _require(self, key: str) -> EvalResult:
        """Look up key and return its EvalResult, or raise NotFoundError."""
        if key not in self._flags:
            raise NotFoundError(key)
        return _to_eval_result(key, self._flags[key])


# ------------------------------------------------------------------
# Module-private helpers
# ------------------------------------------------------------------

def _to_eval_result(key: str, stored_value: Any) -> EvalResult:
    """Convert a stored flag value into an EvalResult.

    Bool handling: Python's str(True) is "True" (capital T), not "true".
    We normalise explicitly so the mock's value field matches the wire format
    used by the real client ("true"/"false" for bool flags).
    """
    enabled = stored_value is True or stored_value == "true"

    if isinstance(stored_value, bool):
        value = "true" if stored_value else "false"
    else:
        value = str(stored_value)

    return EvalResult(
        key=key,
        enabled=enabled,
        value=value,
        reason="mock",
        evaluated_at=_MOCK_EVALUATED_AT,
    )
