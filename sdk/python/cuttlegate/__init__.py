"""Cuttlegate Python SDK.

Public exports:

    CuttlegateClient          — synchronous flag evaluation client
    CachedClient              — in-memory flag cache backed by a single SSE connection
    MockCuttlegateClient      — in-memory test helper; no live server required
    CuttlegateClientProtocol  — PEP 544 protocol for type-safe consumer code
    CuttlegateConfig          — configuration dataclass (api_key suppressed in repr)
    EvalContext               — evaluation context (user_id + attributes)
    EvalResult                — single flag evaluation result

    connect_stream            — start SSE flag state stream (background daemon thread)
    StreamConnection          — handle returned by connect_stream; call close() to stop
    FlagChangeEvent           — dataclass for flag.state_changed SSE events

    ConfigError               — raised at construction for invalid config
    AuthError                 — raised on 401/403 from the server
    FlagNotFoundError         — raised when a flag key is absent from a 200 response
    NotFoundError             — alias for FlagNotFoundError (deprecated name)
    ServerError               — raised on 5xx from the server
    InvalidResponseError      — raised on malformed JSON or unexpected SSE event shape
    SDKError                  — base class for all SDK-specific errors
"""

from .async_client import AsyncCuttlegateClient
from .cached import CachedClient
from .client import CuttlegateClient
from .errors import (
    AuthError,
    ConfigError,
    FlagNotFoundError,
    InvalidResponseError,
    NotFoundError,
    SDKError,
    ServerError,
)
from .streaming import FlagChangeEvent, StreamConnection, connect_stream
from .openfeature import CuttlegateProvider
from .testing import MockCuttlegateClient
from .types import AsyncCuttlegateClientProtocol, CuttlegateClientProtocol, CuttlegateConfig, EvalContext, EvalResult

__all__ = [
    "AsyncCuttlegateClient",
    "AsyncCuttlegateClientProtocol",
    "CachedClient",
    "CuttlegateClient",
    "CuttlegateProvider",
    "MockCuttlegateClient",
    "CuttlegateClientProtocol",
    "CuttlegateConfig",
    "EvalContext",
    "EvalResult",
    "connect_stream",
    "StreamConnection",
    "FlagChangeEvent",
    "ConfigError",
    "AuthError",
    "FlagNotFoundError",
    "NotFoundError",
    "ServerError",
    "InvalidResponseError",
    "SDKError",
]
