"use client"

import dynamic from 'next/dynamic'
import { Suspense, useEffect } from "react"
import { useSearchParams } from "next/navigation"
import ChatOverlay from "../components/chat/ChatOverlay"
import WorkspaceBar from "../components/chat/WorkspaceBar"
import { handleAuthCallback, getAuthState, loginSimulated, wasRecentlyLoggedOut } from "../lib/auth"
import { useChatStore } from "../components/chat/ChatStore"
import { useChatHandshake } from "../lib/useChatHandshake"

const MapView = dynamic(() => import("../components/MapView"), { ssr: false })

function AuthInitializer() {
  const searchParams = useSearchParams()
  const setCurrentUserId = useChatStore((s) => s.setCurrentUserId)
  const currentUserId = useChatStore((s) => s.currentUser.id)

  // Global WebSocket handshake — establishes a persistent connection for
  // presence updates and async room_ready notifications.
  useChatHandshake(currentUserId)

  useEffect(() => {
    // Check if returning from Google OAuth callback
    const code = searchParams.get("code")
    const state = searchParams.get("state")

    if (code && state) {
      // Handle OAuth callback
      handleAuthCallback(code, state).then((user) => {
        if (user) {
          setCurrentUserId(user.id, user.name)
          // Clean URL params
          window.history.replaceState({}, document.title, window.location.pathname)
        }
      })
      return
    }

    // Check if already authenticated from a previous session
    const { isAuthenticated, user: authUser } = getAuthState()
    if (isAuthenticated && authUser) {
      setCurrentUserId(authUser.id, authUser.name)
      return
    }

    // Skip auto-login if the user just logged out, is already authenticated,
    // or has local registered users (let them choose from the login page)
    if (wasRecentlyLoggedOut()) {
      return
    }

    // No session found & no logged-out flag — redirect to login page
    // so users can sign in with their email.
    window.location.href = "/login"
  }, [searchParams, setCurrentUserId])

  return null
}

export default function Home() {
  return (
    <div className="h-screen w-screen overflow-hidden bg-chat-bg flex">
      {/* Workspace Bar — always visible */}
      <WorkspaceBar />

      {/* Main Content Area */}
      <div className="flex-1 relative overflow-hidden">
        <MapView />
      </div>

      {/* Chat Overlay — slides in from the left */}
      <ChatOverlay />

      {/* Handles OAuth callback and session restoration */}
      <Suspense fallback={null}>
        <AuthInitializer />
      </Suspense>
    </div>
  )
}
