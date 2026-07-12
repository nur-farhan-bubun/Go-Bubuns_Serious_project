// ─── Chat Service API Client ────────────────────────────────────────────
// REST + WebSocket client for the chat service, proxied through the API Gateway.

import { getAuthHeaders, getToken } from "./auth"
import type { ChatMessage, Conversation } from "../components/chat/ChatData"
import type { ChatUser } from "../components/chat/ChatData"

// ─── Configuration ──────────────────────────────────────────────────────

// REST API base URL. Defaults to same-origin (empty string) so requests go through
// Next.js rewrites (next.config.ts) which proxy to the API gateway on port 8080,
// avoiding CORS issues. Set NEXT_PUBLIC_API_URL to override (e.g. production).
const API_BASE = process.env.NEXT_PUBLIC_API_URL || ""
const CHAT_API = `${API_BASE}/v1`

// WebSocket URL — must point directly to the API gateway because Next.js
// rewrites do NOT proxy WebSocket upgrade requests.
const WS_BASE = process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8080"

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
  type?: string          // "direct" or "group"
  name?: string          // group name (for group conversations)
  user1_id?: string      // direct conversation
  user2_id?: string      // direct conversation
  match_id?: string
  member_ids?: string[]  // member IDs (for group conversations)
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
  type: "chat_message" | "system" | "presence" | "presence_update" | "typing" | "ack" | "error" | "room_ready"
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
 * Create a new group conversation.
 * POST /v1/groups
 */
