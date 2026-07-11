"use client"

import { useState, useRef, useEffect, useMemo } from "react"
import { motion } from "framer-motion"
import { useChatStore } from "./ChatStore"
import Image from "next/image"

// ─── Helpers ────────────────────────────────────────────────────────────

function formatTime(iso: string) {
  const d = new Date(iso)
  return d.toLocaleTimeString("en-US", { hour: "numeric", minute: "2-digit", hour12: true })
}

function formatDate(iso: string) {
  const d = new Date(iso)
  const today = new Date()
  const yesterday = new Date(today)
  yesterday.setDate(yesterday.getDate() - 1)

  if (d.toDateString() === today.toDateString()) return "Today"
  if (d.toDateString() === yesterday.toDateString()) return "Yesterday"
  return d.toLocaleDateString("en-US", { day: "numeric", month: "short", year: "numeric" })
}

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return "just now"
  if (mins < 60) return `${mins}m ago`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.floor(hours / 24)}d ago`
}

// ─── SVG Icons ──────────────────────────────────────────────────────────

function PhoneIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z" />
    </svg>
  )
}

function VideoIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <polygon points="23 7 16 12 23 17 23 7" />
      <rect x="1" y="5" width="15" height="14" rx="2" ry="2" />
    </svg>
  )
}

function PinIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 2C7.6 2 4 5.6 4 10c0 5.5 7.3 11.5 7.6 11.7.2.2.6.2.8 0C12.7 21.5 20 15.5 20 10c0-4.4-3.6-8-8-8z" />
      <circle cx="12" cy="10" r="3" />
    </svg>
  )
}

function AttachIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21.44 11.05l-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48" />
    </svg>
  )
}

function SendIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M22 2 11 13" />
      <path d="m22 2-7 20-4-9-9-4 20-7z" />
    </svg>
  )
}

function EmojiIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="10" />
      <path d="M8 14s1.5 2 4 2 4-2 4-2" />
      <line x1="9" x2="9.01" y1="9" y2="9" />
      <line x1="15" x2="15.01" y1="9" y2="9" />
    </svg>
  )
}

function DotsIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="1" />
      <circle cx="19" cy="12" r="1" />
      <circle cx="5" cy="12" r="1" />
    </svg>
  )
}

// ─── Helpers ────────────────────────────────────────────────────────────

/**
 * Derives a stable color from a user ID string.
 */
function userIdColor(id: string): string {
  const colors = ["#5865F2", "#ED4245", "#57F287", "#FEE75C", "#EB459E", "#1ABC9C", "#9B59B6", "#3498DB", "#E67E22", "#00BCD4"]
  let hash = 0
  for (let i = 0; i < id.length; i++) {
    hash = id.charCodeAt(i) + ((hash << 5) - hash)
  }
  return colors[Math.abs(hash) % colors.length]
}

/**
 * Derives initials from a user ID.
 */
function userIdInitials(id: string): string {
  const parts = id.split(/[-_]/)
  if (parts.length >= 2) {
    return parts.slice(1).map((p) => p[0]?.toUpperCase() || "").join("").slice(0, 2)
  }
  return id.slice(0, 2).toUpperCase()
}

/**
 * Derives a display name from a user ID.
 */
function userIdDisplayName(id: string): string {
  const parts = id.split(/[-_]/)
  if (parts.length >= 2) {
    return parts.slice(1).map((p) => p.charAt(0).toUpperCase() + p.slice(1)).join(" ").slice(0, 20)
  }
  return `User ${id.slice(0, 6)}`
}

// ─── Message Bubble ─────────────────────────────────────────────────────

function MessageBubble({ msg, isOwn, onUserClick }: { msg: { id: string; senderId: string; content: string; timestamp: string; type: string }; isOwn: boolean; onUserClick: (userId: string) => void }) {
  const color = userIdColor(msg.senderId)
  const initials = userIdInitials(msg.senderId)

  if (msg.type === "system") {
    // System message — merged style like Slack/Discord
    return (
      <motion.div
        initial={{ opacity: 0, y: 6 }}
        animate={{ opacity: 1, y: 0 }}
        className="flex items-center gap-2 px-1 py-1.5"
      >
        <div className="w-px h-4 bg-chat-border" />
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="#8A8D93" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <path d="M18 20V10M12 20V4M6 20v-6" />
        </svg>
        <span className="text-xs text-chat-muted flex-1">{msg.content}</span>
        <span className="text-[10px] text-chat-muted">{timeAgo(msg.timestamp)}</span>
      </motion.div>
    )
  }

  if (msg.type === "call") {
    return (
      <motion.div
        initial={{ opacity: 0, y: 6 }}
        animate={{ opacity: 1, y: 0 }}
        className="flex items-center gap-3 bg-chat-card p-4 rounded-xl mx-1 my-2"
      >
        <div className="w-10 h-10 rounded-full bg-chat-accent/10 flex items-center justify-center text-chat-accent">
          <PhoneIcon />
        </div>
        <div className="flex-1">
          <p className="text-sm font-medium text-white">{msg.content}</p>
          <p className="text-[11px] text-chat-muted">{timeAgo(msg.timestamp)}</p>
        </div>
        <button className="bg-chat-accent text-black rounded-full px-5 py-2 text-sm font-semibold hover:opacity-90 transition-opacity">
          Join
        </button>
      </motion.div>
    )
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      className={`flex gap-3 group px-1 py-1 ${isOwn ? "flex-row-reverse" : ""}`}
    >
      {/* Avatar Column — clickable to view profile */}
      <div className="shrink-0 mt-0.5">
        {isOwn ? (
          <div
            onClick={() => onUserClick(msg.senderId)}
            className="w-8 h-8 rounded-full bg-chat-accent flex items-center justify-center text-[10px] font-bold text-black cursor-pointer hover:opacity-80 transition-opacity"
          >
            Y
          </div>
        ) : (
          <div
            onClick={() => onUserClick(msg.senderId)}
            className="w-8 h-8 rounded-full flex items-center justify-center text-[10px] font-bold text-white shadow-sm cursor-pointer hover:opacity-80 transition-opacity"
            style={{ backgroundColor: color }}
          >
            {initials}
          </div>
        )}
      </div>

      {/* Content */}
      <div className={`flex-1 min-w-0 ${isOwn ? "items-end" : ""}`}>
        <div className={`flex items-center gap-2 mb-0.5 ${isOwn ? "flex-row-reverse" : ""}`}>
          <button
            onClick={() => onUserClick(msg.senderId)}
            className="text-[12px] font-semibold hover:underline cursor-pointer transition-colors"
            style={{ color: isOwn ? "#E6FF7B" : color }}
          >
            {userIdDisplayName(msg.senderId)}
          </button>
          <span className="text-[10px] text-chat-muted">{formatTime(msg.timestamp)}</span>
        </div>
        <div className={`text-sm text-slate-200 leading-relaxed ${isOwn ? "text-right" : ""}`}>
          {msg.content}
        </div>
      </div>
    </motion.div>
  )
}

// ─── Component ──────────────────────────────────────────────────────────

export default function ChatFeed() {
  const { activeConversationId, messages, conversations, sendMessage, currentUser, setSelectedProfileUser } = useChatStore()
  const [input, setInput] = useState("")
  const listRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)



  const conv = conversations.find((c) => c.id === activeConversationId)
  const firstMember = conv?.members[1]

  // Auto-scroll to bottom on new messages
  useEffect(() => {
    if (listRef.current) {
      listRef.current.scrollTop = listRef.current.scrollHeight
    }
  }, [messages.length])

  // Focus input when conversation changes
  useEffect(() => {
    if (activeConversationId) {
      setTimeout(() => inputRef.current?.focus(), 100)
    }
  }, [activeConversationId])

  function handleSend(e: React.FormEvent) {
    e.preventDefault()
    if (!input.trim()) return
    sendMessage(input)
    setInput("")
  }

  // Group messages by date
  const groupedMessages = useMemo(() => {
    const groups: { date: string; messages: typeof messages }[] = []
    let lastDate: string | null = null

    for (const msg of messages) {
      const dateLabel = formatDate(msg.timestamp)
      if (dateLabel !== lastDate) {
        groups.push({ date: dateLabel, messages: [] })
        lastDate = dateLabel
      }
      groups[groups.length - 1].messages.push(msg)
    }
    return groups
  }, [messages])

  return (
    <div className="flex flex-col flex-1 min-w-0 bg-chat-panel h-full">
      {/* ── Header ──────────────────────────────────────────────────── */}
      <div className="flex items-center gap-3 px-5 py-3.5 border-b border-chat-border shrink-0">
        <div className="flex items-center gap-3 flex-1 min-w-0">
          {/* Back button for small screens */}
          <button className="lg:hidden w-8 h-8 rounded-lg flex items-center justify-center text-chat-muted hover:text-white hover:bg-chat-card transition-all">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M19 12H5M12 19l-7-7 7-7" />
            </svg>
          </button>

          {conv && firstMember ? (
            <>
              <button
                onClick={() => setSelectedProfileUser(firstMember.id)}
                className="w-9 h-9 rounded-2xl flex items-center justify-center text-xs font-bold text-white shrink-0 hover:opacity-80 transition-opacity cursor-pointer"
                style={{ backgroundColor: firstMember.color }}
              >
                {conv.avatar}
              </button>
              <div className="min-w-0">
              <button
                onClick={() => setSelectedProfileUser(firstMember.id)}
                className="text-sm font-semibold text-white truncate hover:underline transition-all text-left"
              >
                {conv.name}
              </button>
              {firstMember?.email && (
                <p className="text-[9px] text-chat-muted/70 truncate leading-tight">{firstMember.email}</p>
              )}
              <p className="text-[10px] text-chat-muted flex items-center gap-1">
                <span className="w-1.5 h-1.5 rounded-full bg-chat-online inline-block" />
                {conv.onlineCount} online
              </p>
              </div>
            </>
          ) : (
            <div className="flex items-center gap-3">
              <div className="w-9 h-9 rounded-2xl bg-chat-card flex items-center justify-center text-xs font-bold text-chat-muted">
                #
              </div>
              <div>
                <h3 className="text-sm font-semibold text-white">Select a conversation</h3>
              </div>
            </div>
          )}
        </div>

        {/* Right actions */}
        <div className="flex items-center gap-1">
          <button className="w-8 h-8 rounded-lg flex items-center justify-center text-chat-muted hover:text-white hover:bg-chat-card transition-all" title="Voice Call">
            <PhoneIcon />
          </button>
          <button className="w-8 h-8 rounded-lg flex items-center justify-center text-chat-muted hover:text-white hover:bg-chat-card transition-all" title="Video Call">
            <VideoIcon />
          </button>
          <button className="w-8 h-8 rounded-lg flex items-center justify-center text-chat-muted hover:text-white hover:bg-chat-card transition-all" title="Pinned Messages">
            <PinIcon />
          </button>
          <button className="w-8 h-8 rounded-lg flex items-center justify-center text-chat-muted hover:text-white hover:bg-chat-card transition-all" title="More">
            <DotsIcon />
          </button>
        </div>
      </div>

      {/* ── Featured Activity Banner ────────────────────────────────── */}
      {conv && (
        <div className="relative h-40 bg-gradient-to-br from-chat-card to-chat-inner border-b border-chat-border shrink-0 overflow-hidden">
          <Image
            src="https://images.unsplash.com/photo-1611162617474-5b21e879e113?w=800&h=200&fit=crop"
            alt=""
            width={800}
            height={200}
            className="w-full h-full object-cover opacity-40"
          />
          <div className="absolute inset-0 bg-gradient-to-t from-chat-panel/80 via-transparent to-transparent" />
          <div className="absolute bottom-3 left-5">
            <div className="flex items-center gap-2">
              <div
                className="w-10 h-10 rounded-2xl flex items-center justify-center text-sm font-bold text-white shadow-lg"
                style={{ backgroundColor: firstMember?.color || "#5865F2" }}
              >
                {conv.avatar}
              </div>
              <div>
                <p className="text-base font-bold text-white">{conv.name}</p>
                <p className="text-[11px] text-chat-muted">{conv.members.length} members</p>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* ── Messages ────────────────────────────────────────────────── */}
      <div ref={listRef} className="flex-1 overflow-y-auto px-4 py-3 space-y-1 scrollbar-thin">
        {groupedMessages.length === 0 ? (
          <div className="flex items-center justify-center h-full">
            <div className="text-center">
              <div className="w-14 h-14 rounded-full bg-chat-card flex items-center justify-center mx-auto mb-3">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#8A8D93" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
                </svg>
              </div>
              <p className="text-sm text-chat-muted">No messages yet</p>
              <p className="text-xs text-chat-muted/60 mt-1">Be the first to send a message!</p>
            </div>
          </div>
        ) : (
          groupedMessages.map((group, gi) => (
            <div key={gi}>
              {/* Date separator */}
              <div className="flex items-center gap-3 py-3">
                <div className="flex-1 h-px bg-chat-border" />
                <span className="text-[11px] text-chat-muted font-medium px-2">{group.date}</span>
                <div className="flex-1 h-px bg-chat-border" />
              </div>

              {/* Messages */}
              <div className="space-y-0.5">                  {group.messages.map((msg, mi) => (
                  <MessageBubble
                    key={msg.id || mi}
                    msg={msg}
                    isOwn={msg.senderId === currentUser.id}
                    onUserClick={setSelectedProfileUser}
                  />
                ))}
              </div>
            </div>
          ))
        )}


      </div>

      {/* ── Input ────────────────────────────────────────────────────── */}
      <div className="px-4 py-3 border-t border-chat-border shrink-0">
        <form onSubmit={handleSend} className="flex items-center gap-2 bg-chat-inner rounded-full px-4 py-2.5 border border-chat-border">
          <button
            type="button"
            className="w-8 h-8 rounded-full flex items-center justify-center text-chat-muted hover:text-white transition-colors shrink-0"
            title="Attach file"
          >
            <AttachIcon />
          </button>
          <button
            type="button"
            className="w-8 h-8 rounded-full flex items-center justify-center text-chat-muted hover:text-white transition-colors shrink-0"
            title="Emoji"
          >
            <EmojiIcon />
          </button>
          <input
            ref={inputRef}
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder={`Message #${conv?.name || "chat"}`}
            className="flex-1 bg-transparent text-white text-sm outline-none placeholder:text-chat-muted px-1"
          />
          <button
            type="submit"
            disabled={!input.trim()}
            className="w-8 h-8 rounded-full flex items-center justify-center bg-chat-accent text-black disabled:bg-chat-card disabled:text-chat-muted transition-all shrink-0"
          >
            <SendIcon />
          </button>
        </form>
      </div>
    </div>
  )
}
