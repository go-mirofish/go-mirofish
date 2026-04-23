/**
 * Build ChatGPT / Claude "new chat" URLs with a prefilled prompt (parity with legacy PageActions).
 */
export function buildLlmPrompt(t, pageUrl) {
  return t('home.pageActionsOpenInLlmPrompt', { url: pageUrl })
}

export function chatgptUrlFromPrompt(prompt) {
  return `https://chat.openai.com/?q=${encodeURIComponent(prompt)}`
}

export function claudeUrlFromPrompt(prompt) {
  return `https://claude.ai/new?q=${encodeURIComponent(prompt)}`
}
