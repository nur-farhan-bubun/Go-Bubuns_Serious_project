"use client"

import { motion } from "framer-motion"
import { useChatStore } from "./ChatStore"
import { loginWithGoogle, loginSimulated, logout, getAuthState, getTokenStatus, getLocalRegisteredUsers, type TokenStatus, type AuthUser } from "../../lib/auth"
import { useState, useEffect, useRef } from "react"

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

function LogoutIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
      <polyline points="16 17 21 12 16 7" />
      <line x1="21" x2="9" y1="12" y2="12" />
    </svg>
  )
}

function LoginIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4" />
      <polyline points="10 17 15 12 10 7" />
      <line x1="15" x2="3" y1="12" y2="12" />
    </svg>
  )
}

function UsersIcon() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
      <circle cx="9" cy="7" r="4" />
      <path d="M22 21v-2a4 4 0 0 0-3-3.87" />
      <path d="M16 3.13a4 4 0 0 1 0 7.75" />
    </svg>
  )
}

// ─── Login Dropdown ───────────────────────────────────────────

function LoginDropdown({ onClose, dropdownRef }: { onClose: () => void; dropdownRef: React.RefObject<HTMLDivElement | null> }) {
  const [localUsers, setLocalUsers] = useState<AuthUser[]>([])
  const [emailInput, setEmailInput] = useState("")
  const setCurrentUserId = useChatStore((s) => s.setCurrentUserId)

  useEffect(() => {
    setLocalUsers(getLocalRegisteredUsers())
  }, [])

  async function handleEmailLogin(e: React.FormEvent) {
    e.preventDefault()
    if (!emailInput.trim()) return
    const userId = `user-${emailInput.replace(/[^a-zA-Z0-9]/g, "-").toLowerCase()}`
    const displayName = emailInput.split("@")[0]
    const user = await loginSimulated(userId, displayName, emailInput.trim())
    setCurrentUserId(user.id, user.name, user.email)
    onClose()
  }

  async function handleExistingUserLogin(u: AuthUser) {
    const user = await loginSimulated(u.id, u.name, u.email)
    setCurrentUserId(user.id, user.name, user.email)
    onClose()
  }

  const colors = ["#06D6A0", "#ED4245", "#57F287", "#FEE75C", "#EB459E", "#1ABC9C", "#9B59B6", "#3498DB", "#E67E22", "#00BCD4"]
  function colorFromId(id: string): string {
    let hash = 0
    for (let i = 0; i < id.length; i++) {
      hash = id.charCodeAt(i) + ((hash << 5) - hash)
    }
    return colors[Math.abs(hash) % colors.length]
  }

  return (
    <div ref={dropdownRef} className="absolute left-full ml-3 bottom-0 bg-black/95 border border-chat-border rounded-2xl p-2 shadow-2xl z-[200] min-w-[200px] backdrop-blur-xl">
      {/* Quick email login */}
      <form onSubmit={handleEmailLogin} className="px-2 pb-2">
        <p className="text-[10px] text-chat-muted pb-1.5 font-medium">Quick email login</p>
        <div className="flex gap-1.5">
          <input
            type="email"
            value={emailInput}
            onChange={(e) => setEmailInput(e.target.value)}
            placeholder="your@email.com"
            className="flex-1 bg-chat-card text-white text-xs rounded-xl px-3 py-2 outline-none border border-chat-border focus:border-slate-600 placeholder:text-chat-muted transition-colors"
          />
          <button
            type="submit"
            disabled={!emailInput.trim()}
            className="px-3 py-2 bg-chat-accent text-black text-xs font-semibold rounded-xl hover:opacity-90 transition-all disabled:opacity-50"
          >
            Go
          </button>
        </div>
      </form>

      {/* Google OAuth */}
      <button
        onClick={() => { loginWithGoogle(); onClose() }}
        className="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl hover:bg-chat-card transition-colors text-left"
      >
        <div className="w-7 h-7 rounded-full bg-white flex items-center justify-center text-sm shrink-0">
          G
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-xs font-medium text-white">Sign in with Google</p>
          <p className="text-[10px] text-chat-muted">OAuth</p>
        </div>
      </button>

      {/* Recent users */}
      {localUsers.length > 0 && (
        <>
          <div className="flex items-center gap-2 px-3 py-1.5">
            <div className="flex-1 h-px bg-chat-border" />
            <span className="text-[10px] text-chat-muted">Recent</span>
            <div className="flex-1 h-px bg-chat-border" />
          </div>
          {localUsers.map((u) => (
            <button
              key={u.id}
              onClick={() => handleExistingUserLogin(u)}
              className="w-full flex items-center gap-2.5 px-3 py-2 rounded-xl hover:bg-chat-card transition-colors text-left"
            >
              <div
                className="w-7 h-7 rounded-full flex items-center justify-center text-[10px] font-bold text-white shrink-0"
                style={{ backgroundColor: colorFromId(u.id) }}
              >
                {u.name.split(" ")[0][0]?.toUpperCase() || "U"}
              </div>
              <div className="min-w-0 flex-1">
                <span className="text-xs text-slate-300 truncate block">{u.name}</span>
                <span className="text-[9px] text-chat-muted truncate block">{u.email}</span>
              </div>
            </button>
          ))}
        </>
      )}
    </div>
  )
}

