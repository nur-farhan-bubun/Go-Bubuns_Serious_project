// ─── Types ──────────────────────────────────────────────────────────────

export interface ChatUser {
  id: string
  name: string
  avatar: string
  color: string
  status: "online" | "idle" | "dnd" | "offline"
  role: string
}

export interface Workspace {
  id: string
  label: string
  icon: string
  color: string
}

export interface Conversation {
  id: string
  workspaceId: string
  name: string
  avatar: string
  lastMessage: string
  lastTime: string
  unread: number
  isActive: boolean
  members: ChatUser[]
  onlineCount: number
}

export interface ChatMessage {
  id: string
  conversationId: string
  senderId: string
  content: string
  timestamp: string
  type: "text" | "system" | "call"
  attachments?: { name: string; size: string }[]
}

export interface SharedAsset {
  id: string
  conversationId: string
  type: "photo" | "file" | "link"
  url: string
  name: string
  size?: string
}
