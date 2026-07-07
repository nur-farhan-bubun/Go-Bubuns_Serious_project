"use client"

import { motion } from "framer-motion"
import { useChatStore } from "./ChatStore"

// ─── SVG Icons ──────────────────────────────────────────────────────────

function SettingsIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="3" />
      <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42" />
    </svg>
  )
}

function AddIcon() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 5v14M5 12h14" />
    </svg>
  )
}

function ChatIcon() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
      <path d="M8 10h.01M12 10h.01M16 10h.01" />
    </svg>
  )
}

// ─── Component ──────────────────────────────────────────────────────────

export default function WorkspaceBar() {
  const { workspaces, activeWorkspaceId, setWorkspace, isChatOpen, toggleChat } = useChatStore()

  return (
    <div className="flex flex-col items-center w-20 h-full bg-chat-panel border-r border-chat-border py-3 shrink-0 z-[100]">
      {/* ── Top: Brand Icon ──────────────────────────────────────────── */}
      <div className="w-12 h-12 rounded-2xl bg-chat-accent text-black flex items-center justify-center font-bold text-lg mb-4 shadow-lg">
        R
      </div>

      {/* ── Divider ─────────────────────────────────────────────────── */}
      <div className="w-8 h-px bg-chat-border mb-4" />

      {/* ── Middle: Workspace Stack ─────────────────────────────────── */}
      <div className="flex flex-col items-center gap-3 flex-1">
        {workspaces.map((ws) => {
          const isActive = ws.id === activeWorkspaceId
          return (
            <button
              key={ws.id}
              onClick={() => setWorkspace(ws.id)}
              className="relative group"
              title={ws.label}
            >
              {/* Active indicator bar */}
              {isActive && (
                <motion.div
                  layoutId="ws-indicator"
                  className="absolute -left-3.5 top-1/2 -translate-y-1/2 w-1 h-8 rounded-r-full"
                  style={{ backgroundColor: ws.color }}
                  transition={{ type: "spring", stiffness: 300, damping: 25 }}
                />
              )}
              <div
                className={`w-12 h-12 rounded-2xl flex items-center justify-center text-lg transition-all duration-200
                  ${isActive
                    ? "bg-chat-card text-white shadow-lg shadow-black/40"
                    : "bg-chat-inner text-chat-muted hover:bg-chat-card hover:text-white hover:rounded-xl"
                  }`}
              >
                {ws.icon}
              </div>
              {/* Tooltip */}
              <div className="absolute left-full ml-3 top-1/2 -translate-y-1/2 px-2.5 py-1 bg-black/90 text-white text-xs rounded-lg whitespace-nowrap opacity-0 group-hover:opacity-100 pointer-events-none transition-opacity z-50 border border-chat-border shadow-xl">
                {ws.label}
              </div>
            </button>
          )
        })}
      </div>

      {/* ── Bottom: Actions ─────────────────────────────────────────── */}
      <div className="flex flex-col items-center gap-3 mt-auto">
        {/* Chat Toggle */}
        <button
          onClick={toggleChat}
          className={`w-12 h-12 rounded-2xl flex items-center justify-center text-lg transition-all duration-200 relative
            ${isChatOpen
              ? "bg-chat-accent text-black shadow-lg shadow-chat-accent/20"
              : "bg-chat-inner text-chat-muted hover:bg-chat-card hover:text-white hover:rounded-xl"
            }`}
          title="Chat"
        >
          <ChatIcon />
          {/* Notification dot */}
          <span className="absolute -top-0.5 -right-0.5 w-3 h-3 rounded-full bg-red-500 border-2 border-chat-panel" />
        </button>

        {/* Settings */}
        <button
          className="w-9 h-9 rounded-xl flex items-center justify-center text-chat-muted hover:bg-chat-card hover:text-white transition-all"
          title="Settings"
        >
          <SettingsIcon />
        </button>

        {/* Add */}
        <button
          className="w-9 h-9 rounded-xl flex items-center justify-center bg-chat-accent text-black font-bold hover:opacity-90 transition-all"
          title="Add"
        >
          <AddIcon />
        </button>
      </div>
    </div>
  )
}
