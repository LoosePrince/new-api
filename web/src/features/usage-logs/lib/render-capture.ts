/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

const INDENT = '  '

export type CapturedBodyKind = 'request' | 'response'

export type CapturedBodySection = {
  kind: 'markdown' | 'json' | 'message'
  role?: string
  content: string
}

type ParsedCapture = {
  eventStream: boolean
  values: unknown[]
}

export function renderCapturedBody(
  body: string,
  bodyKind: CapturedBodyKind
): CapturedBodySection[] {
  const trimmed = body.trim()
  if (!trimmed) return []

  const parsed = parseCapturedValues(trimmed)
  const messages = parsed.values.flatMap((value) =>
    extractMessages(value, bodyKind)
  )
  if (messages.length > 0) {
    return bodyKind === 'response' && parsed.eventStream
      ? mergeAdjacentMessages(messages)
      : messages
  }

  const errorMessages = parsed.values
    .map((value) => getNestedText(value, ['error', 'message']))
    .filter((value): value is string => Boolean(value))
  if (errorMessages.length > 0) {
    return errorMessages.map((content) => ({
      kind: 'message',
      role: 'Error',
      content,
    }))
  }

  if (!parsed.eventStream) {
    const prettyJSON = prettifyJson(trimmed)
    if (prettyJSON != null) return [{ kind: 'json', content: prettyJSON }]
  }

  const markdown = parsed.values
    .map(getMarkdownPayload)
    .filter((value): value is string => value != null && value.trim() !== '')
  if (markdown.length > 0) {
    return markdown.map((content) => ({ kind: 'markdown', content }))
  }

  const prettyJSON = parsed.values
    .map((value) => (typeof value === 'string' ? prettifyJson(value) : null))
    .filter((value): value is string => value != null)
  if (prettyJSON.length > 0) {
    return prettyJSON.map((content) => ({ kind: 'json', content }))
  }
  return [{ kind: 'markdown', content: trimmed }]
}

export function capturedBodyCopyText(sections: CapturedBodySection[]): string {
  return sections
    .map((section) => {
      if (section.kind === 'message' && section.role) {
        return `${section.role}:\n${section.content}`
      }
      return section.content
    })
    .join('\n\n')
}

function parseCapturedValues(body: string): ParsedCapture {
  const values: unknown[] = []
  const dataLines = body
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line.startsWith('data:'))
    .map((line) => line.slice('data:'.length).trim())
    .filter((line) => line !== '' && line !== '[DONE]')

  if (dataLines.length > 0) {
    for (const line of dataLines) {
      const parsed = parseJSON(line)
      values.push(parsed.ok ? parsed.value : line)
    }
    return { eventStream: true, values }
  }

  const parsed = parseJSON(body)
  return parsed.ok
    ? { eventStream: false, values: [parsed.value] }
    : { eventStream: false, values: [body] }
}

function extractMessages(
  value: unknown,
  bodyKind: CapturedBodyKind
): CapturedBodySection[] {
  if (!isRecord(value)) return []
  const messages: CapturedBodySection[] = []

  if (bodyKind === 'request') {
    const system = getText(value.system) ?? getText(value.instructions)
    if (system) {
      messages.push({ kind: 'message', role: 'system', content: system })
    }
    let entries: unknown[] = []
    if (Array.isArray(value.messages)) {
      entries = value.messages
    } else if (Array.isArray(value.input)) {
      entries = value.input
    } else if (Array.isArray(value.contents)) {
      entries = value.contents
    }
    for (const entry of entries) {
      if (!isRecord(entry)) continue
      const role = getText(entry.role) ?? 'user'
      const content = contentText(entry.content ?? entry.parts ?? entry.text)
      if (content) messages.push({ kind: 'message', role, content })
    }
    if (typeof value.input === 'string' && value.input.trim()) {
      messages.push({ kind: 'message', role: 'user', content: value.input })
    }
    return messages
  }

  if (Array.isArray(value.choices)) {
    for (const choice of value.choices) {
      if (!isRecord(choice)) continue
      let message = choice
      if (isRecord(choice.message)) {
        message = choice.message
      } else if (isRecord(choice.delta)) {
        message = choice.delta
      }
      const role = getText(message.role) ?? 'assistant'
      const content = contentText(message.content ?? message.text)
      if (content) messages.push({ kind: 'message', role, content })
      const toolCalls = Array.isArray(message.tool_calls)
        ? message.tool_calls
        : []
      for (const toolCall of toolCalls) {
        const toolContent = contentText(toolCall)
        if (toolContent) {
          messages.push({ kind: 'message', role: 'tool', content: toolContent })
        }
      }
    }
  }

  const outputText = getText(value.output_text)
  if (outputText) {
    messages.push({ kind: 'message', role: 'assistant', content: outputText })
  }

  if (Array.isArray(value.output)) {
    for (const item of value.output) {
      if (!isRecord(item)) continue
      const role = getText(item.role) ?? 'assistant'
      const content = contentText(item.content ?? item.text)
      if (content) messages.push({ kind: 'message', role, content })
    }
  }

  if (Array.isArray(value.candidates)) {
    for (const candidate of value.candidates) {
      if (!isRecord(candidate) || !isRecord(candidate.content)) continue
      const role = getText(candidate.content.role) ?? 'assistant'
      const content = contentText(candidate.content.parts)
      if (content) messages.push({ kind: 'message', role, content })
    }
  }

  if (messages.length === 0 && Array.isArray(value.content)) {
    const content = contentText(value.content)
    if (content) messages.push({ kind: 'message', role: 'assistant', content })
  }
  return messages
}