export async function createGroupConversation(name: string, members: string[]): Promise<ChatAPIConversation | null> {
  try {
    const res = await fetch(`${CHAT_API}/groups`, authFetchOptions({
      method: "POST",
      body: JSON.stringify({ name, members }),
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
      ws = new WebSocket(`${WS_BASE}/ws?${params}`)

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

// ─── User Info API Calls ────────────────────────────────────────────────

/**
 * Fetch all cached users from the chat service.
 * GET /v1/chat/users — proxied to chat-service by the API gateway.
 */
export async function fetchChatUsers(): Promise<ChatAPIUser[]> {
  try {
    const res = await fetch(`${CHAT_API}/chat/users`, authFetchOptions())
    if (!res.ok) return []
    return await res.json()
  } catch {
    return []
  }
}

/**
 * Fetch a single cached user by ID from the chat service.
 * GET /v1/chat/users/{userId}
 */
export async function fetchChatUser(userId: string): Promise<ChatAPIUser | null> {
  try {
    const res = await fetch(`${CHAT_API}/chat/users/${userId}`, authFetchOptions())
    if (!res.ok) return null
    return await res.json()
  } catch {
    return null
  }
}

export interface ChatAPIUser {
  user_id: string
  email: string
  display_name: string
  avatar_url?: string
}

// ─── User Search API Call ────────────────────────────────────────────────

export interface SearchUserResult {
  user_id: string
  display_name: string
  avatar_url?: string
}

/**
 * Search for users by display_name or email.
 * First tries the backend API (GET /v1/users/search), then falls back to
 * searching the chat-service user cache and local registered users.
 */
export async function searchUsers(query: string, limit = 20): Promise<SearchUserResult[]> {
  if (!query.trim()) return []

  const q = query.trim().toLowerCase()

  // Try 1: Backend search API (user-service PostgreSQL ILIKE)
  try {
    const res = await fetch(`${API_BASE}/v1/users/search?q=${encodeURIComponent(q)}&limit=${limit}`, authFetchOptions())
    if (res.ok) {
      const data: SearchUserResult[] = await res.json()
      if (Array.isArray(data) && data.length > 0) {
        return data
      }
    }
  } catch {
    // Fall through to local search
  }

  // Try 2: Search chat-service user cache
  try {
    const chatUsers = await fetchChatUsers()
    const matches = chatUsers
      .filter((u) =>
        u.display_name?.toLowerCase().includes(q) ||
        u.email?.toLowerCase().includes(q)
      )
      .slice(0, limit)
      .map((u) => ({
        user_id: u.user_id,
        display_name: u.display_name || u.user_id,
        avatar_url: u.avatar_url,
      }))
    if (matches.length > 0) return matches
  } catch {
    // Fall through
  }

  // Try 3: Search user-service all-users list (paginated fallback)
  try {
    const res = await fetch(`${API_BASE}/v1/users?limit=${limit}`, authFetchOptions())
    if (res.ok) {
      const data = await res.json()
      const usersList: Record<string, unknown>[] = data.users || data || []
      if (Array.isArray(usersList)) {
        const matches: SearchUserResult[] = usersList
          .filter((u) => {
            const name = ((u.name || u.display_name || "") as string).toLowerCase()
            const email = (u.email as string || "").toLowerCase()
            return name.includes(q) || email.includes(q)
          })
          .slice(0, limit)
          .map((u) => ({
            user_id: (u.id || u.user_id) as string,
            display_name: ((u.name || u.display_name || u.id) as string),
            avatar_url: (u.avatar_url || "") as string,
          }))
        if (matches.length > 0) return matches
      }
    }
  } catch {
    // Fall through
  }

  return []
}

// ─── Block / Unblock API Calls ────────────────────────────────────────────

export interface BlockUserResponse {
  message?: string
  blocker_id?: string
  blocked_id?: string
}

/**
 * Block a user.
 * POST /v1/users/{id}/block
 */
export async function blockUser(blockerId: string, blockedId: string): Promise<BlockUserResponse | null> {
  try {
    const res = await fetch(`${API_BASE}/v1/users/${blockerId}/block`, authFetchOptions({
      method: "POST",
      body: JSON.stringify({ blocked_id: blockedId }),
    }))
    if (!res.ok) return null
    return await res.json()
  } catch {
    return null
  }
}

/**
 * Unblock a user.
 * DELETE /v1/users/{id}/block/{blockedId}
 */
export async function unblockUser(blockerId: string, blockedId: string): Promise<BlockUserResponse | null> {
  try {
    const res = await fetch(`${API_BASE}/v1/users/${blockerId}/block/${blockedId}`, authFetchOptions({
      method: "DELETE",
    }))
    if (!res.ok) return null
    return await res.json()
  } catch {
    return null
  }
}

/**
 * Fetch users directly from the user-service API.
 * GET /v1/users — proxied to user-service by the API gateway.
 * The user-service returns a paginated response with a nested "users" array.
 * This is used as a fallback when the chat-service cache is empty.
 */
export async function fetchUsersFromUserService(): Promise<ChatAPIUser[]> {
  try {
    const res = await fetch(`${API_BASE}/v1/users`, authFetchOptions())
    if (!res.ok) return []
    const data = await res.json()
    // user-service returns { users: [...], total, page, limit }
    const usersList = data.users || data || []
    if (!Array.isArray(usersList)) return []

    return usersList.map((u: Record<string, unknown>) => ({
      user_id: (u.id as string) || "",
      email: (u.email as string) || "",
      display_name: (u.name as string) || (u.display_name as string) || "",
      avatar_url: (u.avatar_url as string) || "",
    }))
  } catch {
    return []
  }
}

// ─── Type Converters ────────────────────────────────────────────────────

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
 * Derive a human-readable display name from a user ID string.
 * Splits on [-_] separators and capitalizes each segment.
 * Falls back to "User <short-id>" if parsing fails.
 */
export function userIdToDisplayName(userId: string): string {
  if (!userId) return "Unknown"
  const parts = userId.split(/[-_]/)
  if (parts.length >= 2) {
    return parts
      .slice(1)
      .map((p) => p.charAt(0).toUpperCase() + p.slice(1))
      .join(" ")
      .slice(0, 20)
  }
  return `User ${userId.slice(0, 6)}`
}

/**
 * Derive initials from a user ID string.
 * Uses the segments after [-_] separators.
 * Falls back to the first two characters of the ID.
 */
export function userIdToInitials(userId: string): string {
  if (!userId) return "U"
  const parts = userId.split(/[-_]/)
  if (parts.length >= 2) {
    return parts
      .slice(1)
      .map((p) => p[0]?.toUpperCase() || "")
      .join("")
      .slice(0, 2)
  }
  return userId.slice(0, 2).toUpperCase()
}

/**
 * Convert a ChatAPIConversation (from the API) to the local Conversation type.
 * Handles both direct and group conversations.
 * Members are populated from member_ids if available (group conversations).
 */
export function fromAPIConversation(apiConv: ChatAPIConversation, registeredUsers?: ChatUser[]): Conversation {
  if (apiConv.type === "group") {
    const initials =
      apiConv.name
        ?.split(" ")
        .map((w) => w[0])
        .join("")
        .toUpperCase()
        .slice(0, 2) || "G"

    // Populate members from member_ids if available
    let members: ChatUser[] = []
    if (apiConv.member_ids && registeredUsers) {
      members = apiConv.member_ids
        .map((uid) => registeredUsers.find((u) => u.id === uid))
        .filter((u): u is ChatUser => u !== undefined)
    }

    return {
      id: apiConv.id,
      type: "group",
      workspaceId: "",
      name: apiConv.name || "Group",
      avatar: initials,
      lastMessage: "",
      lastTime: new Date(apiConv.created_at).toLocaleString(),
      unread: 0,
      isActive: false,
      members,
      onlineCount: members.filter((m) => m.status === "online").length,
    }
  }

  // Direct conversation
  const otherUserId = apiConv.user2_id || ""
  const name = userIdToDisplayName(otherUserId) || `Chat #${apiConv.id.slice(0, 4)}`
  const initials = userIdToInitials(otherUserId)

  return {
    id: apiConv.id,
    type: "direct",
    workspaceId: "",
    name,
    avatar: initials,
    lastMessage: "",
    lastTime: new Date(apiConv.created_at).toLocaleString(),
    unread: 0,
    isActive: false,
    members: [],
    onlineCount: 0,
  }
}

/**
 * Get the WebSocket URL for a chat room.
 */
export function getChatWSURL(userID: string, roomID?: string): string {
  const params = new URLSearchParams({ user_id: userID })
  if (roomID) params.set("room_id", roomID)
  return `${WS_BASE}/ws?${params}`
}


