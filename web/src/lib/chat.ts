// ─── Chat Service API Client ────────────────────────────────────────────
// REST + WebSocket client for the chat service, proxied through the API Gateway.

import { getAuthHeaders, getToken } from "./auth"
import type { ChatMessage, Conversation, ChatUser } from "../components/chat/ChatData"

// ─── Configuration ──────────────────────────────────────────────────────

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"
const CHAT_API = `${API_BASE}/v1`

// ─── Auth helpers ───────────────────────────────────────────────────────

/**
 * Build fetch options with auth headers.
 */
function authFetchOptions(options: RequestInit = {}): RequestInit {
  return {
    ...options,
    headers: {
      ...getAuthHeaders(),
      ...(options.headers || {}),
    },
  }
}

// ─── Types ──────────────────────────────────────────────────────────────

export interface ChatAPIConversation {
  id: string
  user1_id: string
  user2_id: string
  match_id?: string
  created_at: string
}

export interface ChatAPIMessage {
  id: string
  conversation_id: string
  sender_id: string
  content: string
  created_at: string
}

export interface ChatAPIPresence {
  user_id: string
  status: "online" | "offline" | "away"
  last_seen: string
}

export interface ChatWSEnvelope {
  type: "chat_message" | "system" | "presence" | "typing" | "ack" | "error"
  room_id?: string
  data: Record<string, unknown>
}

// ─── REST API Calls ─────────────────────────────────────────────────────

/**
 * Get all conversations for the current user.
 * GET /v1/conversations
 */
export async function fetchConversations(): Promise<ChatAPIConversation[]> {
  try {
    const res = await fetch(`${CHAT_API}/conversations`, authFetchOptions())
    if (!res.ok) return []
    return await res.json()
  } catch (_) {
    return []
  }
}

/**
 * Create a new conversation between two users (initiated by a match).
 * POST /v1/conversations
 */
export async function createConversation(userId: string, matchId?: string): Promise<ChatAPIConversation | null> {
  try {
    const res = await fetch(`${CHAT_API}/conversations`, authFetchOptions({
      method: "POST",
      body: JSON.stringify({ user_id: userId, match_id: matchId }),
    }))
    if (!res.ok) return null
    return await res.json()
  } catch (_) {
    return null
  }
}

/**
 * Get messages in a conversation, paginated.
 * GET /v1/conversations/{id}/messages?limit=50
 */
export async function fetchMessages(conversationId: string, limit = 50, before?: string): Promise<ChatAPIMessage[]> {
  const params = new URLSearchParams({ limit: String(limit) })
  if (before) params.set("before", before)

  try {
    const res = await fetch(`${CHAT_API}/conversations/${conversationId}/messages?${params}`, authFetchOptions())
    if (!res.ok) return []
    return await res.json()
  } catch (_) {
    return []
  }
}

/**
 * Send a message in a conversation.
 * POST /v1/conversations/{id}/messages
 */
export async function sendMessageAPI(conversationId: string, content: string): Promise<ChatAPIMessage | null> {
  try {
    const res = await fetch(`${CHAT_API}/conversations/${conversationId}/messages`, authFetchOptions({
      method: "POST",
      body: JSON.stringify({ content }),
    }))
    if (!res.ok) return null
    return await res.json()
  } catch (_) {
    return null
  }
}

/**
 * Get a user's presence status.
 * GET /v1/presence/{userID}
 */
export async function fetchPresence(userID: string): Promise<ChatAPIPresence | null> {
  try {
    const res = await fetch(`${CHAT_API}/presence/${userID}`, authFetchOptions())
    if (!res.ok) return null
    return await res.json()
  } catch (_) {
    return null
  }
}

// ─── WebSocket ──────────────────────────────────────────────────────────

export type ChatWSMessageHandler = (envelope: ChatWSEnvelope) => void

export interface ChatWSConnection {
  send: (data: unknown) => void
  close: () => void
  isConnected: () => boolean
}

/**
 * Connect to the chat WebSocket for real-time messaging.
 * ws://host:port/ws?user_id=xxx&room_id=xxx
 */
export function connectChatWS(
  userID: string,
  roomID?: string,
  onMessage?: ChatWSMessageHandler,
  onStatusChange?: (status: "connecting" | "connected" | "disconnected" | "error") => void,
): ChatWSConnection {
  const wsBase = API_BASE.replace(/^http/, "ws")
  const params = new URLSearchParams({ user_id: userID })
  if (roomID) params.set("room_id", roomID)

  let ws: WebSocket | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let reconnectDelay = 1000
  let mounted = true

  function connect() {
    if (!mounted) return

    onStatusChange?.("connecting")

    try {
      // Pass auth token via query param for WebSocket connections
      // (WebSocket doesn't support custom headers, so query param is the standard approach)
      const token = getToken()
      if (token) {
        params.set("token", token)
      }
      ws = new WebSocket(`${wsBase}/ws?${params}`)

      ws.onopen = () => {
        if (!mounted) {
          ws?.close()
          return
        }
        onStatusChange?.("connected")
        reconnectDelay = 1000
      }

      ws.onmessage = (event: MessageEvent) => {
        if (!mounted) return
        try {
          const envelope: ChatWSEnvelope = JSON.parse(event.data)
          onMessage?.(envelope)
        } catch (_) {
          console.warn("[ChatWS] Failed to parse message")
        }
      }

      ws.onclose = () => {
        if (!mounted) return
        onStatusChange?.("disconnected")
        ws = null
        scheduleReconnect()
      }

      ws.onerror = () => {
        if (!mounted) return
        onStatusChange?.("error")
      }
    } catch (_) {
      onStatusChange?.("error")
      scheduleReconnect()
    }
  }

  function scheduleReconnect() {
    if (reconnectTimer || !mounted) return
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      reconnectDelay = Math.min(reconnectDelay * 1.5, 15000)
      connect()
    }, reconnectDelay)
  }

  connect()

  return {
    send: (data: unknown) => {
      if (ws?.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify(data))
      }
    },
    close: () => {
      mounted = false
      if (reconnectTimer) {
        clearTimeout(reconnectTimer)
        reconnectTimer = null
      }
      ws?.close(1000, "Client disconnect")
      ws = null
    },
    isConnected: () => ws?.readyState === WebSocket.OPEN,
  }
}

// ─── Mock-to-API type converters ────────────────────────────────────────

/**
 * Convert mock ChatMessage to the API format for sending.
 */
export function toAPIMessage(msg: ChatMessage): { content: string } {
  return { content: msg.content }
}

/**
 * Convert API ChatAPIMessage to the local ChatMessage type.
 */
export function fromAPIMessage(msg: ChatAPIMessage, currentUserID: string): ChatMessage {
  return {
    id: msg.id,
    conversationId: msg.conversation_id,
    senderId: msg.sender_id,
    content: msg.content,
    timestamp: msg.created_at,
    type: "text",
  }
}

/**
 * Get the WebSocket URL for a chat room.
 */
export function getChatWSURL(userID: string, roomID?: string): string {
  const wsBase = API_BASE.replace(/^http/, "ws")
  const params = new URLSearchParams({ user_id: userID })
  if (roomID) params.set("room_id", roomID)
  return `${wsBase}/ws?${params}`
}


