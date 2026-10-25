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
*/
import { MessageCircle, Send, Sparkles } from 'lucide-react'
import { useRef, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { useStatus } from '@/hooks/use-status'
import { handleServerError } from '@/lib/handle-server-error'

import { sendPublicChatMessage } from '../../api'

type ChatMessage = {
  id: number
  role: 'user' | 'assistant'
  content: string
  model?: string
}

export function PublicChat() {
  const { t } = useTranslation()
  const { status } = useStatus()
  const [input, setInput] = useState('')
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [isSending, setIsSending] = useState(false)
  const [streamingMessageId, setStreamingMessageId] = useState<number | null>(
    null
  )
  const messageId = useRef(0)
  const enabled = status?.public_chat_enabled === true
  const endpoint =
    typeof status?.public_chat_endpoint === 'string'
      ? status.public_chat_endpoint
      : '/'
  const group =
    typeof status?.public_chat_group === 'string'
      ? status.public_chat_group
      : ''
  const routingMode =
    status?.public_chat_routing_mode === 'model' ? 'model' : 'group'
  const model =
    typeof status?.public_chat_model === 'string'
      ? status.public_chat_model
      : ''

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const content = input.trim()
    if (!content || isSending) return

    const nextMessages = [
      ...messages,
      { id: ++messageId.current, role: 'user' as const, content },
    ]
    const assistantId = ++messageId.current
    const assistantMessage: ChatMessage = {
      id: assistantId,
      role: 'assistant',
      content: '',
    }
    setMessages([...nextMessages, assistantMessage])
    setInput('')
    setIsSending(true)
    setStreamingMessageId(assistantId)

    let assistantContent = ''
    let assistantModel = ''
    try {
      await sendPublicChatMessage(
        endpoint,
        group,
        routingMode,
        model,
        nextMessages.map(({ role, content: messageContent }) => ({
          role,
          content: messageContent,
        })),
        (chunk) => {
          if (chunk.content) assistantContent += chunk.content
          if (chunk.model) assistantModel = chunk.model
          if (!chunk.content && !chunk.model) return

          setMessages((currentMessages) =>
            currentMessages.map((message) =>
              message.id === assistantId
                ? {
                    ...message,
                    content: assistantContent,
                    ...(assistantModel ? { model: assistantModel } : {}),
                  }
                : message
            )
          )
        }
      )
      if (!assistantContent) {
        throw new Error(t('The public chat returned an empty response.'))
      }
    } catch (error) {
      setMessages(messages)
      handleServerError(error, t('Unable to reach the public chat.'))
    } finally {
      setIsSending(false)
      setStreamingMessageId(null)
    }
  }

  return (
    <section id='public-chat' className='px-6 py-16 md:py-24'>
      <div className='mx-auto grid max-w-6xl gap-10 lg:grid-cols-[0.75fr_1.25fr] lg:items-start'>
        <div>
          <div className='text-muted-foreground mb-3 flex items-center gap-2 text-xs font-semibold tracking-[0.18em] uppercase'>
            <MessageCircle className='size-4' aria-hidden='true' />
            {t('Online chat test')}
          </div>
          <h2 className='text-2xl font-semibold tracking-tight md:text-3xl'>
            {t('Ask without a key')}
          </h2>
          <p className='text-muted-foreground mt-4 max-w-md text-sm leading-6'>
            {t(
              'Send a message through the operator-configured group. The gateway chooses an available model automatically.'
            )}
          </p>
          <div className='text-muted-foreground mt-6 flex items-center gap-2 text-xs'>
            <Sparkles className='size-4' aria-hidden='true' />
            {enabled
              ? t('No key or model selection required')
              : t('Online chat is not enabled right now')}
          </div>
        </div>

        <div className='border-border/50 bg-background overflow-hidden rounded-xl border shadow-sm'>
          <div
            className='bg-muted/20 flex min-h-72 flex-col gap-3 p-4'
            aria-live='polite'
          >
            {messages.length === 0 ? (
              <div className='text-muted-foreground flex flex-1 items-center justify-center px-8 text-center text-sm'>
                {t('Start a conversation to test the public route.')}
              </div>
            ) : (
              messages.map((message) => (
                <div
                  key={message.id}
                  className={
                    message.role === 'user'
                      ? 'bg-foreground text-background ml-auto max-w-[85%] rounded-lg px-3 py-2 text-sm leading-6'
                      : 'border-border/50 bg-background max-w-[85%] rounded-lg border px-3 py-2 text-sm leading-6'
                  }
                >
                  {message.content ||
                    (message.id === streamingMessageId
                      ? t('Thinking...')
                      : null)}
                  {message.role === 'assistant' && message.model ? (
                    <div className='text-muted-foreground mt-2 text-xs'>
                      {t('Model {{model}}', { model: message.model })}
                    </div>
                  ) : null}
                </div>
              ))
            )}
          </div>
          <form onSubmit={onSubmit} className='border-border/50 border-t p-4'>
            <label htmlFor='public-chat-input' className='sr-only'>
              {t('Message')}
            </label>
            <Textarea
              id='public-chat-input'
              value={input}
              onChange={(event) => setInput(event.target.value)}
              placeholder={t('Write a message...')}
              disabled={!enabled || isSending}
              rows={3}
              className='min-h-20 resize-none'
            />
            <div className='mt-3 flex items-center justify-between gap-3'>
              <span className='text-muted-foreground text-xs'>
                {t('Please avoid sending private or sensitive information.')}
              </span>
              <Button
                type='submit'
                size='sm'
                disabled={!enabled || isSending || !input.trim()}
                className='rounded-lg'
              >
                <Send className='size-4' aria-hidden='true' />
                {t('Send')}
              </Button>
            </div>
          </form>
        </div>
      </div>
    </section>
  )
}
