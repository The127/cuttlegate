"""Cuttlegate Python SDK.

Public exports:

    CuttlegateClient          — synchronous flag evaluation client
    MockCuttlegateClient      — in-memory test helper; no live server required
    CuttlegateClientProtocol  — PEP 544 protocol for type-safe consumer code
    CuttlegateConfig          — configuration dataclass (api_key suppressed in repr)
    EvalContext               — evaluation context (user_id + attributes)
    EvalResult                — single flag evaluation result

    ConfigError               — raised at construction for invalid config
    AuthError                 — raised on 401/403 from the server
    NotFoundError             — raised when a flag key is absent
    ServerError               — raised on 5xx from the server
    SDKError                  — base class for all SDK-specific errors
"""

from .client import CuttlegateClient
from .errors import AuthError, ConfigError, NotFoundError, SDKError, ServerError
from .testing import MockCuttlegateClient
from .types import CuttlegateClientProtocol, CuttlegateConfig, EvalContext, EvalResult

__all__ = [
    "CuttlegateClient",
    "MockCuttlegateClient",
    "CuttlegateClientProtocol",
    "CuttlegateConfig",
    "EvalContext",
    "EvalResult",
    "ConfigError",
    "AuthError",
    "NotFoundError",
    "ServerError",
    "SDKError",
]
