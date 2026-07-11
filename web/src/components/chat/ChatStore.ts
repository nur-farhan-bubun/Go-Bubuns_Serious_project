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
  fromAPIConversation,
  fetchChatUsers,
  fetchUsersFromUserService,
  type ChatWSConnection,
  type ChatWSEnvelope,
  type ChatAPIUser,
} from "../../lib/chat"
import { getLocalRegisteredUsers } from "../../lib/auth"

// ─── Types ──────────────────────────────────────────────────────────────

export type WorkspaceView = "chat" | "users"

export interface ChatState {
  isChatOpen: boolean
  activeWorkspaceView: WorkspaceView
  activeWorkspaceId: string
  activeConversationId: string
  conversations: Conversation[]
  messages: ChatMessage[]
  assets: SharedAsset[]
  workspaces: Workspace[]
  users: ChatUser[]
  registeredUsers: ChatUser[] // All users from chat-service user cache
  currentUser: ChatUser
  wsStatus: "connecting" | "connected" | "disconnected" | "error"
  selectedProfileUserId: string | null // user ID whose profile is being viewed

  // Actions
  openChat: () => void
  closeChat: () => void
  toggleChat: () => void
  setWorkspaceView: (view: WorkspaceView) => void
  setWorkspace: (id: string) => void
  setConversation: (id: string) => void
  sendMessage: (content: string) => void
  addReaction: (messageId: string) => void
  setCurrentUserId: (userId: string, displayName?: string, userEmail?: string) => void
  startConversationWith: (userId: string, displayName: string, userEmail?: string) => Promise<void>
  joinConversation: (convId: string) => Promise<void>
  loadRegisteredUsers: () => Promise<void>
  setSelectedProfileUser: (userId: string | null) => void
  resetState: () => void
  updateUserPresence: (userId: string, status: "online" | "offline") => void
}

// ─── API integration helpers ────────────────────────────────────────────

let wsConnection: ChatWSConnection | null = null
let currentConvId = "" // Track which conversation we're viewing for race guards

// Tracks whether the current WebSocket connection has ever reached "connected"
// status. Reset on each connectToRoom call. Used to detect reconnection
// (second+ "connected" event) so we can sync missed messages from the API.
let hasConnectedOnce = false

// ─── Presence tracker ───────────────────────────────────────────────────
// Tracks online users across the app. Updates both `registeredUsers`
// (user directory) and `users` (conversation members) when presence events arrive.

function applyPresenceToStore(userId: string, status: "online" | "offline") {
  const state = useChatStore.getState()

  // Update registeredUsers (user directory)
  const newRegistered = state.registeredUsers.map((u) =>
    u.id === userId ? { ...u, status } : u,
  )

  // Update users (conversation members + current user)
  const newUsers = state.users.map((u) =>
    u.id === userId ? { ...u, status } : u,
  )

  // Update conversation member statuses and online counts
  const newConvs = state.conversations.map((conv) => {
    const updatedMembers = conv.members.map((m) =>
      m.id === userId ? { ...m, status } : m,
    )
    const onlineCount = updatedMembers.filter((m) => m.status === "online").length
    return { ...conv, members: updatedMembers, onlineCount }
  })

  useChatStore.setState({
    registeredUsers: newRegistered,
    users: newUsers,
    conversations: newConvs,
  })
}

/**
 * Load conversations from the API.
 * Handles both direct and group conversations.
 */
