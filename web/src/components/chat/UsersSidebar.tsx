"use client"

import { useState, useEffect } from "react"
import { useChatStore } from "./ChatStore"

// ─── SVG Icons ──────────────────────────────────────────────────────────

function SearchIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="11" cy="11" r="8" />
      <path d="m21 21-4.35-4.35" />
    </svg>
  )
}

function MessageIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
    </svg>
  )
}

function OnlineDot() {
  return <span className="w-2 h-2 rounded-full bg-chat-online inline-block shrink-0" />
}

function OfflineDot() {
  return <span className="w-2 h-2 rounded-full bg-slate-600 inline-block shrink-0" />
}

// ─── Component ──────────────────────────────────────────────────────────

export default function UsersSidebar() {
  const { registeredUsers, currentUser, startConversationWith, setWorkspaceView } = useChatStore()
  const [search, setSearch] = useState("")

  const filtered = registeredUsers.filter((u) =>
    u.name.toLowerCase().includes(search.toLowerCase()),
  )

  // Group: online first, then alphabetical
  const sorted = [...filtered].sort((a, b) => {
    if (a.status === "online" && b.status !== "online") return -1
    if (a.status !== "online" && b.status === "online") return 1
    return a.name.localeCompare(b.name)
  })

  return (
    <div className="flex flex-col w-80 bg-chat-panel border-r border-chat-border shrink-0 h-full">
      {/* ── Header ──────────────────────────────────────────────────── */}
      <div className="px-4 pt-4 pb-3 shrink-0">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <button
              onClick={() => setWorkspaceView("chat")}
              className="w-7 h-7 rounded-lg flex items-center justify-center text-chat-muted hover:text-white hover:bg-chat-card transition-all"
            >
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M19 12H5M12 19l-7-7 7-7" />
              </svg>
            </button>
            <h2 className="text-sm font-bold text-white tracking-wide">Users</h2>
            <span className="text-[10px] text-chat-muted font-medium bg-chat-card px-1.5 py-0.5 rounded-md">
              {registeredUsers.length}
            </span>
          </div>
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
            placeholder="Search users..."
            className="w-full bg-chat-inner text-white text-xs rounded-xl pl-9 pr-4 py-2.5 outline-none border border-chat-border focus:border-slate-600 placeholder:text-chat-muted transition-colors"
          />
        </div>
      </div>

      {/* ── User Count ──────────────────────────────────────────────── */}
      {!search && (
        <div className="px-4 pb-2 shrink-0">
          <span className="text-[10px] text-chat-muted font-medium flex items-center gap-1.5">
            <span className="w-1.5 h-1.5 rounded-full bg-chat-online" />
            Online — {registeredUsers.filter((u) => u.status === "online").length}
          </span>
        </div>
      )}

      {/* ── User List ────────────────────────────────────────────────── */}
      <div className="flex-1 overflow-y-auto px-2 pb-2 space-y-0.5 scrollbar-thin">
        {sorted.length === 0 ? (
          <div className="flex items-center justify-center h-24 text-chat-muted text-xs">
            {search ? "No users found" : "No users registered yet"}
          </div>
        ) : (
          sorted.map((user) => {
            const isSelf = user.id === currentUser.id

            return (
              <div
                key={user.id}
                className="flex items-center gap-3 px-3 py-2.5 rounded-xl transition-all duration-150 group hover:bg-chat-card/60"
              >
                {/* Avatar */}
                <div className="relative shrink-0">
                  <div
                    className="w-10 h-10 rounded-2xl flex items-center justify-center text-xs font-bold text-white"
                    style={{ backgroundColor: user.color }}
                  >
                    {user.name.split(" ").map((w) => w[0]).join("").toUpperCase().slice(0, 2)}
                  </div>
                  {isSelf ? (
                    <span className="absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 rounded-full border-2 border-chat-panel bg-chat-accent" />
                  ) : user.status === "online" ? (
                    <OnlineDot />
                  ) : (
                    <OfflineDot />
                  )}
                  {isSelf && (
                    <span className="absolute -top-0.5 -left-0.5 text-[8px]">👤</span>
                  )}
                </div>

                {/* User Info */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-sm text-slate-200 font-medium truncate">
                      {user.name}
                      {isSelf && <span className="text-[10px] text-chat-accent ml-1">(you)</span>}
                    </span>
                    <span className="text-[10px] text-chat-muted capitalize shrink-0">
                      {user.role}
                    </span>
                  </div>
                  <div className="flex items-center gap-1.5 mt-0.5">
                    <span className={`text-[10px] ${isSelf ? "text-chat-accent" : "text-chat-muted"}`}>
                      {isSelf ? "Online" : user.status}
                    </span>
                    {user.email && (
                      <>
                        <span className="text-[8px] text-chat-muted">·</span>
                        <span className="text-[9px] text-chat-muted/60 truncate">{user.email}</span>
                      </>
                    )}
                  </div>
                </div>

                {/* Message button */}
                {!isSelf && (
                  <button
                    onClick={() => startConversationWith(user.id, user.name, user.email)}
                    className="w-8 h-8 rounded-lg flex items-center justify-center text-chat-muted hover:text-white hover:bg-chat-accent/20 transition-all opacity-0 group-hover:opacity-100"
                    title={`Message ${user.name}`}
                  >
                    <MessageIcon />
                  </button>
                )}
              </div>
            )
          })
        )}
      </div>
    </div>
  )
}
