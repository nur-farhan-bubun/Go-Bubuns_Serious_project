"use client"

import { useEffect, useRef } from "react"
import { useParams } from "next/navigation"
import dynamic from "next/dynamic"
import { useChatStore } from "../../../components/chat/ChatStore"
import { useChatHandshake } from "../../../lib/useChatHandshake"
import { fetchConversations, fromAPIConversation } from "../../../lib/chat"
import type { Conversation } from "../../../components/chat/ChatData"

// ─── Dynamic imports ────────────────────────────────────────────────────

const ChatOverlay = dynamic(
  () => import("../../../components/chat/ChatOverlay"),
  { ssr: false },
)
const WorkspaceBar = dynamic(
  () => import("../../../components/chat/WorkspaceBar"),
  { ssr: false },
)

// ─── Helpers ────────────────────────────────────────────────────────────

function userColorFromId(id: string): string {
  const colors = [
    "#06D6A0", "#ED4245", "#57F287", "#FEE75C", "#EB459E",
    "#1ABC9C", "#9B59B6", "#3498DB", "#E67E22", "#00BCD4",
  ]
  let hash = 0
  for (let i = 0; i < id.length; i++) {
    hash = id.charCodeAt(i) + ((hash << 5) - hash)
  }
  return colors[Math.abs(hash) % colors.length]
}

// ─── Page Component ─────────────────────────────────────────────────────

export default function MessagesPage() {
  const params = useParams()
  const conversationId = params?.id as string | undefined

  const currentUser = useChatStore((s) => s.currentUser)
  const isChatOpen = useChatStore((s) => s.isChatOpen)
  const activeConvId = useChatStore((s) => s.activeConversationId)
  const setConversation = useChatStore((s) => s.setConversation)
  const openChat = useChatStore((s) => s.openChat)
  // Global WebSocket handshake for presence and room_ready notifications
  useChatHandshake(currentUser.id)

  const initializedRef = useRef(false)

  // ─── Initialize: set conversation + open chat ─────────────────────
  useEffect(() => {
    // Skip if already targeting this conversation
    if (initializedRef.current && activeConvId === conversationId && isChatOpen) {
      return
    }

    if (!currentUser.id || !conversationId || conversationId === "undefined") {
      return
    }

    initializedRef.current = true

    // Open the chat overlay immediately so the UI feels responsive
    openChat()

    // Try to find the conversation in the store first
    const state = useChatStore.getState()
    const existing = state.conversations.find((c) => c.id === conversationId)

    if (existing) {
      // Already loaded — just set it active
      setConversation(conversationId)
      return
    }

    // Fetch conversations from the API and find the one we need
    fetchConversations().then((apiConvs) => {
      const target = apiConvs.find((c) => c.id === conversationId)
      if (target) {
        const conv = fromAPIConversation(target)
        conv.isActive = true // mark as active since we're navigating to it

        // Build member list from registered users (best-effort).
        // Use getState() to avoid stale closure captures of currentUser.
        const storeState = useChatStore.getState()
        const registeredUsers = storeState.registeredUsers
        const storeCurrentUser = storeState.currentUser
        const otherMember = registeredUsers.find(
          (u) => u.id === target.user2_id,
        )
        if (otherMember) {
          conv.members = [
            storeCurrentUser,
            {
              id: otherMember.id,
              name: otherMember.name,
              email: otherMember.email || "",
              avatar: otherMember.avatar,
              color: userColorFromId(otherMember.id),
              status: otherMember.status,
              role: "Member",
            },
          ]
          conv.onlineCount = otherMember.status === "online" ? 1 : 0
        }

        // Add to store
        useChatStore.setState((prev: { conversations: Conversation[] }) => ({
          conversations: [...prev.conversations, conv],
        }))

        // Set as active
        setConversation(conversationId)
      } else if (conversationId) {
        // Conversation not found in the API — it might be a new one
        // from a room_ready event that hasn't synced yet.
        setConversation(conversationId)
      }
    })
  }, [
    currentUser.id,
    conversationId,
    isChatOpen,
    activeConvId,
    setConversation,
    openChat,
  ])

  // ─── Render ───────────────────────────────────────────────────────
  return (
    <div className="h-screen w-screen overflow-hidden bg-chat-bg flex">
      {/* Workspace Bar — always visible */}
      <WorkspaceBar />

      {/* Chat overlay auto-opens when isChatOpen is set by the effect */}
      <ChatOverlay />
    </div>
  )
}
