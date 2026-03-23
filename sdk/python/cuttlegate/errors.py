"""Error types raised by CuttlegateClient.

All errors are typed — consumers can catch specific error classes rather
than inspecting string messages. No error includes the API key in its
string representation.
"""


class SDKError(Exception):
    """Base class for all Cuttlegate SDK errors."""


class ConfigError(ValueError):
    """Raised at construction time for invalid or missing configuration."""


class AuthError(SDKError):
    """Raised when the server responds with 401 or 403."""

    def __init__(self, status_code: int) -> None:
        self.status_code = status_code
        # Deliberately omits the api_key — only the status code is included.
        super().__init__(f"authentication failed: HTTP {status_code}")


class NotFoundError(SDKError):
    """Raised when the requested flag key does not exist in the project."""

    def __init__(self, key: str) -> None:
        self.key = key
        super().__init__(f"flag not found: {key!r}")


class ServerError(SDKError):
    """Raised when the server returns an unexpected 5xx status."""

    def __init__(self, status_code: int) -> None:
        self.status_code = status_code
        super().__init__(f"server error: HTTP {status_code}")
