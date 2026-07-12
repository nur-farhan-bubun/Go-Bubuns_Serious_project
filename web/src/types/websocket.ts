// ─── WebSocket Types ────────────────────────────────────────────────────

/** Matches the Go backend's LocationMessage struct */
export interface WSLocationMessage {
  type: "location_update"
  user_id: string
  latitude: number
  longitude: number
}

/** Chat handshake / presence event types */
export interface WSPresenceUpdate {
  type: "presence"
  user_id: string
  status: string
}

export interface WSRoomReady {
  type: "room_ready"
  conversation_id: string
}

export type WSStatus = "connecting" | "connected" | "disconnected" | "error"

export interface WSState {
  status: WSStatus
  lastMessage: WSLocationMessage | null
  messageCount: number
}
