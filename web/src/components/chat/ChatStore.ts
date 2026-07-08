"use client"

import { create } from "zustand"
import {
  type ChatMessage,
  type Conversation,
  type Workspace,
  type SharedAsset,
  type ChatUser,
} from "./ChatData"
import {
  fetchConversations,
  fetchMessages,
  sendMessageAPI,
  createConversation,
  connectChatWS,
  fromAPIMessage,
  type ChatWSConnection,
  type ChatWSEnvelope,
} from "../../lib/chat"

// ─── Types ──────────────────────────────────────────────────────────────

export interface ChatState {
  isChatOpen: boolean
  activeWorkspaceId: string
  activeConversationId: string
  conversations: Conversation[]
  messages: ChatMessage[]
  assets: SharedAsset[]
  workspaces: Workspace[]
  users: ChatUser[]
  currentUser: ChatUser
  wsStatus: "connecting" | "connected" | "disconnected" | "error"

  // Actions
  openChat: () => void
  closeChat: () => void
  toggleChat: () => void
  setWorkspace: (id: string) => void
  setConversation: (id: string) => void
  sendMessage: (content: string) => void
  addReaction: (messageId: string) => void
  setCurrentUserId: (userId: string, displayName?: string) => void
  resetState: () => void
}

// ─── API integration helpers ────────────────────────────────────────────

let wsConnection: ChatWSConnection | null = null
let currentConvId = "" // Track which conversation we're viewing for race guards

// Tracks whether the current WebSocket connection has ever reached "connected"
// status. Reset on each connectToRoom call. Used to detect reconnection
// (second+ "connected" event) so we can sync missed messages from the API.
let hasConnectedOnce = false

/**
 * Load conversations from the API.
 */
async function loadConversations(): Promise<Conversation[]> {
  try {
    const apiConvs = await fetchConversations()
    return apiConvs.map((c, i) => ({
      id: c.id,
      workspaceId: "",
      name: `Chat #${i + 1}`,
      avatar: `C${i + 1}`,
      lastMessage: "",
      lastTime: new Date(c.created_at).toLocaleString(),
      unread: 0,
      isActive: i === 0,
      members: [],
      onlineCount: 0,
    }))
  } catch (_) {
    return []
  }
}

/**
 * Load messages for a conversation from the API.
 */
async function loadMessages(conversationId: string, currentUserId: string): Promise<ChatMessage[]> {
  try {
    const apiMsgs = await fetchMessages(conversationId)
    if (apiMsgs.length > 0) {
      return apiMsgs.map((m) => fromAPIMessage(m, currentUserId))
    }
  } catch (_) {
    // API unavailable
  }
  return []
}

// ─── WS message IDs set for deduplication ──────────────────────────────
// Tracks message IDs received via WebSocket to avoid duplicates.
const wsMessageIds = new Set<string>()

// ─── Store ──────────────────────────────────────────────────────────────

