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

// ─── Users ──────────────────────────────────────────────────────────────

export const CHAT_USERS: ChatUser[] = [
  { id: "u-me", name: "You", avatar: "/avatars/me.jpg", color: "#5865F2", status: "online", role: "Admin" },
  { id: "u-rachel", name: "Rachel Green", avatar: "/avatars/rachel.jpg", color: "#ED4245", status: "online", role: "Admin" },
  { id: "u-mike", name: "Mike Chen", avatar: "/avatars/mike.jpg", color: "#57F287", status: "online", role: "Member" },
  { id: "u-alex", name: "Alex Rivera", avatar: "/avatars/alex.jpg", color: "#FEE75C", status: "online", role: "Member" },
  { id: "u-sarah", name: "Sarah Kim", avatar: "/avatars/sarah.jpg", color: "#EB459E", status: "idle", role: "Member" },
  { id: "u-jordan", name: "Jordan Lee", avatar: "/avatars/jordan.jpg", color: "#1ABC9C", status: "online", role: "Admin" },
  { id: "u-taylor", name: "Taylor Swift", avatar: "/avatars/taylor.jpg", color: "#9B59B6", status: "offline", role: "Member" },
  { id: "u-chris", name: "Chris Evans", avatar: "/avatars/chris.jpg", color: "#3498DB", status: "dnd", role: "Member" },
  { id: "u-emma", name: "Emma Watson", avatar: "/avatars/emma.jpg", color: "#E67E22", status: "online", role: "Member" },
  { id: "u-david", name: "David Park", avatar: "/avatars/david.jpg", color: "#00BCD4", status: "offline", role: "Member" },
]

export const CURRENT_USER = CHAT_USERS[0]

// ─── Workspaces ─────────────────────────────────────────────────────────

export const WORKSPACES: Workspace[] = [
  { id: "ws-work", label: "Work", icon: "💼", color: "#5865F2" },
  { id: "ws-icg", label: "ICG", icon: "🎨", color: "#EB459E" },
  { id: "ws-sp", label: "SP", icon: "🚀", color: "#57F287" },
  { id: "ws-bff", label: "BFF", icon: "💜", color: "#9B59B6" },
]

// ─── Conversations ──────────────────────────────────────────────────────

export const CONVERSATIONS: Conversation[] = [
  {
    id: "conv-1",
    workspaceId: "ws-icg",
    name: "ICG chat",
    avatar: "IC",
    lastMessage: "@rwilsons Nice work on the deployment!",
    lastTime: "2m ago",
    unread: 3,
    isActive: true,
    members: [CHAT_USERS[0], CHAT_USERS[1], CHAT_USERS[2], CHAT_USERS[3], CHAT_USERS[4]],
    onlineCount: 4,
  },
  {
    id: "conv-2",
    workspaceId: "ws-icg",
    name: "design-sync",
    avatar: "DS",
    lastMessage: "Sarah: Updated the mockups 🎨",
    lastTime: "15m ago",
    unread: 0,
    isActive: false,
    members: [CHAT_USERS[0], CHAT_USERS[4], CHAT_USERS[6], CHAT_USERS[9]],
    onlineCount: 2,
  },
  {
    id: "conv-3",
    workspaceId: "ws-work",
    name: "engineering",
    avatar: "EN",
    lastMessage: "Jordan: PR is ready for review",
    lastTime: "1h ago",
    unread: 5,
    isActive: false,
    members: [CHAT_USERS[0], CHAT_USERS[2], CHAT_USERS[5], CHAT_USERS[7]],
    onlineCount: 3,
  },
  {
    id: "conv-4",
    workspaceId: "ws-work",
    name: "product-team",
    avatar: "PT",
    lastMessage: "Meeting at 3pm tomorrow",
    lastTime: "3h ago",
    unread: 0,
    isActive: false,
    members: [CHAT_USERS[0], CHAT_USERS[1], CHAT_USERS[3], CHAT_USERS[5]],
    onlineCount: 2,
  },
  {
    id: "conv-5",
    workspaceId: "ws-bff",
    name: "weekend-planning",
    avatar: "WP",
    lastMessage: "Emma: How about Napa Valley? 🍷",
    lastTime: "30m ago",
    unread: 7,
    isActive: false,
    members: [CHAT_USERS[0], CHAT_USERS[4], CHAT_USERS[6], CHAT_USERS[8], CHAT_USERS[9]],
    onlineCount: 3,
  },
  {
    id: "conv-6",
    workspaceId: "ws-bff",
    name: "book-club",
    avatar: "BC",
    lastMessage: "Next month's pick: Project Hail Mary 🚀",
    lastTime: "2h ago",
    unread: 1,
    isActive: false,
    members: [CHAT_USERS[0], CHAT_USERS[4], CHAT_USERS[6], CHAT_USERS[8]],
    onlineCount: 1,
  },
  {
    id: "conv-7",
    workspaceId: "ws-sp",
    name: "sprint-24",
    avatar: "S24",
    lastMessage: "Mike: Blocked on auth migration",
    lastTime: "45m ago",
    unread: 2,
    isActive: false,
    members: [CHAT_USERS[0], CHAT_USERS[2], CHAT_USERS[5], CHAT_USERS[7]],
    onlineCount: 3,
  },
  {
    id: "conv-8",
    workspaceId: "ws-icg",
    name: "random",
    avatar: "RN",
    lastMessage: "Check out this meme 😂",
    lastTime: "1d ago",
    unread: 0,
    isActive: false,
    members: [CHAT_USERS[0], CHAT_USERS[1], CHAT_USERS[2], CHAT_USERS[3], CHAT_USERS[4], CHAT_USERS[5], CHAT_USERS[6], CHAT_USERS[7], CHAT_USERS[8], CHAT_USERS[9]],
    onlineCount: 5,
  },
]

