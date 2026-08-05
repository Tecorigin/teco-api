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
import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2, Plus, Send, Trash2, MessageSquare } from 'lucide-react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Textarea } from '@/components/ui/textarea'
import { Markdown } from '@/components/ui/markdown'
import { getCommonHeaders } from '@/lib/api'

import {
  createKbSession,
  deleteKbSession,
  listKbSessions,
} from '../api'
import type { KbSession, KbChatMessage, KbReference } from '../types'

type StoredKbChat = {
  activeSession: string | null
  messages: Record<string, KbChatMessage[]>
  sessionNames: Record<string, string>
}

const KB_CHAT_STORAGE_PREFIX = 'knowledge-base-chat:'
const MAX_SESSION_TITLE_LENGTH = 24

function getKbChatStorageKey(): string | null {
  if (typeof window === 'undefined') return null
  const userId = window.localStorage.getItem('uid')
  return userId ? `${KB_CHAT_STORAGE_PREFIX}${userId}` : null
}

function loadStoredKbChat(): StoredKbChat {
  const key = getKbChatStorageKey()
  if (!key) {
    return { activeSession: null, messages: {}, sessionNames: {} }
  }
  try {
    const parsed = JSON.parse(window.localStorage.getItem(key) || '{}') as Partial<StoredKbChat>
    return {
      activeSession: typeof parsed.activeSession === 'string' ? parsed.activeSession : null,
      messages: parsed.messages || {},
      sessionNames: parsed.sessionNames || {},
    }
  } catch {
    return { activeSession: null, messages: {}, sessionNames: {} }
  }
}

function getSessionTitle(query: string): string {
  const normalized = query.replaceAll(/\s+/g, ' ').trim()
  if (normalized.length <= MAX_SESSION_TITLE_LENGTH) return normalized
  return `${normalized.slice(0, MAX_SESSION_TITLE_LENGTH)}...`
}