export const useChatStore = create<ChatState>((set, get) => ({
  isChatOpen: false,
  activeWorkspaceId: "",
  activeConversationId: "",
  conversations: [],
  messages: [],
  assets: [],
  workspaces: [],
  users: [],
  currentUser: { id: "", name: "", avatar: "", color: "#5865F2", status: "online", role: "Member" },
  wsStatus: "disconnected",

  openChat: () => {
    set({ isChatOpen: true })

    const state = get()
    if (!state.currentUser.id) return

    // Refresh conversations from API, then auto-create one if none exist
    loadConversations().then(async (convs) => {
      const s = get()
      if (!s.isChatOpen) return // user closed chat while loading

      // If no conversations exist in the backend, auto-create a demo one
      if (convs.length === 0) {
        const demoUserID = "user-alice"
        const newConv = await createConversation(demoUserID)
        if (newConv) {
          const targetId = newConv.id
          const conversation: Conversation = {
            id: newConv.id,
            workspaceId: "",
            name: "Chat with Alice",
            avatar: "A",
            lastMessage: "",
            lastTime: "just now",
            unread: 0,
            isActive: true,
            members: [s.currentUser, {
              id: demoUserID,
              name: "Alice",
              avatar: "",
              color: "#f59e0b",
              status: "online",
              role: "Member",
            }],
            onlineCount: 1,
          }
          set({
            conversations: [conversation],
            activeConversationId: targetId,
          })
          connectToRoom(s.currentUser.id, targetId)
          loadMessages(targetId, s.currentUser.id).then((msgs) => {
            if (get().activeConversationId !== targetId) return
            set({ messages: msgs })
          })
          return
        }
      }

      const firstConv = convs[0]
      if (firstConv) {
        set({
          conversations: convs,
          activeConversationId: firstConv.id,
        })
        const targetId = firstConv.id
        connectToRoom(s.currentUser.id, targetId)
        loadMessages(targetId, s.currentUser.id).then((msgs) => {
          if (get().activeConversationId !== targetId) return // stale
          set({ messages: msgs })
        })
      }
    })
  },

  closeChat: () => {
    if (wsConnection) {
      wsConnection.close()
      wsConnection = null
    }
    wsMessageIds.clear()
    set({ isChatOpen: false, wsStatus: "disconnected" })
  },

  toggleChat: () => {
    const { isChatOpen } = get()
    if (isChatOpen) {
      get().closeChat()
    } else {
      get().openChat()
    }
  },

  resetState: () => {
    if (wsConnection) {
      wsConnection.close()
      wsConnection = null
    }
    wsMessageIds.clear()
    hasConnectedOnce = false
    set({
      isChatOpen: false,
      activeWorkspaceId: "",
      activeConversationId: "",
      conversations: [],
      messages: [],
      assets: [],
      workspaces: [],
      users: [],
      currentUser: { id: "", name: "", avatar: "", color: "#5865F2", status: "online", role: "Member" },
      wsStatus: "disconnected",
    })
  },

  setCurrentUserId: (userId: string, displayName?: string) => {
    const runtimeUser: ChatUser = {
      id: userId,
      name: displayName || `User ${userId.slice(0, 6)}`,
      avatar: "",
      color: "#5865F2",
      status: "online",
      role: "Member",
    }
    set((s) => ({
      currentUser: runtimeUser,
      users: [...s.users, runtimeUser],
    }))
  },

  setConversation: (id) => {
    const { currentUser } = get()
    currentConvId = id
    set({ activeConversationId: id })

    // Load messages for this conversation
    loadMessages(id, currentUser.id).then((msgs) => {
      if (currentConvId !== id) return // stale — user switched again
      set({ messages: msgs })
    })

    // Reconnect WebSocket to the new room
    connectToRoom(currentUser.id, id)
  },

  sendMessage: (content) => {
    const { activeConversationId, messages, currentUser } = get()
    if (!content.trim() || !activeConversationId) return

    const newMsg: ChatMessage = {
      id: `msg-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`,
      conversationId: activeConversationId,
      senderId: currentUser.id,
      content: content.trim(),
      timestamp: new Date().toISOString(),
      type: "text",
    }

    // Optimistically add to local state
    set({ messages: [...messages, newMsg] })

    // Try sending via REST API (fire and forget)
    sendMessageAPI(activeConversationId, content.trim()).catch(() => {
      // API failed — local message stays (offline mode)
    })

    // Send via WebSocket for real-time delivery to other clients
    if (wsConnection?.isConnected()) {
      wsConnection.send({
        type: "chat_message",
        room_id: activeConversationId,
        data: { content: content.trim() },
      })
    }

    // Update last message in conversations list
    set((s) => ({
      conversations: s.conversations.map((c) =>
        c.id === activeConversationId
          ? { ...c, lastMessage: content.trim(), lastTime: "just now", unread: 0 }
          : c,
      ),
    }))
  },

  setWorkspace: () => {
    // No workspaces in current UI — kept for component compatibility
  },

  addReaction: () => {
    // Placeholder for emoji reactions
  },
}))

// ─── WebSocket helper ───────────────────────────────────────────────────