// ─── Component ──────────────────────────────────────────────────────────

export default function WorkspaceBar() {
  const { workspaces, activeWorkspaceId, setWorkspace, isChatOpen, openChat, toggleChat, currentUser, resetState, setWorkspaceView, activeWorkspaceView } = useChatStore()
  const [showLogin, setShowLogin] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)

  // Derive login state from chat store's currentUser
  const isLoggedIn = currentUser.id !== "u-me" || getAuthState().isAuthenticated

  // Auth token status
  const tokenStatus: TokenStatus = isLoggedIn ? getTokenStatus().status : "no-token"
  const statusColors: Record<TokenStatus, string> = {
    "valid": "bg-chat-online",
    "expiring-soon": "bg-yellow-400",
    "expired": "bg-red-500",
    "no-token": "bg-slate-600",
  }
  const statusLabels: Record<TokenStatus, string> = {
    "valid": "Session active",
    "expiring-soon": "Token expiring soon",
    "expired": "Session expired — re-login",
    "no-token": "Not signed in",
  }

  // Close login dropdown when clicking outside
  useEffect(() => {
    if (!showLogin) return
    function handleClick(e: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node) &&
          buttonRef.current && !buttonRef.current.contains(e.target as Node)) {
        setShowLogin(false)
      }
    }
    document.addEventListener("mousedown", handleClick)
    return () => document.removeEventListener("mousedown", handleClick)
  }, [showLogin])

  // Initials for the avatar
  const initials = currentUser.name.split(" ").map((w) => w[0]).join("").toUpperCase().slice(0, 2) || "U"

  return (
    <div className="flex flex-col items-center w-20 h-full bg-chat-panel border-r border-chat-border py-3 shrink-0 z-[100] relative">
      {/* ── Top: User Avatar / Login Trigger ──────────────────────────── */}
      <div className="relative">
        <button
          ref={buttonRef}
          onClick={() => setShowLogin(!showLogin)}
          className={`w-12 h-12 rounded-2xl flex items-center justify-center text-sm font-bold shadow-lg transition-all duration-200 hover:rounded-xl
            ${isLoggedIn
              ? "bg-chat-accent text-black"
              : "bg-chat-card text-chat-muted border border-chat-border hover:text-white"
            }`}
          title={isLoggedIn ? `Logged in as ${currentUser.name}` : "Sign in"}
        >
          {isLoggedIn ? initials : <LoginIcon />}
        </button>
        {isLoggedIn && (
          <span
            className={`absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full border-2 border-chat-panel ${statusColors[tokenStatus]}`}
            title={statusLabels[tokenStatus]}
          />
        )}

        {/* Login dropdown */}
        {showLogin && <LoginDropdown dropdownRef={dropdownRef} onClose={() => setShowLogin(false)} />}
      </div>

      {/* ── Divider ─────────────────────────────────────────────────── */}
      <div className="w-8 h-px bg-chat-border my-4" />

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
        {/* Users Directory */}
        <button
          onClick={() => {
            if (!isChatOpen) openChat()
            setWorkspaceView("users")
          }}
          className={`w-12 h-12 rounded-2xl flex items-center justify-center text-lg transition-all duration-200
            ${isChatOpen && activeWorkspaceView === "users"
              ? "bg-chat-accent text-black shadow-lg shadow-chat-accent/20"
              : "bg-chat-inner text-chat-muted hover:bg-chat-card hover:text-white hover:rounded-xl"
            }`}
          title="Users"
        >
          <UsersIcon />
        </button>

        {/* Chat Toggle */}
        <button
          onClick={() => {
            if (!isChatOpen) openChat()
            else setWorkspaceView("chat")
          }}
          className={`w-12 h-12 rounded-2xl flex items-center justify-center text-lg transition-all duration-200 relative
            ${isChatOpen && activeWorkspaceView === "chat"
              ? "bg-chat-accent text-black shadow-lg shadow-chat-accent/20"
              : "bg-chat-inner text-chat-muted hover:bg-chat-card hover:text-white hover:rounded-xl"
            }`}
          title="Chat"
        >
          <ChatIcon />
          {/* Notification dot */}
          {isLoggedIn && (
            <span className="absolute -top-0.5 -right-0.5 w-3 h-3 rounded-full bg-red-500 border-2 border-chat-panel" />
          )}
        </button>

        {/* Settings / Logout */}
        {isLoggedIn ? (
          <button
            onClick={() => {
              resetState()
              logout()
            }}
            className="w-9 h-9 rounded-xl flex items-center justify-center text-chat-muted hover:bg-chat-card hover:text-red-400 transition-all"
            title="Logout"
          >
            <LogoutIcon />
          </button>
        ) : (
          <button
            className="w-9 h-9 rounded-xl flex items-center justify-center text-chat-muted hover:bg-chat-card hover:text-white transition-all"
            title="Settings"
          >
            <SettingsIcon />
          </button>
        )}

        {/* Add */}
        <button            className="w-9 h-9 rounded-xl flex items-center justify-center bg-chat-accent text-black font-bold hover:opacity-90 transition-all"
          title="Add"
        >
          <AddIcon />
        </button>
      </div>
    </div>
  )
}
