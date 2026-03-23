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
    """Raised when the server responds with 401 or 403.

    ``status_code`` is 401 or 403. The message contains only the HTTP
    status text — the API key is never included.
    """

    def __init__(self, status_code: int) -> None:
        self.status_code = status_code
        # Deliberately omits the api_key — only the status code is included.
        super().__init__(f"authentication failed: HTTP {status_code}")


class FlagNotFoundError(SDKError):
    """Raised when the requested flag key is absent from a 200 response.

    Distinct from a project/environment 404 (which raises SDKError).
    ``key`` is the flag key that was not found.
    """

    def __init__(self, key: str) -> None:
        self.key = key
        super().__init__(f"flag not found: {key!r}")


# Backward-compatible alias — prefer FlagNotFoundError in new code.
NotFoundError = FlagNotFoundError


class ServerError(SDKError):
    """Raised when the server returns an unexpected 5xx status.

    Message format: ``"cuttlegate: server error {status_code}"``
    """

    def __init__(self, status_code: int) -> None:
        self.status_code = status_code
        super().__init__(f"cuttlegate: server error {status_code}")


class InvalidResponseError(SDKError):
    """Raised when a response body cannot be parsed or has an unexpected shape.

    Used by the streaming client when a ``data:`` line contains malformed JSON
    or an SSE event cannot be decoded. Non-fatal in streaming context — the
    stream continues after calling ``on_error``.

    ``message`` describes the parse failure. No request/response body is
    included to avoid inadvertent logging of sensitive content.
    """

    def __init__(self, message: str) -> None:
        self.message = message
        super().__init__(f"cuttlegate: invalid response: {message}")
