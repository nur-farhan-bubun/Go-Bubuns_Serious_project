"use client"

// ─── Chat Handshake Hook ────────────────────────────────────────────────
// Unified React lifecycle pattern that executes when the main app layout
// loads. It:
//   1. Establishes a global WebSocket loop via the Gateway route.
//   2. Listens for "presence" and "room_ready" message frames.
//   3. Provides a `startConversation` function that POSTs /v1/conversations
//      and on success redirects to /messages/<conversation_id>.
//   4. Automatically redirects when a "room_ready" frame is received
//      (the recipient path — the initiator receives a 201 with the ID).

import { useEffect, useRef, useCallback, useState } from "react"
import { useRouter } from "next/navigation"
import { getToken } from "./auth"
import type { ChatWSEnvelope } from "./chat"
import { useChatStore } from "../components/chat/ChatStore"

// ─── Configuration ──────────────────────────────────────────────────────

// REST API base URL — goes through Next.js rewrites to avoid CORS.
const API_BASE = process.env.NEXT_PUBLIC_API_URL || ""
// WebSocket URL — must point directly to the API gateway (Next.js rewrites
// don't proxy WebSocket upgrades).
const WS_BASE = process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8080"

// ─── Types ──────────────────────────────────────────────────────────────

export type HandshakeStatus =
  | "idle"
  | "connecting"
  | "connected"
  | "disconnected"
  | "error"

export interface HandshakeState {
  status: HandshakeStatus
  conversationId: string | null // set when room_ready is received
}

// ─── Hook ───────────────────────────────────────────────────────────────

export function useChatHandshake(userId: string) {
  const router = useRouter()
  const [state, setState] = useState<HandshakeState>({
    status: "idle",
    conversationId: null,
  })

  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const reconnectDelayRef = useRef(1000)
  const mountedRef = useRef(true)
  const userIdRef = useRef(userId)

  // Keep the user ID ref current without causing reconnects
  useEffect(() => {
    userIdRef.current = userId
  }, [userId])

  // ─── Build WebSocket URL ──────────────────────────────────────────
  const getWSURL = useCallback(() => {
    const token = getToken()
    const params = new URLSearchParams({ user_id: userId })
    if (token) params.set("token", token)
    return `${WS_BASE}/ws?${params}`
  }, [userId])

  // ─── Connect ──────────────────────────────────────────────────────
  const connect = useCallback(() => {
    if (!mountedRef.current || !userIdRef.current) return

    const url = getWSURL()
    setState((s) => ({ ...s, status: "connecting" }))

    try {
      const ws = new WebSocket(url)
      wsRef.current = ws

      ws.onopen = () => {
        if (!mountedRef.current) {
          ws.close()
          return
        }
        setState((s) => ({ ...s, status: "connected" }))
        reconnectDelayRef.current = 1000 // reset backoff
      }

      ws.onmessage = (event: MessageEvent) => {
        if (!mountedRef.current) return
        try {
          const envelope: ChatWSEnvelope = JSON.parse(event.data)

          // ── Handle presence updates ──────────────────────────────
          // Handle both "presence" (existing backend type) and
          // "presence_update" (spec-compliant type) for compatibility.
          if (envelope.type === "presence" || envelope.type === "presence_update") {
            // The global ChatStore handles presence for the UI.
            // This hook forwards the event so the store can update.
            if (typeof window !== "undefined") {
              window.dispatchEvent(
                new CustomEvent("chat:presence", {
                  detail: envelope.data,
                }),
              )
            }
          }

          // ── Handle room_ready (recipient side) ───────────────────
          // Opens the chat overlay on the current page and connects to
          // the conversation room — no full-page navigation.
          if (envelope.type === "room_ready") {
            const data = envelope.data as Record<string, unknown>
            const convId = (data.conversation_id as string) || envelope.room_id
            if (convId) {
              setState((s) => ({ ...s, conversationId: convId }))
              // Open chat overlay in-place and connect to the room
              useChatStore.getState().joinConversation(convId)
            }
          }
        } catch {
          // Silently ignore malformed messages
        }
      }

      ws.onclose = () => {
        if (!mountedRef.current) return
        setState((s) => ({ ...s, status: "disconnected" }))
        wsRef.current = null
        scheduleReconnect()
      }

      ws.onerror = () => {
        if (!mountedRef.current) return
        setState((s) => ({ ...s, status: "error" }))
      }
    } catch {
      setState((s) => ({ ...s, status: "error" }))
      scheduleReconnect()
    }
  }, [getWSURL, router])

  // ─── Reconnect with backoff ───────────────────────────────────────
  const scheduleReconnect = useCallback(() => {
    if (reconnectTimerRef.current) return

    const delay = reconnectDelayRef.current
    reconnectTimerRef.current = setTimeout(() => {
      reconnectTimerRef.current = null
      reconnectDelayRef.current = Math.min(reconnectDelayRef.current * 1.5, 15000)
      connect()
    }, delay)
  }, [connect])

  // ─── Start a conversation (initiator side) ────────────────────────
  // POST /v1/conversations and redirect on success.
  const startConversation = useCallback(
    async (recipientId: string, matchId?: string) => {
      if (!userIdRef.current || !recipientId) return null

      try {
        const res = await fetch(`${API_BASE}/v1/conversations`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${getToken() ?? ""}`,
          },
          body: JSON.stringify({
            user_id: recipientId,
            match_id: matchId,
          }),
        })

        if (!res.ok) return null

        const data = await res.json()
        const convId = data.id || data.conversation_id

        if (convId) {
          // Redirect to the conversation dashboard
          router.push(`/messages/${convId}`)
          return convId
        }

        return null
      } catch {
        return null
      }
    },
    [router],
  )

  // ─── Disconnect ───────────────────────────────────────────────────
  const disconnect = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current)
      reconnectTimerRef.current = null
    }
    if (wsRef.current) {
      wsRef.current.close(1000, "Client disconnect")
      wsRef.current = null
    }
    setState({ status: "disconnected", conversationId: null })
  }, [])

  // ─── Lifecycle ────────────────────────────────────────────────────
  useEffect(() => {
    if (!userId) return

    mountedRef.current = true
    connect()

    return () => {
      mountedRef.current = false
      disconnect()
    }
  }, [userId, connect, disconnect])

  return {
    ...state,
    startConversation,
    disconnect,
  }
}
