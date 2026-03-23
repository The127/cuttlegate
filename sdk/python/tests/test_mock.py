"""Unit tests for MockCuttlegateClient — all 15 BDD scenarios from grooming."""

from __future__ import annotations

import pytest

from cuttlegate import MockCuttlegateClient, NotFoundError
from cuttlegate.types import EvalContext


_CTX = EvalContext(user_id="u1")


# ---------------------------------------------------------------------------
# @happy — bool() returns True for a configured bool flag
# ---------------------------------------------------------------------------

def test_happy_bool_returns_true_for_bool_true_flag():
    """@happy: bool() returns True when stored value is True."""
    client = MockCuttlegateClient(flags={"my-flag": True})
    assert client.bool("my-flag", _CTX) is True


# ---------------------------------------------------------------------------
# @happy — bool() returns False for a flag set to False
# ---------------------------------------------------------------------------

def test_happy_bool_returns_false_for_bool_false_flag():
    """@happy: bool() returns False when stored value is False."""
    client = MockCuttlegateClient(flags={"my-flag": False})
    assert client.bool("my-flag", _CTX) is False


# ---------------------------------------------------------------------------
# @happy — bool() returns True for a flag set to the string "true"
# ---------------------------------------------------------------------------

def test_happy_bool_returns_true_for_string_true_flag():
    """@happy: bool() returns True when stored value is the string "true"."""
    client = MockCuttlegateClient(flags={"my-flag": "true"})
    assert client.bool("my-flag", _CTX) is True


# ---------------------------------------------------------------------------
# @happy — string() returns the configured string value
# ---------------------------------------------------------------------------

def test_happy_string_returns_configured_value():
    """@happy: string() returns the stored string value."""
    client = MockCuttlegateClient(flags={"color": "blue"})
    assert client.string("color", _CTX) == "blue"


# ---------------------------------------------------------------------------
# @happy — evaluate() returns EvalResult matching the configured value
# ---------------------------------------------------------------------------

def test_happy_evaluate_returns_eval_result():
    """@happy: evaluate() returns EvalResult with correct fields for a bool True flag."""
    client = MockCuttlegateClient(flags={"my-flag": True})
    result = client.evaluate("my-flag", _CTX)
    assert result.key == "my-flag"
    assert result.enabled is True
    assert result.value == "true"
    assert result.reason == "mock"
    assert result.evaluated_at == "1970-01-01T00:00:00Z"


# ---------------------------------------------------------------------------
# @happy — evaluate_all() returns all configured flags
# ---------------------------------------------------------------------------

def test_happy_evaluate_all_returns_all_flags():
    """@happy: evaluate_all() returns an EvalResult for each configured flag."""
    client = MockCuttlegateClient(flags={"flag-a": True, "flag-b": "red"})
    results = client.evaluate_all(_CTX)
    assert set(results.keys()) == {"flag-a", "flag-b"}
    assert results["flag-a"].enabled is True
    assert results["flag-b"].value == "red"


# ---------------------------------------------------------------------------
# @happy — set_flag mutation is reflected in subsequent calls
# ---------------------------------------------------------------------------

def test_happy_set_flag_mutation_reflected_in_calls():
    """@happy: set_flag updates are visible to subsequent bool() calls."""
    client = MockCuttlegateClient()
    client.set_flag("my-flag", False)
    assert client.bool("my-flag", _CTX) is False
    client.set_flag("my-flag", True)
    assert client.bool("my-flag", _CTX) is True


# ---------------------------------------------------------------------------
# @edge — unknown key raises NotFoundError on bool()
# ---------------------------------------------------------------------------

def test_edge_bool_unknown_key_raises_not_found_error():
    """@edge: bool() raises NotFoundError for an absent key; error message contains the key."""
    client = MockCuttlegateClient()
    with pytest.raises(NotFoundError) as exc_info:
        client.bool("absent-flag", _CTX)
    assert "absent-flag" in str(exc_info.value)


# ---------------------------------------------------------------------------
# @edge — unknown key raises NotFoundError on string()
# ---------------------------------------------------------------------------

def test_edge_string_unknown_key_raises_not_found_error():
    """@edge: string() raises NotFoundError for an absent key."""
    client = MockCuttlegateClient()
    with pytest.raises(NotFoundError):
        client.string("absent-flag", _CTX)


# ---------------------------------------------------------------------------
# @edge — unknown key raises NotFoundError on evaluate()
# ---------------------------------------------------------------------------

def test_edge_evaluate_unknown_key_raises_not_found_error():
    """@edge: evaluate() raises NotFoundError for an absent key."""
    client = MockCuttlegateClient()
    with pytest.raises(NotFoundError):
        client.evaluate("absent-flag", _CTX)


# ---------------------------------------------------------------------------
# @edge — evaluate_all() on empty mock returns empty dict (does NOT raise)
# ---------------------------------------------------------------------------

def test_edge_evaluate_all_empty_mock_returns_empty_dict():
    """@edge: evaluate_all() on a mock with no flags returns {} without raising."""
    client = MockCuttlegateClient()
    result = client.evaluate_all(_CTX)
    assert result == {}


# ---------------------------------------------------------------------------
# @happy — assert_evaluated passes after the flag is evaluated
# ---------------------------------------------------------------------------

def test_happy_assert_evaluated_passes_after_evaluation():
    """@happy: assert_evaluated() does not raise when the flag was evaluated."""
    client = MockCuttlegateClient(flags={"my-flag": True})
    client.bool("my-flag", _CTX)
    client.assert_evaluated("my-flag")  # must not raise


# ---------------------------------------------------------------------------
# @edge — assert_evaluated raises AssertionError if flag was not evaluated
# ---------------------------------------------------------------------------

def test_edge_assert_evaluated_raises_if_not_evaluated():
    """@edge: assert_evaluated() raises AssertionError if the flag was never evaluated;
    error message mentions the flag key."""
    client = MockCuttlegateClient(flags={"my-flag": True})
    with pytest.raises(AssertionError) as exc_info:
        client.assert_evaluated("my-flag")
    assert "my-flag" in str(exc_info.value)


# ---------------------------------------------------------------------------
# @edge — assert_not_evaluated raises AssertionError if flag was evaluated
# ---------------------------------------------------------------------------

def test_edge_assert_not_evaluated_raises_if_was_evaluated():
    """@edge: assert_not_evaluated() raises AssertionError if the flag was evaluated."""
    client = MockCuttlegateClient(flags={"my-flag": True})
    client.bool("my-flag", _CTX)
    with pytest.raises(AssertionError):
        client.assert_not_evaluated("my-flag")


# ---------------------------------------------------------------------------
# @happy — reset() clears flag state and evaluation history
# ---------------------------------------------------------------------------

def test_happy_reset_clears_flag_state_and_evaluation_history():
    """@happy: reset() makes previously-set flags absent and clears evaluated tracking."""
    client = MockCuttlegateClient(flags={"my-flag": True})
    client.bool("my-flag", _CTX)
    client.reset()

    with pytest.raises(NotFoundError):
        client.evaluate("my-flag", _CTX)

    with pytest.raises(AssertionError):
        client.assert_evaluated("my-flag")