async function loadConversations(): Promise<Conversation[]> {
  try {
    const apiConvs = await fetchConversations()
    return apiConvs.map(fromAPIConversation)
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

// ─── Helpers ──────────────────────────────────────────────────────────
// Tracks message IDs received via WebSocket to avoid duplicates.
const wsMessageIds = new Set<string>()

/**
 * Add a conversation to the list, replacing any existing entry with the same ID.
 * This prevents duplicate key errors in the conversations list.
 */
function upsertConversation(convs: Conversation[], newConv: Conversation): Conversation[] {
  const idx = convs.findIndex((c) => c.id === newConv.id)
  if (idx >= 0) {
    const result = convs.slice()
    result[idx] = { ...convs[idx], ...newConv }
    return result
  }
  return [...convs, newConv]
}

/**
 * Merge an array of incoming conversations into the existing list,
 * deduplicating by ID (incoming replaces existing).
 */
function upsertConversations(existing: Conversation[], incoming: Conversation[]): Conversation[] {
  const map = new Map(existing.map((c) => [c.id, c]))
  for (const conv of incoming) {
    map.set(conv.id, conv)
  }
  return Array.from(map.values())
}

// ─── Store ──────────────────────────────────────────────────────────────

export const useChatStore = create<ChatState>((set, get) => ({
  isChatOpen: false,
  activeWorkspaceView: "chat",
  activeWorkspaceId: "",
  activeConversationId: "",
  conversations: [],
  messages: [],
  assets: [],
  workspaces: [],
  users: [],
  registeredUsers: [],
  currentUser: { id: "", name: "", email: "", avatar: "", color: "#5865F2", status: "online", role: "Member" },
  wsStatus: "disconnected",
  selectedProfileUserId: null,

  openChat: () => {
    set({ isChatOpen: true })

    const state = get()
    if (!state.currentUser.id) return

    // Refresh conversations from API
    loadConversations().then((convs) => {
      const s = get()
      if (!s.isChatOpen) return // user closed chat while loading

      if (convs.length > 0) {
        // Merge with existing conversations (don't replace, to avoid losing
        // conversations added by startConversationWith / joinConversation).
        set((prev) => ({
          conversations: upsertConversations(prev.conversations, convs),
        }))
      }
    })

    // Refresh the user directory so newly registered users appear
    get().loadRegisteredUsers()
  },

  closeChat: () => {
    if (wsConnection) {
      wsConnection.close()
      wsConnection = null
    }
    wsMessageIds.clear()
    set({ isChatOpen: false, wsStatus: "disconnected", selectedProfileUserId: null })
  },

  toggleChat: () => {
    const { isChatOpen } = get()
    if (isChatOpen) {
      get().closeChat()
    } else {
      get().openChat()
    }
  },

  setWorkspaceView: (view) => {
    set({ activeWorkspaceView: view })
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
      activeWorkspaceView: "chat",
      activeWorkspaceId: "",
      activeConversationId: "",
      conversations: [],
      messages: [],
      assets: [],
      workspaces: [],
      users: [],
      registeredUsers: [],
      currentUser: { id: "", name: "", email: "", avatar: "", color: "#5865F2", status: "online", role: "Member" },
      wsStatus: "disconnected",
      selectedProfileUserId: null,
    })
  },

  setCurrentUserId: (userId: string, displayName?: string, userEmail?: string) => {
    const runtimeUser: ChatUser = {
      id: userId,
      name: displayName || `User ${userId.slice(0, 6)}`,
      email: userEmail || "",
      avatar: "",
      color: "#5865F2",
      status: "online",
      role: "Member",
    }
    set((s) => ({
      currentUser: runtimeUser,
      users: [...s.users, runtimeUser],
    }))

    // Load registered users from chat-service cache
    get().loadRegisteredUsers()
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

  startConversationWith: async (userId, displayName, userEmail) => {
    const { currentUser } = get()
    if (userId === currentUser.id) return

    // Create or fetch existing conversation via the API
    const newConv = await createConversation(userId)
    if (!newConv) return

    // If the returned conversation ID already exists in the store,
    // just activate it — no duplicate entry needed.
    const alreadyExists = get().conversations.find((c) => c.id === newConv.id)
    if (alreadyExists) {
      set({ activeConversationId: newConv.id, activeWorkspaceView: "chat" })
      connectToRoom(currentUser.id, newConv.id)
      return
    }

    const otherUserColor = userColorFromId(userId)
    // Look up email from registered users if not passed
    const registeredUser = get().registeredUsers.find((u) => u.id === userId)
    const memberEmail = userEmail || registeredUser?.email || ""
    const conversation: Conversation = {
      id: newConv.id,
      workspaceId: "",
      name: displayName,
      avatar: displayName.charAt(0).toUpperCase(),
      lastMessage: "",
      lastTime: "just now",
      unread: 0,
      isActive: true,
      members: [
        currentUser,
        { id: userId, name: displayName, email: memberEmail, avatar: "", color: otherUserColor, status: "online", role: "Member" } as ChatUser,
      ],
      onlineCount: 1,
    }

    set((s) => ({
      conversations: upsertConversation(s.conversations, conversation),
      activeConversationId: conversation.id,
      activeWorkspaceView: "chat",
    }))

    connectToRoom(currentUser.id, conversation.id)
  },

  loadRegisteredUsers: async () => {
    const currentUserId = get().currentUser.id

    try {
      // First try fetching from chat-service API
      const apiUsers: ChatAPIUser[] = await fetchChatUsers()
      if (apiUsers.length > 0) {
        const chatUsers: ChatUser[] = apiUsers
          .filter((u) => u.user_id !== currentUserId) // exclude self
          .map((u) => ({
            id: u.user_id,
            name: u.display_name || u.user_id,
            email: u.email || "",
            avatar: u.avatar_url || "",
            color: userColorFromId(u.user_id),
            status: "offline" as const,
            role: "Member",
          }))
        set({ registeredUsers: chatUsers })
        return
      }
    } catch {
      // API unavailable — fall through to local fallback
    }

    // Second fallback: fetch directly from the user-service API.
    // This works if users have been registered in the DB (via login or register)
    // even if the Kafka → chat-service sync hasn't happened yet.
    try {
      const svcUsers: ChatAPIUser[] = await fetchUsersFromUserService()
      if (svcUsers.length > 0) {
        const chatUsers: ChatUser[] = svcUsers
          .filter((u) => u.user_id !== currentUserId)
          .map((u) => ({
            id: u.user_id,
            name: u.display_name || u.email.split("@")[0],
            email: u.email || "",
            avatar: u.avatar_url || "",
            color: userColorFromId(u.user_id),
            status: "offline" as const,
            role: "Member",
          }))
        if (chatUsers.length > 0) {
          set({ registeredUsers: chatUsers })
          return
        }
      }
    } catch {
      // API unavailable — fall through to local fallback
    }

    // Third fallback: use locally registered users from localStorage.
    // This ensures users can see each other even without a running backend.
    try {
      const localUsers = getLocalRegisteredUsers()
      const chatUsers: ChatUser[] = localUsers
        .filter((u) => u.id !== currentUserId) // exclude self
        .map((u) => ({
          id: u.id,
          name: u.name || u.email.split("@")[0],
          email: u.email || "",
          avatar: u.avatar_url || "",
          color: userColorFromId(u.id),
          status: "offline" as const,
          role: "Member",
        }))
      if (chatUsers.length > 0) {
        set({ registeredUsers: chatUsers })
      }
    } catch {
      // Silently fail
    }
  },

  joinConversation: async (convId: string) => {
    const state = get()

    // Open chat if not already open
    if (!state.isChatOpen) {
      set({ isChatOpen: true })
    }

    // If conversation already in store, just set it active
    if (get().conversations.find((c) => c.id === convId)) {
      get().setConversation(convId)
      return
    }

    let addedToStore = false

    // Fetch conversations from API to find the one we need
    try {
      const apiConvs = await fetchConversations()
      const target = apiConvs.find((c) => c.id === convId)
      if (target) {
        const conv = fromAPIConversation(target)
        // Build member list from registered users
        const s = get()
        const otherMember = s.registeredUsers.find(
          (u) => u.id === target.user2_id,
        )
        if (otherMember) {
          conv.members = [
            s.currentUser,
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
        // Add to store using upsert to prevent duplicates
        set((prev) => ({
          conversations: upsertConversation(prev.conversations, conv),
        }))
        addedToStore = true
      }
    } catch {
      // API unavailable — proceed with fallback
    }

    if (!addedToStore) {
      // Build a fallback conversation so the ChatFeed header shows correctly
      const s = get()
      const fallback: Conversation = {
        id: convId,
        workspaceId: "",
        name: "Chat",
        avatar: "#",
        lastMessage: "",
        lastTime: "",
        unread: 0,
        isActive: true,
        members: [s.currentUser],
        onlineCount: 0,
      }
      set((prev) => ({
        conversations: upsertConversation(prev.conversations, fallback),
      }))
    }

    // Set as active and connect to room
    get().setConversation(convId)
  },

  setSelectedProfileUser: (userId) => {
    set({ selectedProfileUserId: userId })
  },

  updateUserPresence: (userId, status) => {
    applyPresenceToStore(userId, status)
  },

  setWorkspace: () => {
    // No workspaces in current UI — kept for component compatibility
  },

  addReaction: () => {
    // Placeholder for emoji reactions
  },
}))

// ─── Helper ───────────────────────────────────────────────────────────────

function userColorFromId(id: string): string {
  const colors = ["#5865F2", "#ED4245", "#57F287", "#FEE75C", "#EB459E", "#1ABC9C", "#9B59B6", "#3498DB", "#E67E22", "#00BCD4"]
  let hash = 0
  for (let i = 0; i < id.length; i++) {
    hash = id.charCodeAt(i) + ((hash << 5) - hash)
  }
  return colors[Math.abs(hash) % colors.length]
}

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
      const state = useChatStore.getState()

      if (envelope.type === "chat_message") {
        const data = envelope.data as Record<string, unknown>
        const messageId = (data.message_id as string) || ""
        const senderId = (data.sender_id as string) || ""

        // Deduplicate: skip if we already have this message (optimistic add)
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

      // ── Handle presence events ─────────────────────────────────
      if (envelope.type === "presence") {
        const data = envelope.data as Record<string, unknown>
        const presenceUserId = (data.user_id as string) || ""
        const presenceStatus = (data.status as string) || "offline"

        if (!presenceUserId) return

        // Map to our status type
        const mappedStatus = presenceStatus === "online" ? "online" as const : "offline" as const

        // Update presence in both registeredUsers and users arrays
        applyPresenceToStore(presenceUserId, mappedStatus)
      }

      // ── Handle room_ready events ──────────────────────────────
      if (envelope.type === "room_ready") {
        const data = envelope.data as Record<string, unknown>
        const convId = (data.conversation_id as string) || ""
        if (convId && typeof window !== "undefined") {
          // Dispatch a custom event so the handshake hook picks it up
          window.dispatchEvent(
            new CustomEvent("chat:room_ready", { detail: { conversation_id: convId } }),
          )
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
      // Only match against messages with client-generated IDs ("msg-" prefix)
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
