"use client"

import { useMemo, useState, useCallback } from "react"
import { useChatStore } from "./ChatStore"
import { blockUser, unblockUser } from "../../lib/chat"
import type { ChatUser } from "./ChatData"

// ─── SVG Icons ──────────────────────────────────────────────────────────

function ArrowLeftIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M19 12H5M12 19l-7-7 7-7" />
    </svg>
  )
}

function MessageIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
    </svg>
  )
}

function MailIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect x="2" y="4" width="20" height="16" rx="2" />
      <path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7" />
    </svg>
  )
}

function CalendarIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
      <line x1="16" x2="16" y1="2" y2="6" />
      <line x1="8" x2="8" y1="2" y2="6" />
      <line x1="3" x2="21" y1="10" y2="10" />
    </svg>
  )
}

function ShieldIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
    </svg>
  )
}

function BlockIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="10" />
      <line x1="4.93" y1="4.93" x2="19.07" y2="19.07" />
    </svg>
  )
}

function UnlockIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect x="5" y="11" width="14" height="10" rx="2" />
      <circle cx="12" cy="16" r="1" />
      <path d="M8 11V7a4 4 0 0 1 8 0" />
    </svg>
  )
}

// ─── Color helpers ──────────────────────────────────────────────────────

function userColorFromId(id: string): string {
  const colors = ["#06D6A0", "#ED4245", "#57F287", "#FEE75C", "#EB459E", "#1ABC9C", "#9B59B6", "#3498DB", "#E67E22", "#00BCD4"]
  let hash = 0
  for (let i = 0; i < id.length; i++) {
    hash = id.charCodeAt(i) + ((hash << 5) - hash)
  }
  return colors[Math.abs(hash) % colors.length]
}

// ─── Component ──────────────────────────────────────────────────────────

