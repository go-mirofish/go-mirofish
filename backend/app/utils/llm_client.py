"""
"""

import json
import re
import time
import random
from typing import Optional, Dict, Any, List
from openai import OpenAI, APIConnectionError, APITimeoutError, APIStatusError, InternalServerError, RateLimitError

from ..config import Config
from ..llm_provider import resolve_llm_api_key


class LLMClient:
    
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

    def _create_completion(self, **kwargs):
        delay = 2.0
        last_exception = None

        for attempt in range(4):
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
                    raise

            if attempt == 3:
                raise last_exception

            sleep_for = min(delay, 20.0) * (0.5 + random.random())
            time.sleep(sleep_for)
            delay *= 2.0

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
        content = response.choices[0].message.content
        content = re.sub(r'<think>[\s\S]*?</think>', '', content).strip()
        return content
    
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
        cleaned_response = response.strip()
        cleaned_response = re.sub(r'^```(?:json)?\s*\n?', '', cleaned_response, flags=re.IGNORECASE)
        cleaned_response = re.sub(r'\n?```\s*$', '', cleaned_response)
        cleaned_response = cleaned_response.strip()

        try:
            return json.loads(cleaned_response)
        except json.JSONDecodeError:
            raise ValueError(f"LLM returned invalid JSON: {cleaned_response}")
