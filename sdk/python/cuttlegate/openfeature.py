"""OpenFeature Provider for Cuttlegate.

Implements the OpenFeature ``AbstractProvider`` class using the
``openfeature-sdk`` package.

Usage::

    from openfeature import api as openfeature
    from cuttlegate import CuttlegateClient
    from cuttlegate.openfeature import CuttlegateProvider

    client = CuttlegateClient(config)
    openfeature.set_provider(CuttlegateProvider(client))

    of_client = openfeature.get_client()
    value = of_client.get_boolean_value("dark-mode", False)
"""

from __future__ import annotations

import json
import typing

from openfeature.evaluation_context import EvaluationContext
from openfeature.flag_evaluation import FlagResolutionDetails, Reason
from openfeature.provider import AbstractProvider, Metadata

from .client import CuttlegateClient
from .types import EvalContext


class CuttlegateProvider(AbstractProvider):
    """OpenFeature provider backed by a sync CuttlegateClient."""

    def __init__(self, client: CuttlegateClient) -> None:
        super().__init__()
        self._client = client

    def get_metadata(self) -> Metadata:
        return Metadata(name="cuttlegate")

    def _eval_context(self, context: typing.Optional[EvaluationContext] = None) -> EvalContext:
        if context is None:
            return EvalContext(user_id="")
        targeting_key = context.targeting_key or ""
        attrs = {k: v for k, v in (context.attributes or {}).items() if isinstance(v, str)}
        return EvalContext(user_id=str(targeting_key), attributes=attrs)

    def resolve_boolean_details(
        self,
        flag_key: str,
        default_value: bool,
        evaluation_context: typing.Optional[EvaluationContext] = None,
    ) -> FlagResolutionDetails[bool]:
        try:
            result = self._client.bool(flag_key, self._eval_context(evaluation_context))
            return FlagResolutionDetails(
                value=result,
                variant="true" if result else "false",
                reason=Reason.TARGETING_MATCH,
            )
        except Exception as exc:
            return FlagResolutionDetails(
                value=default_value,
                reason=Reason.ERROR,
                error_code="GENERAL",
                error_message=str(exc),
            )

    def resolve_string_details(
        self,
        flag_key: str,
        default_value: str,
        evaluation_context: typing.Optional[EvaluationContext] = None,
    ) -> FlagResolutionDetails[str]:
        try:
            result = self._client.string(flag_key, self._eval_context(evaluation_context))
            return FlagResolutionDetails(
                value=result,
                variant=result,
                reason=Reason.TARGETING_MATCH,
            )
        except Exception as exc:
            return FlagResolutionDetails(
                value=default_value,
                reason=Reason.ERROR,
                error_code="GENERAL",
                error_message=str(exc),
            )

    def resolve_integer_details(
        self,
        flag_key: str,
        default_value: int,
        evaluation_context: typing.Optional[EvaluationContext] = None,
    ) -> FlagResolutionDetails[int]:
        try:
            result = self._client.string(flag_key, self._eval_context(evaluation_context))
            return FlagResolutionDetails(
                value=int(result),
                variant=result,
                reason=Reason.TARGETING_MATCH,
            )
        except Exception as exc:
            return FlagResolutionDetails(
                value=default_value,
                reason=Reason.ERROR,
                error_code="GENERAL",
                error_message=str(exc),
            )

    def resolve_float_details(
        self,
        flag_key: str,
        default_value: float,
        evaluation_context: typing.Optional[EvaluationContext] = None,
    ) -> FlagResolutionDetails[float]:
        try:
            result = self._client.string(flag_key, self._eval_context(evaluation_context))
            return FlagResolutionDetails(
                value=float(result),
                variant=result,
                reason=Reason.TARGETING_MATCH,
            )
        except Exception as exc:
            return FlagResolutionDetails(
                value=default_value,
                reason=Reason.ERROR,
                error_code="GENERAL",
                error_message=str(exc),
            )

    def resolve_object_details(
        self,
        flag_key: str,
        default_value: typing.Any,
        evaluation_context: typing.Optional[EvaluationContext] = None,
    ) -> FlagResolutionDetails[typing.Any]:
        try:
            result = self._client.string(flag_key, self._eval_context(evaluation_context))
            return FlagResolutionDetails(
                value=json.loads(result),
                variant=result,
                reason=Reason.TARGETING_MATCH,
            )
        except Exception as exc:
            return FlagResolutionDetails(
                value=default_value,
                reason=Reason.ERROR,
                error_code="GENERAL",
                error_message=str(exc),
            )
