/**
 * Must match the heading id algorithm used in DocsMarkdown and DocsView.extractToc.
 */
export function docSlugify(text) {
  return String(text || '')
    .trim()
    .toLowerCase()
    .replace(/[^\w\s-]/g, '')
    .replace(/\s+/g, '-')
    .slice(0, 80)
}

export function plainHeadingText(token) {
  if (!token) return ''
  if (typeof token.text === 'string' && token.text) return token.text
  if (typeof token.raw === 'string') return token.raw.replace(/^#{1,6}\s+/, '').replace(/\s+#+\s*$/, '').trim()
  return ''
}
