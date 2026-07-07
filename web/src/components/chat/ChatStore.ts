"use client"

import { create } from "zustand"
import {
  CHAT_USERS,
  CURRENT_USER,
  WORKSPACES,
  getMessagesForConversation,
  getConversationsForWorkspace,
  getAssetsForConversation,
  type ChatMessage,
  type Conversation,
  type Workspace,
  type SharedAsset,
} from "./ChatData"

// ─── Types ──────────────────────────────────────────────────────────────

export interface ChatState {
  isChatOpen: boolean
  activeWorkspaceId: string
  activeConversationId: string
  conversations: Conversation[]
  messages: ChatMessage[]
  assets: SharedAsset[]
  users: typeof CHAT_USERS
  currentUser: typeof CURRENT_USER
  workspaces: Workspace[]

  // Actions
  openChat: () => void
  closeChat: () => void
  toggleChat: () => void
  setWorkspace: (id: string) => void
  setConversation: (id: string) => void
  sendMessage: (content: string) => void
  addReaction: (messageId: string) => void
}

// ─── Store ──────────────────────────────────────────────────────────────

export const useChatStore = create<ChatState>((set, get) => ({
  isChatOpen: false,
  activeWorkspaceId: "ws-icg",
  activeConversationId: "conv-1",
  conversations: getConversationsForWorkspace("ws-icg"),
  messages: getMessagesForConversation("conv-1"),
  assets: getAssetsForConversation("conv-1"),
  users: CHAT_USERS,
  currentUser: CURRENT_USER,
  workspaces: WORKSPACES,

  openChat: () => set({ isChatOpen: true }),
  closeChat: () => set({ isChatOpen: false }),
  toggleChat: () => set((s) => ({ isChatOpen: !s.isChatOpen })),

  setWorkspace: (id) => {
    const conversations = getConversationsForWorkspace(id)
    const firstConv = conversations[0]
    set({
      activeWorkspaceId: id,
      conversations,
      activeConversationId: firstConv?.id || "",
      messages: firstConv ? getMessagesForConversation(firstConv.id) : [],
      assets: firstConv ? getAssetsForConversation(firstConv.id) : [],
    })
  },

  setConversation: (id) => {
    set({
      activeConversationId: id,
      messages: getMessagesForConversation(id),
      assets: getAssetsForConversation(id),
    })
  },

  sendMessage: (content) => {
    const { activeConversationId, messages, currentUser } = get()
    if (!content.trim()) return

    const newMsg: ChatMessage = {
      id: `msg-${Date.now()}`,
      conversationId: activeConversationId,
      senderId: currentUser.id,
      content: content.trim(),
      timestamp: new Date().toISOString(),
      type: "text",
    }

    set({ messages: [...messages, newMsg] })

    // Update last message in conversations list
    set((s) => ({
      conversations: s.conversations.map((c) =>
        c.id === activeConversationId
          ? { ...c, lastMessage: content.trim(), lastTime: "just now", unread: 0 }
          : c,
      ),
    }))
  },

  addReaction: () => {
    // Placeholder for emoji reactions
    // Could toggle a like on a message
  },
}))