// ─── Messages ───────────────────────────────────────────────────────────

let msgCounter = 0
function msg(id: string, convId: string, senderIdx: number, content: string, minsAgo: number, type: "text" | "system" | "call" = "text", attachments?: { name: string; size: string }[]): ChatMessage {
  msgCounter++
  const sender = senderIdx === 0 ? CURRENT_USER : CHAT_USERS[senderIdx]
  return {
    id: id || `msg-${msgCounter}`,
    conversationId: convId,
    senderId: sender.id,
    content,
    timestamp: new Date(Date.now() - minsAgo * 60000).toISOString(),
    type,
    attachments,
  }
}

export const MESSAGES: Record<string, ChatMessage[]> = {
  "conv-1": [
    msg("m1", "conv-1", 1, "Hey team, great work on the new feature! 🎉", 60),
    msg("m2", "conv-1", 2, "Thanks Rachel! The backend is running smoothly now.", 55),
    msg("m3", "conv-1", 3, "Frontend looks clean too. Deploy preview is up.", 50),
    msg("m4", "conv-1", 0, "I'll review the PR this afternoon 👀", 45),
    msg("m5", "conv-1", 4, "Should we add more test coverage before merging?", 40),
    msg("m6", "conv-1", 0, "Good call, Sarah. Let's aim for 90% coverage.", 35),
    msg("m7", "conv-1", 0, "Added @rwilsons Nice work on the deployment!", 2, "system"),
    msg("m8", "conv-1", 1, "@rwilsons Nice work on the deployment! 🚀", 2),
  ],
  "conv-3": [
    msg("m20", "conv-3", 5, "PR #342 is ready for review — auth service refactor", 60),
    msg("m21", "conv-3", 7, "I'll take a look after standup", 50),
    msg("m22", "conv-3", 2, "Tests are passing locally, need to check CI", 40),
    msg("m23", "conv-3", 5, "CI is green! Ready whenever you are.", 30),
    msg("m24", "conv-3", 0, "Great work @jordan! Reviewed and approved ✅", 1, "system"),
    msg("m25", "conv-3", 5, "Thanks! Merging now.", 0),
  ],
  "conv-5": [
    msg("m30", "conv-5", 8, "Who's free this weekend? 🎉", 180),
    msg("m31", "conv-5", 6, "I'm in! Any ideas?", 170),
    msg("m32", "conv-5", 4, "How about Napa Valley? 🍷", 160),
    msg("m33", "conv-5", 8, "Emma: How about Napa Valley? 🍷", 30, "system"),
    msg("m34", "conv-5", 8, "Ooh yes! I know a great vineyard", 25),
    msg("m35", "conv-5", 0, "I can drive! Minivan with room for 5", 20),
    msg("m36", "conv-5", 9, "Count me in! 🚗", 15),
  ],
  "conv-7": [
    msg("m40", "conv-7", 2, "Facing issues with the auth migration script", 90),
    msg("m41", "conv-7", 5, "What's the error?", 80),
    msg("m42", "conv-7", 2, "Token validation failing on the new endpoint", 70),
    msg("m43", "conv-7", 5, "I think the JWT secret needs to be rotated", 60),
    msg("m44", "conv-7", 7, "Already created a ticket for that", 50),
    msg("m45", "conv-7", 0, "Let's sync on this after lunch", 40),
    msg("m46", "conv-7", 2, "Mike: Blocked on auth migration", 45, "system"),
  ],
  "conv-8": [
    msg("m50", "conv-8", 3, "Check out this meme 😂", 1440),
    msg("m51", "conv-8", 5, "LMAO that's too accurate", 1430),
    msg("m52", "conv-8", 4, "Saving this for later", 1420),
    msg("m53", "conv-8", 1, "😂😂😂", 1410),
  ],
}

