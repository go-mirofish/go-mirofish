"""
Helpers for OpenAI-compatible LLM provider configuration.
"""

from __future__ import annotations

from typing import Optional
from urllib.parse import urlparse


LOCAL_PROVIDER_HOSTS = {
    "",
    "127.0.0.1",
    "0.0.0.0",
    "localhost",
    "::1",
    "host.docker.internal",
}


def is_local_openai_compatible_base_url(base_url: Optional[str]) -> bool:
    if not base_url:
        return False
    parsed = urlparse(base_url)
    hostname = (parsed.hostname or "").strip().lower()
    return hostname in LOCAL_PROVIDER_HOSTS


def resolve_llm_api_key(api_key: Optional[str], base_url: Optional[str]) -> Optional[str]:
    if api_key:
        return api_key
    if is_local_openai_compatible_base_url(base_url):
        return "local-openai-compatible"
    return api_key


def llm_api_key_required(base_url: Optional[str]) -> bool:
    return not is_local_openai_compatible_base_url(base_url)