export default function UserProfileView() {
  const { selectedProfileUserId, setSelectedProfileUser, users, currentUser, registeredUsers, startConversationWith } = useChatStore()
  const [isBlocked, setIsBlocked] = useState(false)
  const [blockLoading, setBlockLoading] = useState(false)

  const profileUser = useMemo((): ChatUser | null => {
    if (!selectedProfileUserId) return null
    // First look in users (conversation members)
    const fromUsers = users.find((u) => u.id === selectedProfileUserId)
    if (fromUsers) return fromUsers
    // Fallback to registeredUsers (chat-service cache)
    const fromRegistered = registeredUsers.find((u) => u.id === selectedProfileUserId)
    if (fromRegistered) return fromRegistered
    // Last resort: derive a basic user object from the ID
    return {
      id: selectedProfileUserId,
      name: selectedProfileUserId.replace(/^user-/, "").replace(/[-_]/g, " ").replace(/\b\w/g, (c) => c.toUpperCase()),
      email: "",
      avatar: "",
      color: userColorFromId(selectedProfileUserId),
      status: "offline" as const,
      role: "Member",
    }
  }, [selectedProfileUserId, users, registeredUsers])

  if (!profileUser) return null

  const isSelf = profileUser.id === currentUser.id
  const color = userColorFromId(profileUser.id)
  const initials = profileUser.name
    .split(" ")
    .map((w) => w[0])
    .join("")
    .toUpperCase()
    .slice(0, 2)

  const statusColors = {
    online: "bg-chat-online",
    idle: "bg-yellow-400",
    dnd: "bg-red-500",
    offline: "bg-slate-600",
  }

  async function handleStartConversation() {
    if (!profileUser) return
    setSelectedProfileUser(null)
    await startConversationWith(profileUser.id, profileUser.name, profileUser.email)
  }

  const handleBlockToggle = useCallback(async () => {
    if (!profileUser || blockLoading) return
    setBlockLoading(true)
    try {
      if (isBlocked) {
        const res = await unblockUser(currentUser.id, profileUser.id)
        if (res) {
          setIsBlocked(false)
        }
      } else {
        const res = await blockUser(currentUser.id, profileUser.id)
        if (res) {
          setIsBlocked(true)
        }
      }
    } finally {
      setBlockLoading(false)
    }
  }, [profileUser, currentUser.id, isBlocked, blockLoading])

  return (
    <div className="h-full flex flex-col bg-chat-panel">
      {/* ── Header ──────────────────────────────────────────────────── */}
      <div className="flex items-center gap-2 px-4 py-3 border-b border-chat-border shrink-0">
        <button
          onClick={() => setSelectedProfileUser(null)}
          className="w-8 h-8 rounded-lg flex items-center justify-center text-chat-muted hover:text-white hover:bg-chat-card transition-all"
        >
          <ArrowLeftIcon />
        </button>
        <h2 className="text-sm font-semibold text-white">Profile</h2>
      </div>

      {/* ── Hero Section ────────────────────────────────────────────── */}
      <div className="relative h-32 bg-gradient-to-br from-chat-card to-chat-inner border-b border-chat-border shrink-0 overflow-hidden">
        <div className="absolute inset-0 bg-gradient-to-t from-chat-panel/60 via-transparent to-transparent" />
        <div className="absolute -bottom-10 left-5">
          <div className="relative inline-block">
            <div
              className="w-20 h-20 rounded-2xl flex items-center justify-center text-2xl font-bold text-white shadow-xl border-4 border-chat-panel"
              style={{ backgroundColor: color }}
            >
              {initials}
            </div>
            <span
              className={`absolute -bottom-0.5 -right-0.5 w-4 h-4 rounded-full border-[3px] border-chat-panel ${
                isSelf ? "bg-chat-accent" : statusColors[profileUser.status]
              }`}
            />
          </div>
        </div>
      </div>

      {/* ── Info Content ────────────────────────────────────────────── */}
      <div className="flex-1 overflow-y-auto px-5 py-4 space-y-5 scrollbar-thin">
        {/* Display name & status */}
        <div className="pt-2">
          <h3 className="text-lg font-bold text-white flex items-center gap-2">
            {profileUser.name}
            {isSelf && (
              <span className="text-[10px] bg-chat-accent/20 text-chat-accent px-2 py-0.5 rounded-full font-medium">
                You
              </span>
            )}
          </h3>
          <div className="flex items-center gap-2 mt-1">
            <span
              className={`w-2 h-2 rounded-full ${
                isSelf ? "bg-chat-accent" : statusColors[profileUser.status]
              }`}
            />
            <span className="text-xs text-chat-muted capitalize">
              {isSelf ? "Online" : profileUser.status}
            </span>
          </div>
        </div>

        {/* Role badge */}
        <div className="flex items-center gap-3 bg-chat-card rounded-xl px-4 py-3 border border-chat-border">
          <div className="w-9 h-9 rounded-lg bg-chat-accent/10 flex items-center justify-center text-chat-accent shrink-0">
            <ShieldIcon />
          </div>
          <div>
            <p className="text-xs text-chat-muted">Role</p>
            <p className="text-sm font-medium text-white capitalize">{profileUser.role}</p>
          </div>
        </div>

        {/* Email (from registered users cache if available) */}
        <div className="flex items-center gap-3 bg-chat-card rounded-xl px-4 py-3 border border-chat-border">
          <div className="w-9 h-9 rounded-lg bg-blue-500/10 flex items-center justify-center text-blue-400 shrink-0">
            <MailIcon />
          </div>
          <div className="min-w-0">
            <p className="text-xs text-chat-muted">Email</p>
            <p className="text-sm font-medium text-white truncate">
              {profileUser.email || `${profileUser.id}@demo.local`}
            </p>
          </div>
        </div>

        {/* User ID */}
        <div className="flex items-center gap-3 bg-chat-card rounded-xl px-4 py-3 border border-chat-border">
          <div className="w-9 h-9 rounded-lg bg-amber-500/10 flex items-center justify-center text-amber-400 shrink-0">
            #
          </div>
          <div className="min-w-0">
            <p className="text-xs text-chat-muted">User ID</p>
            <p className="text-xs font-mono text-slate-300 truncate">{profileUser.id}</p>
          </div>
        </div>

        {/* Bio (placeholder) */}
        <div className="flex items-start gap-3 bg-chat-card rounded-xl px-4 py-3 border border-chat-border">
          <div className="w-9 h-9 rounded-lg bg-purple-500/10 flex items-center justify-center text-purple-400 shrink-0 mt-0.5">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
              <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
            </svg>
          </div>
          <div>
            <p className="text-xs text-chat-muted">Bio</p>
            <p className="text-sm text-slate-300 mt-0.5 leading-relaxed">
              {isSelf
                ? "This is you! Edit your profile to add a bio."
                : `Hey there! I'm using RideShare.`}
            </p>
          </div>
        </div>

        {/* Joined date (derived from user ID timestamp) */}
        <div className="flex items-center gap-3 bg-chat-card rounded-xl px-4 py-3 border border-chat-border">
          <div className="w-9 h-9 rounded-lg bg-green-500/10 flex items-center justify-center text-green-400 shrink-0">
            <CalendarIcon />
          </div>
          <div>
            <p className="text-xs text-chat-muted">Joined</p>
            <p className="text-sm font-medium text-white">Recently</p>
          </div>
        </div>
      </div>

      {/* ── Action Footer ───────────────────────────────────────────── */}
      {!isSelf && (
        <div className="px-4 py-3 border-t border-chat-border shrink-0 space-y-2">
          <button
            onClick={handleStartConversation}
            className="w-full flex items-center justify-center gap-2 bg-chat-accent text-black font-semibold rounded-xl py-3 text-sm hover:opacity-90 transition-all"
          >
            <MessageIcon />
            Send Message
          </button>
          <button
            onClick={handleBlockToggle}
            disabled={blockLoading}
            className={`w-full flex items-center justify-center gap-2 rounded-xl py-3 text-sm font-medium transition-all ${
              isBlocked
                ? "bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20 border border-emerald-500/20"
                : "bg-red-500/10 text-red-400 hover:bg-red-500/20 border border-red-500/20"
            } disabled:opacity-50 disabled:cursor-not-allowed`}
          >
            {blockLoading ? (
              <svg className="animate-spin w-4 h-4" viewBox="0 0 24 24" fill="none">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
            ) : isBlocked ? (
              <UnlockIcon />
            ) : (
              <BlockIcon />
            )}
            {blockLoading ? (isBlocked ? "Unblocking..." : "Blocking...") : isBlocked ? "Unblock User" : "Block User"}
          </button>
        </div>
      )}
    </div>
  )
}