export function KbChat() {
  const { t } = useTranslation()
  const [storedChat] = useState<StoredKbChat>(() => loadStoredKbChat())
  const [sessions, setSessions] = useState<KbSession[]>([])
  const [activeSession, setActiveSession] = useState<string | null>(storedChat.activeSession)
  const [messages, setMessages] = useState<Record<string, KbChatMessage[]>>(storedChat.messages)
  const [sessionNames, setSessionNames] = useState<Record<string, string>>(storedChat.sessionNames)
  const [input, setInput] = useState('')
  const [streaming, setStreaming] = useState(false)
  const bottomRef = useRef<HTMLDivElement>(null)
  const abortRef = useRef<AbortController | null>(null)
  const sessionNamesRef = useRef(sessionNames)

  const loadSessions = useCallback(async () => {
    try {
      const res = await listKbSessions()
      if (res.success) {
        setSessions((previous) => {
          return (res.data || []).map((session) => ({
            ...session,
            name: sessionNamesRef.current[session.session_id] || session.name,
          }))
        })
      }
    } catch {
      // ignore
    }
  }, [])

  useEffect(() => {
    loadSessions()
  }, [loadSessions])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, activeSession])

  useEffect(() => {
    sessionNamesRef.current = sessionNames
  }, [sessionNames])

  useEffect(() => {
    const key = getKbChatStorageKey()
    if (!key) return
    try {
      window.localStorage.setItem(key, JSON.stringify({ activeSession, messages, sessionNames }))
    } catch {
      // Ignore unavailable or full browser storage.
    }
  }, [activeSession, messages, sessionNames])

  const handleNewSession = async () => {
    try {
      const res = await createKbSession()
      if (res.success && res.data?.session_id) {
        const sid = res.data.session_id
        setSessions((prev) => [...prev, { session_id: sid, name: t('新会话') }])
        setSessionNames((prev) => ({ ...prev, [sid]: t('新会话') }))
        setMessages((prev) => ({ ...prev, [sid]: [] }))
        setActiveSession(sid)
      } else {
        toast.error(res.message || t('创建会话失败'))
      }
    } catch {
      toast.error(t('创建会话失败'))
    }
  }

  const handleDeleteSession = async (sid: string) => {
    try {
      const res = await deleteKbSession(sid)
      if (res.success) {
        setSessions((prev) => prev.filter((s) => s.session_id !== sid))
        setMessages((prev) => {
          const next = { ...prev }
          delete next[sid]
          return next
        })
        setSessionNames((prev) => {
          const next = { ...prev }
          delete next[sid]
          return next
        })
        if (activeSession === sid) setActiveSession(null)
      } else {
        toast.error(res.message || t('删除失败'))
      }
    } catch {
      toast.error(t('删除失败'))
    }
  }

  const appendMessage = (sid: string, msg: KbChatMessage) => {
    setMessages((prev) => ({
      ...prev,
      [sid]: [...(prev[sid] || []), msg],
    }))
  }

  const updateLastAssistant = (sid: string, content: string, references?: KbReference[]) => {
    setMessages((prev) => {
      const list = [...(prev[sid] || [])]
      for (let i = list.length - 1; i >= 0; i--) {
        if (list[i].role === 'assistant') {
          list[i] = { role: 'assistant', content, references }
          break
        }
      }
      return { ...prev, [sid]: list }
    })
  }

  const handleSend = async () => {
    const query = input.trim()
    if (!query || streaming) return
    let sid = activeSession
    if (!sid) {
      try {
        const res = await createKbSession()
        if (!res.success || !res.data?.session_id) {
          toast.error(t('创建会话失败'))
          return
        }
        sid = res.data.session_id
        setSessions((prev) => [...prev, { session_id: res.data.session_id, name: t('新会话') }])
        setSessionNames((prev) => ({ ...prev, [res.data.session_id]: t('新会话') }))
        setActiveSession(sid)
      } catch {
        toast.error(t('创建会话失败'))
        return
      }
    }
    const sessionId = sid
    setInput('')
    if ((messages[sessionId] || []).length === 0) {
      const title = getSessionTitle(query)
      setSessionNames((previous) => ({ ...previous, [sessionId]: title }))
      setSessions((current) => current.map((session) => (
        session.session_id === sessionId ? { ...session, name: title } : session
      )))
    }
    appendMessage(sessionId, { role: 'user', content: query })
    appendMessage(sessionId, { role: 'assistant', content: '' })
    setStreaming(true)

    const controller = new AbortController()
    abortRef.current = controller
    let acc = ''
    let refs: KbReference[] | undefined
    try {
      const resp = await fetch('/api/knowledge-base/chat/session_chat', {
        method: 'POST',
        headers: getCommonHeaders(),
        credentials: 'same-origin',
        body: JSON.stringify({ query, session_id: sessionId, stream: true, return_docs: true }),
        signal: controller.signal,
      })
      if (!resp.ok || !resp.body) {
        const text = await resp.text().catch(() => '')
        throw new Error(text || `HTTP ${resp.status}`)
      }
      const reader = resp.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      for (;;) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''
        for (const line of lines) {
          const trimmed = line.trim()
          if (!trimmed.startsWith('data:')) continue
          const raw = trimmed.slice(5).trim()
          if (!raw) continue
          try {
            const evt = JSON.parse(raw)
            if (evt.content) acc += evt.content
            if (evt.reference) refs = evt.reference
            updateLastAssistant(sessionId, acc, refs)
          } catch {
            // ignore malformed chunk
          }
        }
      }
      updateLastAssistant(sessionId, acc, refs)
    } catch (e) {
      if ((e as Error).name !== 'AbortError') {
        toast.error(t('问答失败'))
        updateLastAssistant(sessionId, acc || t('（请求失败）'), refs)
      }
    } finally {
      setStreaming(false)
      abortRef.current = null
    }
  }

  const currentMessages = activeSession ? messages[activeSession] || [] : []

  return (
    <div className='flex h-full min-h-0 gap-4 pt-2'>
      {/* Sessions sidebar */}
      <div className='flex min-h-0 w-56 shrink-0 flex-col gap-2'>
        <Button variant='outline' size='sm' onClick={handleNewSession} className='w-full'>
          <Plus className='mr-1.5 h-4 w-4' />
          {t('新会话')}
        </Button>
        <ScrollArea className='flex-1'>
          <div className='space-y-1'>
            {sessions.map((s) => (
              <div
                key={s.session_id}
                className={`group flex items-center gap-1 rounded-md px-2 py-1.5 text-sm hover:bg-accent ${
                  activeSession === s.session_id ? 'bg-accent' : ''
                }`}
              >
                <button
                  type='button'
                  className='flex min-w-0 flex-1 items-center gap-2 text-left'
                  onClick={() => setActiveSession(s.session_id)}
                >
                  <MessageSquare className='h-3.5 w-3.5 shrink-0 text-muted-foreground' />
                  <span className='truncate'>{s.name || s.session_id.slice(0, 12)}</span>
                </button>
                <button
                  type='button'
                  className='opacity-0 transition-opacity group-hover:opacity-100'
                  onClick={() => handleDeleteSession(s.session_id)}
                  title={t('删除会话')}
                >
                  <Trash2 className='h-3.5 w-3.5 text-muted-foreground hover:text-destructive' />
                </button>
              </div>
            ))}
          </div>
        </ScrollArea>
      </div>

      {/* Chat area */}
      <div className='flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden rounded-md border'>
        <ScrollArea className='min-h-0 flex-1 p-4'>
          {currentMessages.length === 0 ? (
            <div className='flex h-full flex-col items-center justify-center gap-2 text-muted-foreground'>
              <MessageSquare className='h-10 w-10' />
              <p className='text-sm'>{t('开始提问，基于你的知识库进行问答')}</p>
            </div>
          ) : (
            <div className='min-w-0 space-y-4'>
              {currentMessages.map((m, i) => (
                // eslint-disable-next-line react/no-array-index-key -- messages are append-only and order-stable
                <div key={i} className={`flex ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}>
                  <div
                    className={`min-w-0 max-w-[85%] overflow-x-auto break-words rounded-lg px-3 py-2 text-sm ${
                      m.role === 'user'
                        ? 'bg-primary text-primary-foreground'
                        : 'bg-muted'
                    }`}
                  >
                    {m.role === 'assistant' ? (
                      <>
                        {m.content ? <Markdown className='min-w-0 break-words [&_pre]:max-w-full [&_pre]:overflow-x-auto'>{m.content}</Markdown> : (
                          <Loader2 className='h-4 w-4 animate-spin' />
                        )}
                        {m.references && m.references.length > 0 && (
                          <div className='mt-2 border-t pt-2 text-xs text-muted-foreground'>
                            <div className='mb-1 font-medium'>{t('参考来源')}</div>
                            {m.references.map((r, j) => (
                              // eslint-disable-next-line react/no-array-index-key -- reference list is short and static per message
                              <div key={j} className='truncate'>
                                · {r.document_name || r.document_id}
                                {r.similarity != null && ` (${(r.similarity * 100).toFixed(0)}%)`}
                              </div>
                            ))}
                          </div>
                        )}
                      </>
                    ) : (
                      <span className='whitespace-pre-wrap'>{m.content}</span>
                    )}
                  </div>
                </div>
              ))}
              <div ref={bottomRef} />
            </div>
          )}
        </ScrollArea>
        <div className='border-t p-3'>
          <div className='flex items-end gap-2'>
            <Textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder={t('输入问题，Enter 发送，Shift+Enter 换行')}
              rows={2}
              className='min-h-[52px] resize-none'
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  handleSend()
                }
              }}
            />
            <Button onClick={handleSend} disabled={streaming || !input.trim()} size='icon' className='h-[52px] w-[52px]'>
              {streaming ? <Loader2 className='h-4 w-4 animate-spin' /> : <Send className='h-4 w-4' />}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
