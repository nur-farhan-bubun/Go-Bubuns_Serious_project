"use client"

import { useEffect, useRef, useCallback, useState } from "react"

// ─── Types ──────────────────────────────────────────────────────────────

import type { WSLocationMessage, WSState } from "../types/websocket"

// ─── Configuration ──────────────────────────────────────────────────────

// Reconnect delays (ms) — exponential backoff
const INITIAL_RECONNECT_DELAY = 1000
const MAX_RECONNECT_DELAY = 15000
const BACKOFF_MULTIPLIER = 1.5

// ─── Hook ───────────────────────────────────────────────────────────────

export function useLocationWebSocket(
  userID: string,
  onLocationUpdate?: (msg: WSLocationMessage) => void,
) {
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const reconnectDelayRef = useRef(INITIAL_RECONNECT_DELAY)
  const mountedRef = useRef(true)
  const onUpdateRef = useRef(onLocationUpdate)

  // Keep the callback ref current without causing reconnects
  useEffect(() => {
    onUpdateRef.current = onLocationUpdate
  }, [onLocationUpdate])

  const [state, setState] = useState<WSState>({
    status: "disconnected",
    lastMessage: null,
    messageCount: 0,
  })

  // ─── Build WebSocket URL ──────────────────────────────────────────
  const getWSURL = useCallback(() => {
    const apiBase = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"
    // Convert http:// to ws://, https:// to wss://
    const wsBase = apiBase.replace(/^http/, "ws")
    return `${wsBase}/v1/location/ws?user_id=${userID}`
  }, [userID])

  // ─── Connect ──────────────────────────────────────────────────────
  const connect = useCallback(() => {
    if (!mountedRef.current) return

    const url = getWSURL()
    console.log(`🔌 [WS] Connecting to ${url}`)
    setState((s) => ({ ...s, status: "connecting" }))

    try {
      const ws = new WebSocket(url)
      wsRef.current = ws

      ws.onopen = () => {
        if (!mountedRef.current) {
          ws.close()
          return
        }
        console.log(`✅ [WS] Connected — user_id=${userID}`)
        setState((s) => ({ ...s, status: "connected" }))
        reconnectDelayRef.current = INITIAL_RECONNECT_DELAY // reset backoff
      }

      ws.onmessage = (event: MessageEvent) => {
        if (!mountedRef.current) return
        try {
          const msg: WSLocationMessage = JSON.parse(event.data)

          console.log(
            `📡 [WS] RECEIVED → user=${msg.user_id} ` +
              `(${msg.latitude.toFixed(6)}, ${msg.longitude.toFixed(6)})`,
          )

          setState((s) => ({
            ...s,
            lastMessage: msg,
            messageCount: s.messageCount + 1,
          }))

          onUpdateRef.current?.(msg)
        } catch (err) {
          console.warn(`⚠️ [WS] Failed to parse message: ${err}`)
        }
      }

      ws.onclose = (event) => {
        if (!mountedRef.current) return
        console.log(
          `🔌 [WS] Disconnected (code=${event.code})` +
            (event.code !== 1000 ? " — will reconnect..." : ""),
        )
        setState((s) => ({ ...s, status: "disconnected" }))
        wsRef.current = null

        // Auto-reconnect on unexpected close
        if (event.code !== 1000) {
          scheduleReconnect()
        }
      }

      ws.onerror = () => {
        if (!mountedRef.current) return
        console.warn(`⚠️ [WS] Connection error`)
        setState((s) => ({ ...s, status: "error" }))
      }
    } catch (err) {
      console.warn(`⚠️ [WS] Failed to create WebSocket: ${err}`)
      setState((s) => ({ ...s, status: "error" }))
      scheduleReconnect()
    }
  }, [getWSURL, userID])

  // ─── Reconnect with backoff ───────────────────────────────────────
  const scheduleReconnect = useCallback(() => {
    if (reconnectTimerRef.current) return // already scheduled

    const delay = reconnectDelayRef.current
    console.log(`🔌 [WS] Reconnecting in ${delay}ms...`)

    reconnectTimerRef.current = setTimeout(() => {
      reconnectTimerRef.current = null
      reconnectDelayRef.current = Math.min(
        reconnectDelayRef.current * BACKOFF_MULTIPLIER,
        MAX_RECONNECT_DELAY,
      )
      connect()
    }, delay)
  }, [connect])

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
    setState({ status: "disconnected", lastMessage: null, messageCount: 0 })
  }, [])

  // ─── Lifecycle ────────────────────────────────────────────────────
  useEffect(() => {
    mountedRef.current = true
    connect()

    return () => {
      mountedRef.current = false
      disconnect()
    }
  }, [connect, disconnect])

  return {
    ...state,
    connect,
    disconnect,
  }
}
