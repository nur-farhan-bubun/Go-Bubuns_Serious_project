"use client"

import dynamic from 'next/dynamic'
import { useEffect } from "react"
import { useSearchParams } from "next/navigation"
import ChatOverlay from "../components/chat/ChatOverlay"
import WorkspaceBar from "../components/chat/WorkspaceBar"
import { handleAuthCallback, getAuthState, loginSimulated, wasRecentlyLoggedOut } from "../lib/auth"
import { useChatStore } from "../components/chat/ChatStore"

const MapView = dynamic(() => import("../components/MapView"), { ssr: false })

function AuthInitializer() {
  const searchParams = useSearchParams()
  const setCurrentUserId = useChatStore((s) => s.setCurrentUserId)

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

    // Skip auto-login if the user just logged out
    if (wasRecentlyLoggedOut()) {
      return
    }

    // Auto-login with simulated user for quick dev demos
    loginSimulated("user-sim-001").then((simUser) => {
      setCurrentUserId(simUser.id, simUser.name)
    })
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
      <AuthInitializer />
    </div>
  )
}