// ─── Shared Assets ──────────────────────────────────────────────────────

export const SHARED_ASSETS: Record<string, SharedAsset[]> = {
  "conv-1": [
    { id: "a1", conversationId: "conv-1", type: "photo", url: "https://images.unsplash.com/photo-1611162617474-5b21e879e113?w=200&h=200&fit=crop", name: "Dashboard v2" },
    { id: "a2", conversationId: "conv-1", type: "photo", url: "https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=200&h=200&fit=crop", name: "Analytics" },
    { id: "a3", conversationId: "conv-1", type: "photo", url: "https://images.unsplash.com/photo-1460925895917-afdab827c52f?w=200&h=200&fit=crop", name: "KPI Dashboard" },
    { id: "a4", conversationId: "conv-1", type: "file", url: "#", name: "Q4_Strategy_Doc.pdf", size: "2.4 MB" },
    { id: "a5", conversationId: "conv-1", type: "file", url: "#", name: "Sprint_23_Retro.md", size: "12 KB" },
    { id: "a6", conversationId: "conv-1", type: "file", url: "#", name: "Architecture_Diagram.png", size: "4.1 MB" },
    { id: "a7", conversationId: "conv-1", type: "link", url: "#", name: "Figma Prototype", size: "External" },
  ],
  "conv-3": [
    { id: "a10", conversationId: "conv-3", type: "photo", url: "https://images.unsplash.com/photo-1555949963-aa79dcee981c?w=200&h=200&fit=crop", name: "System Diagram" },
    { id: "a11", conversationId: "conv-3", type: "photo", url: "https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=200&h=200&fit=crop", name: "Performance" },
    { id: "a12", conversationId: "conv-3", type: "file", url: "#", name: "API_Spec_v3.yaml", size: "45 KB" },
    { id: "a13", conversationId: "conv-3", type: "file", url: "#", name: "Migration_Plan.md", size: "28 KB" },
  ],
  "conv-5": [
    { id: "a20", conversationId: "conv-5", type: "photo", url: "https://images.unsplash.com/photo-1539776180221-e7221c2c63a7?w=200&h=200&fit=crop", name: "Napa Valley" },
    { id: "a21", conversationId: "conv-5", type: "photo", url: "https://images.unsplash.com/photo-1516594915697-87eb3b1c14ea?w=200&h=200&fit=crop", name: "Vineyard" },
    { id: "a22", conversationId: "conv-5", type: "photo", url: "https://images.unsplash.com/photo-1506377247377-2a5b3b417ebb?w=200&h=200&fit=crop", name: "Wine Tasting" },
    { id: "a23", conversationId: "conv-5", type: "file", url: "#", name: "Weekend_Itinerary.pdf", size: "1.2 MB" },
  ],
}

// ─── Helpers ────────────────────────────────────────────────────────────

export function getConversationsForWorkspace(workspaceId: string): Conversation[] {
  return CONVERSATIONS.filter((c) => c.workspaceId === workspaceId)
}

export function getMessagesForConversation(conversationId: string): ChatMessage[] {
  return MESSAGES[conversationId] || []
}

export function getAssetsForConversation(conversationId: string): SharedAsset[] {
  return SHARED_ASSETS[conversationId] || []
}

export function getUserById(userId: string): ChatUser | undefined {
  return CHAT_USERS.find((u) => u.id === userId)
}