function mergeAdjacentMessages(
  sections: CapturedBodySection[]
): CapturedBodySection[] {
  const merged: CapturedBodySection[] = []
  for (const section of sections) {
    const previous = merged.at(-1)
    if (
      previous?.kind === 'message' &&
      section.kind === 'message' &&
      previous.role === section.role
    ) {
      previous.content += section.content
    } else {
      merged.push({ ...section })
    }
  }
  return merged
}

function contentText(value: unknown): string | null {
  if (typeof value === 'string') return value.trim() ? value : null
  if (Array.isArray(value)) {
    const parts: string[] = []
    for (const item of value) {
      if (typeof item === 'string') {
        if (item.trim()) parts.push(item)
        continue
      }
      if (!isRecord(item)) continue
      const text = getText(item.text) ?? getText(item.content)
      if (text) {
        parts.push(text)
      } else if (item.type === 'image' || item.type === 'image_url') {
        parts.push('_[image content]_')
      } else if (
        item.type === 'tool_use' ||
        item.type === 'tool_result' ||
        item.type === 'function_call'
      ) {
        const name = getText(item.name) ?? getText(item.type) ?? 'tool'
        const input = item.input ?? item.arguments ?? item.result
        const serialized =
          typeof input === 'string' ? input : JSON.stringify(input ?? '')
        parts.push(`**${name}**\n\n\`\`\`json\n${serialized}\n\`\`\``)
      }
    }
    return parts.length > 0 ? parts.join('\n\n') : null
  }
  if (isRecord(value)) {
    return getText(value.text) ?? getText(value.content)
  }
  return null
}

function getMarkdownPayload(value: unknown): string | null {
  if (typeof value === 'string') return value
  if (!isRecord(value)) return null
  return (
    getText(value.text) ??
    getText(value.message) ??
    getText(value.content) ??
    getText(value.output_text)
  )
}

function getNestedText(value: unknown, path: string[]): string | null {
  let current = value
  for (const key of path) {
    if (!isRecord(current)) return null
    current = current[key]
  }
  return getText(current)
}

function getText(value: unknown): string | null {
  return typeof value === 'string' && value.trim() ? value : null
}

function parseJSON(
  value: string
): { ok: true; value: unknown } | { ok: false } {
  try {
    return { ok: true, value: JSON.parse(value) as unknown }
  } catch {
    return { ok: false }
  }
}

function prettifyJson(value: string): string | null {
  try {
    JSON.parse(value)
  } catch {
    return null
  }
  const output: string[] = []
  let indent = 0
  let index = 0
  while (index < value.length) {
    const char = value[index]
    if (char === '"') {
      const end = stringEnd(value, index)
      output.push(value.slice(index, end))
      index = end
      continue
    }
    if (char === '{' || char === '[') {
      const close = char === '{' ? '}' : ']'
      const next = nextSignificantIndex(value, index + 1)
      if (value[next] === close) {
        output.push(char + close)
        index = next + 1
        continue
      }
      indent++
      output.push(char, '\n', INDENT.repeat(indent))
      index++
      continue
    }
    if (char === '}' || char === ']') {
      indent = Math.max(0, indent - 1)
      output.push('\n', INDENT.repeat(indent), char)
      index++
      continue
    }
    if (char === ',') {
      output.push(',', '\n', INDENT.repeat(indent))
      index++
      continue
    }
    if (char === ':') {
      output.push(': ')
      index++
      continue
    }
    if (/\s/.test(char)) {
      index++
      continue
    }
    output.push(char)
    index++
  }
  return output.join('')
}

function stringEnd(value: string, start: number): number {
  let escaped = false
  for (let index = start + 1; index < value.length; index++) {
    const char = value[index]
    if (escaped) {
      escaped = false
    } else if (char === '\\') {
      escaped = true
    } else if (char === '"') {
      return index + 1
    }
  }
  return value.length
}

function nextSignificantIndex(value: string, start: number): number {
  let index = start
  while (index < value.length && /\s/.test(value[index])) index++
  return index
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value != null && typeof value === 'object' && !Array.isArray(value)
}
