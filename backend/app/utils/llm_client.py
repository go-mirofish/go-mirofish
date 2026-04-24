"""
"""

import json
import re
import time
from typing import Optional, Dict, Any, List
from openai import OpenAI, APIConnectionError, APITimeoutError, APIStatusError, InternalServerError, RateLimitError

from ..config import Config
from ..llm_provider import resolve_llm_api_key


class LLMUpstreamError(Exception):
    """Structured LLM upstream failure for API-layer error mapping."""

    def __init__(
        self,
        *,
        message: str,
        status_code: int = 500,
        error_type: str = "upstream_error",
        error_code: Optional[str] = None,
        retry_after: Optional[str] = None,
        details: Optional[Dict[str, Any]] = None
    ):
        super().__init__(message)
        self.message = message
        self.status_code = status_code
        self.error_type = error_type
        self.error_code = error_code
        self.retry_after = retry_after
        self.details = details or {}

    def to_dict(self) -> Dict[str, Any]:
        payload = {
            "message": self.message,
            "status_code": self.status_code,
            "type": self.error_type,
            "code": self.error_code,
            "retry_after": self.retry_after,
        }
        if self.details:
            payload["details"] = self.details
        return payload


class LLMClient:
    MAX_RETRY_ATTEMPTS = 4
    INITIAL_RETRY_DELAY_SECONDS = 2.0
    MAX_RETRY_DELAY_SECONDS = 20.0
    
    def __init__(
        self,
        api_key: Optional[str] = None,
        base_url: Optional[str] = None,
        model: Optional[str] = None
    ):
        self.base_url = base_url or Config.LLM_BASE_URL
        self.api_key = resolve_llm_api_key(api_key or Config.LLM_API_KEY, self.base_url)
        self.model = model or Config.LLM_MODEL_NAME
        
        if not self.api_key:
            raise ValueError("LLM_API_KEY is not configured")
        
        self.client = OpenAI(
            api_key=self.api_key,
            base_url=self.base_url
        )

    @staticmethod
    def _clean_text_response(content: Any) -> str:
        """Normalize provider output into a plain string before parsing."""
        if isinstance(content, str):
            text = content
        elif isinstance(content, list):
            parts: List[str] = []
            for item in content:
                if isinstance(item, str):
                    parts.append(item)
                elif isinstance(item, dict) and isinstance(item.get("text"), str):
                    parts.append(item["text"])
            text = "\n".join(part for part in parts if part).strip()
        elif content is None:
            text = ""
        else:
            text = str(content)

        text = text.replace("\ufeff", "").strip()
        text = re.sub(r'<think>[\s\S]*?</think>', '', text).strip()
        return text

    def _extract_upstream_error(self, exc: Exception) -> Optional[LLMUpstreamError]:
        response = getattr(exc, "response", None)
        status_code = getattr(exc, "status_code", None)
        retry_after = None
        response_text = ""
        payload: Dict[str, Any] = {}

        if response is not None:
            try:
                retry_after = response.headers.get("retry-after") or response.headers.get("x-ratelimit-reset-tokens")
            except Exception:
                retry_after = None
            try:
                payload = response.json()
            except Exception:
                payload = {}
            try:
                response_text = response.text or ""
            except Exception:
                response_text = ""

        error_obj = payload.get("error") if isinstance(payload, dict) else None
        error_message = None
        error_type = None
        error_code = None
        if isinstance(error_obj, dict):
            error_message = error_obj.get("message")
            error_type = error_obj.get("type")
            error_code = error_obj.get("code")

        if isinstance(exc, RateLimitError) or status_code == 429:
            return LLMUpstreamError(
                message=error_message or str(exc),
                status_code=429,
                error_type=error_type or "rate_limit_exceeded",
                error_code=error_code,
                retry_after=retry_after,
                details={"provider_response": payload or response_text},
            )

        if status_code and 400 <= status_code < 500:
            return LLMUpstreamError(
                message=error_message or str(exc),
                status_code=status_code,
                error_type=error_type or "upstream_client_error",
                error_code=error_code,
                retry_after=retry_after,
                details={"provider_response": payload or response_text},
            )

        return None

    def _create_completion(self, **kwargs):
        delay = self.INITIAL_RETRY_DELAY_SECONDS
        last_exception = None

        for attempt in range(self.MAX_RETRY_ATTEMPTS):
            try:
                return self.client.chat.completions.create(**kwargs)
            except (APIConnectionError, APITimeoutError, RateLimitError, InternalServerError) as exc:
                last_exception = exc
            except APIStatusError as exc:
                last_exception = exc
                response_text = ""
                if exc.response is not None:
                    try:
                        response_text = exc.response.text
                    except Exception:
                        response_text = ""

                status_code = getattr(exc, "status_code", None)
                is_retryable_status = status_code in {408, 409, 425, 429} or (status_code is not None and status_code >= 500)
                looks_like_transient_json_failure = "json_validate_failed" in response_text
                if not is_retryable_status and not looks_like_transient_json_failure:
                    upstream_error = self._extract_upstream_error(exc)
                    if upstream_error is not None:
                        raise upstream_error
                    raise

            if attempt == self.MAX_RETRY_ATTEMPTS - 1:
                upstream_error = self._extract_upstream_error(last_exception)
                if upstream_error is not None:
                    raise upstream_error
                raise last_exception

            sleep_for = min(delay, self.MAX_RETRY_DELAY_SECONDS)
            time.sleep(sleep_for)
            delay *= 2.0

        upstream_error = self._extract_upstream_error(last_exception)
        if upstream_error is not None:
            raise upstream_error
        raise last_exception
    
    def chat(
        self,
        messages: List[Dict[str, str]],
        temperature: float = 0.7,
        max_tokens: int = 4096,
        response_format: Optional[Dict] = None
    ) -> str:
        """
        
        Args:
            
        Returns:
        """
        kwargs = {
            "model": self.model,
            "messages": messages,
            "temperature": temperature,
            "max_tokens": max_tokens,
        }
        
        if response_format:
            kwargs["response_format"] = response_format
        
        response = self._create_completion(**kwargs)
        content = self._clean_text_response(response.choices[0].message.content)
        if not content:
            raise ValueError("LLM returned empty content")
        return content

    def _repair_json_text(self, broken_json: str) -> Dict[str, Any]:
        repair_messages = [
            {
                "role": "system",
                "content": (
                    "You repair malformed JSON. Return only valid compact JSON with the same intended structure. "
                    "Do not add markdown fences or explanations."
                ),
            },
            {
                "role": "user",
                "content": broken_json,
            },
        ]
        repaired = self.chat(
            messages=repair_messages,
            temperature=0,
            max_tokens=4096,
            response_format={"type": "json_object"},
        )
        repaired = repaired.strip()
        repaired = re.sub(r'^```(?:json)?\s*\n?', '', repaired, flags=re.IGNORECASE)
        repaired = re.sub(r'\n?```\s*$', '', repaired)
        return json.loads(repaired.strip())
    
    def chat_json(
        self,
        messages: List[Dict[str, str]],
        temperature: float = 0.3,
        max_tokens: int = 4096
    ) -> Dict[str, Any]:
        """
        
        Args:
            
        Returns:
        """
        response = self.chat(
            messages=messages,
            temperature=temperature,
            max_tokens=max_tokens,
            response_format={"type": "json_object"}
        )
        cleaned_response = self._clean_text_response(response)
        cleaned_response = re.sub(r'^```(?:json)?\s*\n?', '', cleaned_response, flags=re.IGNORECASE)
        cleaned_response = re.sub(r'\n?```\s*$', '', cleaned_response)
        cleaned_response = cleaned_response.strip()
        if not cleaned_response:
            raise ValueError("LLM returned empty JSON response")

        try:
            parsed = json.loads(cleaned_response)
        except json.JSONDecodeError:
            try:
                parsed = self._repair_json_text(cleaned_response)
            except Exception:
                raise ValueError(f"LLM returned invalid JSON: {cleaned_response}")

        if not isinstance(parsed, dict):
            raise ValueError(f"LLM returned JSON that was not an object: {type(parsed).__name__}")
        return parsed
