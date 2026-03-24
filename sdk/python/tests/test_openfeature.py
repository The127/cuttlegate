"""Tests for the CuttlegateProvider OpenFeature adapter."""

from __future__ import annotations

import pytest

from openfeature import api as openfeature
from openfeature.evaluation_context import EvaluationContext
from openfeature.provider import AbstractProvider

from cuttlegate import MockCuttlegateClient
from cuttlegate.openfeature import CuttlegateProvider


def test_is_abstract_provider_subclass():
    mock = MockCuttlegateClient(flags={"x": True})
    provider = CuttlegateProvider(mock)
    assert isinstance(provider, AbstractProvider)


def test_metadata():
    mock = MockCuttlegateClient(flags={"x": True})
    provider = CuttlegateProvider(mock)
    assert provider.get_metadata().name == "cuttlegate"


# ── Boolean ────────────────────────────────────────────────────────────────

def test_boolean_enabled():
    mock = MockCuttlegateClient(flags={"dark-mode": True})
    provider = CuttlegateProvider(mock)
    ctx = EvaluationContext(targeting_key="u1")

    result = provider.resolve_boolean_details("dark-mode", False, ctx)

    assert result.value is True
    assert result.variant == "true"
    assert result.reason.value == "TARGETING_MATCH"


def test_boolean_disabled():
    mock = MockCuttlegateClient(flags={"dark-mode": False})
    provider = CuttlegateProvider(mock)
    ctx = EvaluationContext(targeting_key="u1")

    result = provider.resolve_boolean_details("dark-mode", True, ctx)

    assert result.value is False
    assert result.variant == "false"


# ── String ─────────────────────────────────────────────────────────────────

def test_string_returns_variant():
    mock = MockCuttlegateClient(flags={"color": "blue"})
    provider = CuttlegateProvider(mock)
    ctx = EvaluationContext(targeting_key="u1")

    result = provider.resolve_string_details("color", "red", ctx)

    assert result.value == "blue"
    assert result.variant == "blue"
    assert result.reason.value == "TARGETING_MATCH"


# ── Integer ────────────────────────────────────────────────────────────────

def test_integer_returns_parsed_value():
    mock = MockCuttlegateClient(flags={"rate-limit": "42"})
    provider = CuttlegateProvider(mock)
    ctx = EvaluationContext(targeting_key="u1")

    result = provider.resolve_integer_details("rate-limit", 10, ctx)

    assert result.value == 42


def test_integer_non_numeric_returns_default():
    mock = MockCuttlegateClient(flags={"rate-limit": "not-a-number"})
    provider = CuttlegateProvider(mock)
    ctx = EvaluationContext(targeting_key="u1")

    result = provider.resolve_integer_details("rate-limit", 10, ctx)

    assert result.value == 10
    assert result.reason.value == "ERROR"


# ── Float ──────────────────────────────────────────────────────────────────

def test_float_returns_parsed_value():
    mock = MockCuttlegateClient(flags={"ratio": "3.14"})
    provider = CuttlegateProvider(mock)
    ctx = EvaluationContext(targeting_key="u1")

    result = provider.resolve_float_details("ratio", 1.0, ctx)

    assert result.value == pytest.approx(3.14)


# ── Object ─────────────────────────────────────────────────────────────────

def test_object_returns_parsed_json():
    mock = MockCuttlegateClient(flags={"config": '{"key": "value"}'})
    provider = CuttlegateProvider(mock)
    ctx = EvaluationContext(targeting_key="u1")

    result = provider.resolve_object_details("config", {}, ctx)

    assert result.value == {"key": "value"}


def test_object_invalid_json_returns_default():
    mock = MockCuttlegateClient(flags={"config": "not-json"})
    provider = CuttlegateProvider(mock)
    ctx = EvaluationContext(targeting_key="u1")

    result = provider.resolve_object_details("config", {"fallback": True}, ctx)

    assert result.value == {"fallback": True}
    assert result.reason.value == "ERROR"


# ── OpenFeature SDK conformance ────────────────────────────────────────────

def test_conformance_boolean():
    """Register provider with the real OpenFeature API and resolve through it."""
    mock = MockCuttlegateClient(flags={"feature-x": True})
    provider = CuttlegateProvider(mock)

    openfeature.set_provider(provider)
    client = openfeature.get_client()

    value = client.get_boolean_value("feature-x", False, EvaluationContext(targeting_key="u1"))
    assert value is True

    openfeature.shutdown()


def test_conformance_string():
    mock = MockCuttlegateClient(flags={"theme": "dark"})
    provider = CuttlegateProvider(mock)

    openfeature.set_provider(provider)
    client = openfeature.get_client()

    value = client.get_string_value("theme", "light", EvaluationContext(targeting_key="u1"))
    assert value == "dark"

    openfeature.shutdown()


# ── Context mapping ────────────────────────────────────────────────────────

def test_context_maps_targeting_key():
    mock = MockCuttlegateClient(flags={"flag": True})
    provider = CuttlegateProvider(mock)
    ctx = EvaluationContext(targeting_key="user-42", attributes={"plan": "pro"})

    result = provider.resolve_boolean_details("flag", False, ctx)

    assert result.value is True
    mock.assert_evaluated("flag")


def test_context_handles_none():
    mock = MockCuttlegateClient(flags={"flag": True})
    provider = CuttlegateProvider(mock)

    result = provider.resolve_boolean_details("flag", False)

    assert result.value is True
