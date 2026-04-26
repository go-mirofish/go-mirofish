# Ollama (local LLM)

Use this when you want to run **LLM inference on your own machine** instead of a hosted API. You still need a **Zep Cloud** key for graph memory unless your deployment replaces that separately.

## What Ollama does here

[Ollama](https://ollama.com/) serves an **OpenAI-compatible HTTP API** on your computer. go-mirofish’s backend talks to LLMs through an OpenAI-style **`base_url` + `api_key` + `model`** interface, so pointing those at Ollama is usually enough for the **LLM** side.

## Install Ollama

Follow the official install steps for your OS: [https://ollama.com/download](https://ollama.com/download)

Pull a model you will use (example):

```bash
ollama pull llama3.1
```

Start the Ollama server if it is not already running (the desktop app or `ollama serve`).

## Configure `.env`

From the repo root:

```bash
cp .env.example .env
```

Set **LLM** variables to match Ollama’s OpenAI-compatible endpoint. Typical local settings:

```env
LLM_BASE_URL=http://127.0.0.1:11434/v1
LLM_MODEL_NAME=llama3.1
```

- **`LLM_API_KEY`:** optional for local Ollama if the server accepts unauthenticated requests.  
- **`LLM_BASE_URL`:** must be the **`/v1`** compatible base (see Ollama docs for your version).  
- **`LLM_MODEL_NAME`:** must match a model you **`ollama pull`**'d.

Leave **`ZEP_API_KEY`** set to a real Zep Cloud key unless you are using a different memory stack.

## Copy-paste block (local machine)

```env
LLM_BASE_URL=http://127.0.0.1:11434/v1
LLM_MODEL_NAME=llama3.1

ZEP_API_KEY=your_zep_api_key_here
```

Then start the app using [Installation](../getting-started/installation.md) (Docker or `npm run dev`).

## JSON mode and model caveats

Parts of the go-mirofish stack rely on **structured JSON** from the model in some code paths. Not every local model enforces JSON output reliably.

> [!WARNING]
> If simulations or graph steps fail with parse errors, try a model known for strong instruction-following / JSON mode, reduce scenario complexity, or switch to a hosted model that supports JSON-style responses consistently.

## Zep still required

> [!IMPORTANT]
> This guide only replaces the **LLM API**. **Zep Cloud** (`ZEP_API_KEY`) is still required for the default graph-memory workflow unless you change the backend integration.

## See also

- [Installation](../getting-started/installation.md)  
- [OpenAI-compatible providers](./providers.md)  
- Cloud LLM and full `.env` reference (expanding on [go-mirofish.vercel.app](https://go-mirofish.vercel.app) under **Configuration**; go.mirofish.ai when the custom domain is live)
