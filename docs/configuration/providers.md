# OpenAI-Compatible Providers

go-mirofish configures the LLM with three variables, passed through the OpenAI SDK and a configurable `base_url`. That single pattern covers most vendors that expose an **OpenAI-compatible** HTTP API.

> [!NOTE]
> **Shape of config:** set `LLM_API_KEY`, `LLM_BASE_URL`, and `LLM_MODEL_NAME` in `.env` (see root `.env.example`). The backend does not use separate SDKs per brand, only this endpoint + model tuple.

| Variable | Purpose |
| --- | --- |
| `LLM_API_KEY` | API key the remote (or proxy) expects **when required** |
| `LLM_BASE_URL` | Base URL for the **OpenAI-compatible** `/v1/...` surface (include `/v1` where the vendor docs say to) |
| `LLM_MODEL_NAME` | Model id as the server understands it (provider-specific string) |

> [!IMPORTANT]
> Treat `LLM_API_KEY` (and any cloud keys in `.env`) as **secrets**. Do not commit `.env`; copy from `.env.example` and keep keys out of git history and screenshots.

## Providers (quick reference)

Use these as **examples**. Model names and URLs change when vendors update their docs.

| Kind | Provider | `LLM_BASE_URL` (example) | `LLM_MODEL_NAME` (example) |
| --- | --- | --- | --- |
| Local | **Ollama** | `http://127.0.0.1:11434/v1` | `llama3.1` |
| Local | **llama.cpp** (OpenAI-compatible server) | `http://127.0.0.1:8080/v1` | your model alias |
| Hosted | **OpenAI** | `https://api.openai.com/v1` | `gpt-4o-mini` |
| Hosted | **xAI / Grok** | `https://api.x.ai/v1` | `grok-2-latest` |
| Hosted | **Groq** | `https://api.groq.com/openai/v1` | `llama-3.1-70b-versatile` |
| Hosted | **Qwen (DashScope compatible mode)** | `https://dashscope.aliyuncs.com/compatible-mode/v1` | `qwen-plus` |

> [!NOTE]
> **Local runtimes (Ollama, llama.cpp on localhost):** `LLM_API_KEY` can be **omitted** when `LLM_BASE_URL` points at `127.0.0.1` or `localhost`. The backend treats the key as optional for that case.

> [!TIP]
> If a row above is close but not exact, check the provider’s “OpenAI-compatible” or “Chat Completions” base URL in their current docs, then set `LLM_MODEL_NAME` to the id they list for that endpoint.

## Copy-paste examples

### Local: Ollama

```env
LLM_BASE_URL=http://127.0.0.1:11434/v1
LLM_MODEL_NAME=llama3.1
```

### Local: llama.cpp server

Start the OpenAI-compatible server (example):

```bash
./server -m /path/to/model.gguf --host 127.0.0.1 --port 8080
```

Then in `.env`:

```env
LLM_BASE_URL=http://127.0.0.1:8080/v1
LLM_MODEL_NAME=your-model-alias
```

### Hosted: OpenAI

```env
LLM_API_KEY=<your OpenAI key>
LLM_BASE_URL=https://api.openai.com/v1
LLM_MODEL_NAME=gpt-4o-mini
```

### Hosted: xAI / Grok

```env
LLM_API_KEY=<your xAI key>
LLM_BASE_URL=https://api.x.ai/v1
LLM_MODEL_NAME=grok-2-latest
```

### Hosted: Groq

```env
LLM_API_KEY=<your Groq key>
LLM_BASE_URL=https://api.groq.com/openai/v1
LLM_MODEL_NAME=llama-3.1-70b-versatile
```

### Hosted: Qwen / DashScope

```env
LLM_API_KEY=<your DashScope key>
LLM_BASE_URL=https://dashscope.aliyuncs.com/compatible-mode/v1
LLM_MODEL_NAME=qwen-plus
```

> [!WARNING]
> **Gemini and Anthropic:** this repo does **not** include native Google Gemini or Anthropic SDK wiring. To use those model families, put an **OpenAI-compatible bridge or proxy** in front, then set `LLM_API_KEY` / `LLM_BASE_URL` / `LLM_MODEL_NAME` to whatever that gateway expects:

```env
LLM_API_KEY=<bridge key>
LLM_BASE_URL=<bridge /v1 endpoint>
LLM_MODEL_NAME=<bridge model name>
```

> [!NOTE]
> **Why one interface:** a single “OpenAI-compatible” path keeps the codebase small and still lets you swap OpenAI, Grok, Groq, Qwen, local servers, and gateway-backed Gemini/Anthropic by configuration alone.

## See also

- [Installation](../getting-started/installation.md)
- [Ollama](./ollama.md)
