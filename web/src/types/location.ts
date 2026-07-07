// ─── Location / Map API Types ──────────────────────────────────────────

/** Payload for PUT /v1/location — updates a user's real-time position */
export interface LocationUpdate {
  user_id: string
  latitude: number
  longitude: number
  updated_at?: string
}

/** Entry in the frontend location log panel */
export interface LocationLogEntry {
  timestamp: string
  latitude: number
  longitude: number
  source: "click" | "drag" | "simulation" | "api" | "websocket"
}

/** A map post (pin) fetched from or sent to the Location Service API */
export interface MapPost {
  id: string
  user_id: string
  title: string
  content?: string
  category: "RESTAURANT" | "SOCIAL_LIFE" | "EVENT"
  image_urls?: string[]
  latitude: number
  longitude: number
  created_at: string
  updated_at: string
}

/** Category key for map posts */
export type PostCategory = "RESTAURANT" | "SOCIAL_LIFE" | "EVENT"

/** A simulated other user on the map */
export interface SimulatedUser {
  user_id: string
  name: string
  color: "#f59e0b" | "#10b981" | "#8b5cf6"
  lat: number
  lng: number
}

/** Payload for creating a new map post (POST /v1/posts) */
export interface CreatePostPayload {
  title: string
  content?: string
  category: "RESTAURANT" | "SOCIAL_LIFE" | "EVENT"
  image_urls?: string[]
  latitude: number
  longitude: number
}
