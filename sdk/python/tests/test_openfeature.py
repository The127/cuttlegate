"""Tests for the CuttlegateProvider OpenFeature adapter."""

from __future__ import annotations

import pytest

from cuttlegate import MockCuttlegateClient
from cuttlegate.openfeature import CuttlegateProvider


def test_metadata():
    mock = MockCuttlegateClient(flags={"x": True})
    provider = CuttlegateProvider(mock)
    assert provider.get_metadata().name == "cuttlegate"


# ── Boolean ────────────────────────────────────────────────────────────────

def test_boolean_enabled():
    mock = MockCuttlegateClient(flags={"dark-mode": True})
    provider = CuttlegateProvider(mock)

    result = provider.resolve_boolean_details("dark-mode", False, {"targetingKey": "u1"})

    assert result.value is True
    assert result.variant == "true"
    assert result.reason == "TARGETING_MATCH"


def test_boolean_disabled():
    mock = MockCuttlegateClient(flags={"dark-mode": False})
    provider = CuttlegateProvider(mock)

    result = provider.resolve_boolean_details("dark-mode", True, {"targetingKey": "u1"})

    assert result.value is False
    assert result.variant == "false"


def test_boolean_missing_returns_false():
    """Mock returns enabled=False for unknown flags (not an error)."""
    mock = MockCuttlegateClient()
    provider = CuttlegateProvider(mock)

    result = provider.resolve_boolean_details("missing", True, {"targetingKey": "u1"})

    assert result.value is False
    assert result.reason == "TARGETING_MATCH"


# ── String ─────────────────────────────────────────────────────────────────

def test_string_returns_variant():
    mock = MockCuttlegateClient(flags={"color": "blue"})
    provider = CuttlegateProvider(mock)

    result = provider.resolve_string_details("color", "red", {"targetingKey": "u1"})

    assert result.value == "blue"
    assert result.variant == "blue"
    assert result.reason == "TARGETING_MATCH"


def test_string_missing_returns_empty():
    """Mock returns empty variant for unknown flags (not an error)."""
    mock = MockCuttlegateClient()
    provider = CuttlegateProvider(mock)

    result = provider.resolve_string_details("missing", "fallback", {"targetingKey": "u1"})

    assert result.value == ""
    assert result.reason == "TARGETING_MATCH"


# ── Integer ────────────────────────────────────────────────────────────────

def test_integer_returns_parsed_value():
    mock = MockCuttlegateClient(flags={"rate-limit": "42"})
    provider = CuttlegateProvider(mock)

    result = provider.resolve_integer_details("rate-limit", 10, {"targetingKey": "u1"})

    assert result.value == 42
    assert result.reason == "TARGETING_MATCH"


def test_integer_non_numeric_returns_default():
    mock = MockCuttlegateClient(flags={"rate-limit": "not-a-number"})
    provider = CuttlegateProvider(mock)

    result = provider.resolve_integer_details("rate-limit", 10, {"targetingKey": "u1"})

    assert result.value == 10
    assert result.reason == "ERROR"


# ── Float ──────────────────────────────────────────────────────────────────

def test_float_returns_parsed_value():
    mock = MockCuttlegateClient(flags={"ratio": "3.14"})
    provider = CuttlegateProvider(mock)

    result = provider.resolve_float_details("ratio", 1.0, {"targetingKey": "u1"})

    assert result.value == pytest.approx(3.14)
    assert result.reason == "TARGETING_MATCH"


# ── Object ─────────────────────────────────────────────────────────────────

def test_object_returns_parsed_json():
    mock = MockCuttlegateClient(flags={"config": '{"key": "value"}'})
    provider = CuttlegateProvider(mock)

    result = provider.resolve_object_details("config", {}, {"targetingKey": "u1"})

    assert result.value == {"key": "value"}
    assert result.reason == "TARGETING_MATCH"


def test_object_invalid_json_returns_default():
    mock = MockCuttlegateClient(flags={"config": "not-json"})
    provider = CuttlegateProvider(mock)

    result = provider.resolve_object_details("config", {"fallback": True}, {"targetingKey": "u1"})

    assert result.value == {"fallback": True}
    assert result.reason == "ERROR"


# ── Context mapping ────────────────────────────────────────────────────────

def test_context_maps_targeting_key():
    mock = MockCuttlegateClient(flags={"flag": True})
    provider = CuttlegateProvider(mock)

    result = provider.resolve_boolean_details("flag", False, {
        "targetingKey": "user-42",
        "plan": "pro",
    })

    assert result.value is True
    mock.assert_evaluated("flag")


def test_context_handles_none():
    mock = MockCuttlegateClient(flags={"flag": True})
    provider = CuttlegateProvider(mock)

    result = provider.resolve_boolean_details("flag", False)

    assert result.value is True
