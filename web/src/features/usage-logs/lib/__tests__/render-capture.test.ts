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
import { describe, expect, it } from 'vitest'

import { capturedBodyCopyText, renderCapturedBody } from '../render-capture'

describe('renderCapturedBody', () => {
  it('renders request messages with their roles', () => {
    const sections = renderCapturedBody(
      '{"messages":[{"role":"user","content":"# Hello"}]}',
      'request'
    )

    expect(sections).toEqual([
      { kind: 'message', role: 'user', content: '# Hello' },
    ])
  })

  it('renders an OpenAI-style response as assistant content', () => {
    const sections = renderCapturedBody(
      '{"choices":[{"message":{"role":"assistant","content":"**Done**"}}]}',
      'response'
    )

    expect(sections).toEqual([
      { kind: 'message', role: 'assistant', content: '**Done**' },
    ])
  })

  it('combines streamed response deltas before Markdown rendering', () => {
    const body = [
      'data: {"choices":[{"delta":{"content":"# Hel"}}]}',
      '',
      'data: {"choices":[{"delta":{"content":"lo"}}]}',
      '',
      'data: [DONE]',
    ].join('\n')

    expect(renderCapturedBody(body, 'response')).toEqual([
      { kind: 'message', role: 'assistant', content: '# Hello' },
    ])
  })

  it('renders a plain response body as Markdown', () => {
    expect(
      renderCapturedBody('## Upstream note\n\n**Important**', 'response')
    ).toEqual([
      { kind: 'markdown', content: '## Upstream note\n\n**Important**' },
    ])
  })

  it('preserves large integer lexemes in JSON fallback content', () => {
    const sections = renderCapturedBody('{"id":1234567890123456789}', 'request')

    expect(sections[0]?.content).toContain('1234567890123456789')
    expect(sections[0]?.content).not.toContain('1234567890123456800')
  })

  it('renders error responses as an error message', () => {
    expect(
      renderCapturedBody('{"error":{"message":"Upstream failed"}}', 'response')
    ).toEqual([{ kind: 'message', role: 'Error', content: 'Upstream failed' }])
  })

  it('returns raw text for an empty or unstructured body', () => {
    expect(renderCapturedBody('   ', 'response')).toEqual([])
    expect(renderCapturedBody('upstream failed', 'response')).toEqual([
      { kind: 'markdown', content: 'upstream failed' },
    ])
  })

  it('builds copy text from rendered sections', () => {
    expect(
      capturedBodyCopyText([
        { kind: 'message', role: 'user', content: 'Hello' },
        { kind: 'markdown', content: '**World**' },
      ])
    ).toBe('user:\nHello\n\n**World**')
  })
})