function connectToRoom(userId: string, roomId: string) {
  if (wsConnection) {
    wsConnection.close()
    wsConnection = null
  }

  // Clear the dedup set from the previous connection so old IDs don't
  // accumulate indefinitely.
  wsMessageIds.clear()

  hasConnectedOnce = false

  wsConnection = connectChatWS(
    userId,
    roomId,
    (envelope: ChatWSEnvelope) => {
      if (envelope.type === "chat_message") {
        const data = envelope.data as Record<string, unknown>
        const messageId = (data.message_id as string) || ""
        const senderId = (data.sender_id as string) || ""

        // Deduplicate: skip if we already have this message (optimistic add)
        const state = useChatStore.getState()
        if (messageId && wsMessageIds.has(messageId)) return
        if (messageId) wsMessageIds.add(messageId)

        // Skip messages from self (we already optimistically added them)
        if (senderId === state.currentUser.id) return

        // Guard against stale room — only add if viewing this conversation
        if (state.activeConversationId !== roomId) return

        const newMsg: ChatMessage = {
          id: messageId || `ws-${Date.now()}`,
          conversationId: (data.conversation_id as string) || roomId,
          senderId: senderId,
          content: (data.content as string) || "",
          timestamp: (data.created_at as string) || new Date().toISOString(),
          type: "text",
        }
        // Only add if not a duplicate by checking IDs
        if (!state.messages.some((m) => m.id === newMsg.id)) {
          useChatStore.setState({ messages: [...state.messages, newMsg] })
        }
      }
    },
    (status) => {
      useChatStore.setState({ wsStatus: status })

      // Detect reconnection: first "connected" is the initial connect;
      // subsequent "connected" (after a disconnect) means we reconnected.
      if (status === "connected") {
        if (hasConnectedOnce) {
          const s = useChatStore.getState()
          if (s.activeConversationId) {
            syncMissedOnReconnect(s.activeConversationId, s.currentUser.id)
          }
        }
        hasConnectedOnce = true
      }
    },
  )
}

// ─── Message sync on reconnect ──────────────────────────────────────────

/**
 * Called after a WebSocket reconnection. Fetches the latest messages from
 * the API for the active conversation and merges any new ones into the
 * store, deduplicating by message ID.
 *
 * Handles optimistic messages: if a fetched message has the same sender
 * and content as an existing message but a different ID (e.g. the user
 * sent while offline and REST persisted it with a real UUID), it replaces
 * the optimistic copy instead of creating a duplicate.
 */
async function syncMissedOnReconnect(conversationId: string, currentUserId: string) {
  try {
    const apiMsgs = await fetchMessages(conversationId)
    if (!apiMsgs.length) return

    const state = useChatStore.getState()
    // Guard against stale sync — user may have switched conversations
    // while we were fetching.
    if (state.activeConversationId !== conversationId) return

    const fetchedMsgs = apiMsgs.map((m) => fromAPIMessage(m, currentUserId))
    const existingIds = new Set(state.messages.map((m) => m.id))

    let mergedMessages = state.messages.slice()
    let changed = false

    for (const msg of fetchedMsgs) {
      if (existingIds.has(msg.id)) continue // already have this exact message

      // Check if this fetched message is the server-confirmed version of an
      // optimistically-added local message (same sender + content).
      // Only match against messages with client-generated IDs ("msg-..." prefix)
      // to avoid incorrectly replacing a different server message with the same content.
      const optimisticIdx = mergedMessages.findIndex(
        (m) => m.id.startsWith("msg-") && m.senderId === msg.senderId && m.content === msg.content,
      )

      if (optimisticIdx !== -1) {
        // Replace the optimistic version with the server-confirmed one.
        mergedMessages[optimisticIdx] = msg
      } else {
        // Genuinely new message.
        mergedMessages.push(msg)
      }
      changed = true

      // Add to dedup set so WS broadcasts are skipped.
      wsMessageIds.add(msg.id)
    }

    if (!changed) return

    // Sort chronologically.
    mergedMessages.sort(
      (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime(),
    )

    useChatStore.setState({ messages: mergedMessages })
  } catch (_) {
    // API unavailable — can't sync. Messages will be visible once the
    // WebSocket delivers them in real time.
  }
}
