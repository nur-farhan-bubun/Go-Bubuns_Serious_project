"use client"

import { useState } from "react"
import { useChatStore } from "./ChatStore"

// ─── Search Icon ────────────────────────────────────────────────────────

function SearchIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="11" cy="11" r="8" />
      <path d="m21 21-4.35-4.35" />
    </svg>
  )
}

// ─── Component ──────────────────────────────────────────────────────────

export default function ChatConversations() {
  const { conversations, activeConversationId, setConversation, workspaces, activeWorkspaceId } = useChatStore()
  const [search, setSearch] = useState("")

  const activeWorkspace = workspaces.find((w) => w.id === activeWorkspaceId)
  const filtered = conversations.filter((c) =>
    c.name.toLowerCase().includes(search.toLowerCase()),
  )

  // Group: pinned/unread first
  const sorted = [...filtered].sort((a, b) => {
    if (a.unread > 0 && b.unread === 0) return -1
    if (a.unread === 0 && b.unread > 0) return 1
    if (a.isActive && !b.isActive) return -1
    if (!a.isActive && b.isActive) return 1
    return 0
  })

  return (
    <div className="flex flex-col w-80 bg-chat-panel border-r border-chat-border shrink-0 h-full">
      {/* ── Header ──────────────────────────────────────────────────── */}
      <div className="px-4 pt-4 pb-3 shrink-0">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <h2 className="text-sm font-bold text-white tracking-wide">
              {activeWorkspace?.label || "Chats"}
            </h2>
            <span className="text-[10px] text-chat-muted font-medium bg-chat-card px-1.5 py-0.5 rounded-md">
              {conversations.length}
            </span>
          </div>
          <button className="w-7 h-7 rounded-lg flex items-center justify-center text-chat-muted hover:text-white hover:bg-chat-card transition-all">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="12" cy="12" r="1" />
              <circle cx="19" cy="12" r="1" />
              <circle cx="5" cy="12" r="1" />
            </svg>
          </button>
        </div>
        {/* Search */}
        <div className="relative">
          <span className="absolute left-3 top-1/2 -translate-y-1/2 text-chat-muted">
            <SearchIcon />
          </span>
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search conversations..."
            className="w-full bg-chat-inner text-white text-xs rounded-xl pl-9 pr-4 py-2.5 outline-none border border-chat-border focus:border-slate-600 placeholder:text-chat-muted transition-colors"
          />
        </div>
      </div>

      {/* ── Online count ────────────────────────────────────────────── */}
      {!search && (
        <div className="px-4 pb-2 shrink-0">
          <span className="text-[10px] text-chat-muted font-medium flex items-center gap-1.5">
            <span className="w-1.5 h-1.5 rounded-full bg-chat-online" />
            Online — {conversations.reduce((acc, c) => acc + c.onlineCount, 0)}
          </span>
        </div>
      )}

      {/* ── Conversation List ────────────────────────────────────────── */}
      <div className="flex-1 overflow-y-auto px-2 pb-2 space-y-0.5 scrollbar-thin">
        {sorted.length === 0 ? (
          <div className="flex items-center justify-center h-24 text-chat-muted text-xs">
            No conversations found
          </div>
        ) : (
          sorted.map((conv) => {
            const isActive = conv.id === activeConversationId
            const firstMember = conv.members[1] // Skip self
            const statusColor = "bg-slate-600"

            return (
              <button
                key={conv.id}
                onClick={() => setConversation(conv.id)}
                className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl text-left transition-all duration-150 group
                  ${isActive
                    ? "bg-chat-card"
                    : "hover:bg-chat-card/60"
                  }`}
              >
                {/* Avatar */}
                <div className="relative shrink-0">
                  <div
                    className={`w-10 h-10 rounded-2xl flex items-center justify-center text-xs font-bold text-white ${isActive ? "ring-2 ring-chat-accent/30" : ""}`}
                    style={{
                      backgroundColor: conv.members[1]?.color || "#5865F2",
                    }}
                  >
                    {conv.avatar}
                  </div>
                  <span className={`absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 rounded-full border-2 border-chat-panel ${statusColor}`} />
                </div>

                {/* Content */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between gap-2">
                    <div className="min-w-0 flex-1">
                      <span className={`text-sm truncate block ${isActive ? "text-white font-semibold" : "text-slate-300 font-medium"}`}>
                        {conv.name}
                      </span>
                      {firstMember?.email && (
                        <span className="text-[9px] text-chat-muted/60 truncate block leading-tight">{firstMember.email}</span>
                      )}
                    </div>
                    <span className="text-[10px] text-chat-muted shrink-0">{conv.lastTime}</span>
                  </div>
                  <div className="flex items-center justify-between gap-2 mt-0.5">
                    <span className="text-xs text-chat-muted truncate">{conv.lastMessage}</span>
                    {conv.unread > 0 && (
                      <span className="shrink-0 bg-chat-accent text-black text-[10px] font-bold px-1.5 py-0.5 rounded-full min-w-[18px] text-center leading-tight">
                        {conv.unread}
                      </span>
                    )}
                  </div>
                </div>
              </button>
            )
          })
        )}
      </div>
    </div>
  )
}
