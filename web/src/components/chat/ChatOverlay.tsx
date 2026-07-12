"use client"

import { useState } from "react"
import { motion, AnimatePresence } from "framer-motion"
import { useChatStore } from "./ChatStore"
import ChatConversations from "./ChatConversations"
import ChatFeed from "./ChatFeed"
import ChatInfoSidebar from "./ChatInfoSidebar"
import UsersSidebar from "./UsersSidebar"
import CreateGroupDialog from "./CreateGroupDialog"

// ─── Component ──────────────────────────────────────────────────────────

export default function ChatOverlay() {
  const { isChatOpen, closeChat, activeWorkspaceView } = useChatStore()
  const [isCreateGroupOpen, setIsCreateGroupOpen] = useState(false)

  return (
    <AnimatePresence>
      {isChatOpen && (
        <>
          {/* ─── Backdrop ──────────────────────────────────────────────── */}
          <motion.div
            key="chat-backdrop"
            className="fixed inset-0 z-[90] bg-black/60 backdrop-blur-sm"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
            onClick={closeChat}
          />

          {/* ─── Chat Panel ────────────────────────────────────────────── */}
          <motion.div
            key="chat-panel"
            className="fixed inset-y-0 left-20 right-0 z-[95] flex pointer-events-none"
            initial={{ x: "-100%" }}
            animate={{ x: 0 }}
            exit={{ x: "-100%" }}
            transition={{ type: "spring", damping: 28, stiffness: 300, mass: 0.8 }}
          >
            <div className="flex flex-1 pointer-events-auto h-full overflow-hidden">
              {activeWorkspaceView === "users" ? (
                /* ── Users Directory View ────────────────────────────── */
                <>
                  <UsersSidebar onOpenCreateGroup={() => setIsCreateGroupOpen(true)} />
                  <div className="flex-1 flex items-center justify-center bg-chat-panel">
                    <div className="text-center">
                      <div className="w-16 h-16 rounded-full bg-chat-card flex items-center justify-center mx-auto mb-4">
                        <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#8A8D93" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                          <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
                          <circle cx="9" cy="7" r="4" />
                          <path d="M22 21v-2a4 4 0 0 0-3-3.87" />
                          <path d="M16 3.13a4 4 0 0 1 0 7.75" />
                        </svg>
                      </div>
                      <h3 className="text-base font-semibold text-white">User Directory</h3>
                      <p className="text-sm text-chat-muted mt-1 max-w-xs">
                        Browse registered users and start a conversation by clicking the message icon.
                      </p>
                    </div>
                  </div>
                </>
              ) : (
                /* ── Chat View ──────────────────────────────────────── */
                <>
                  {/* Conversations */}
                  <div className="hidden md:block">
                    <ChatConversations onOpenCreateGroup={() => setIsCreateGroupOpen(true)} />
                  </div>

                  {/* Main Feed */}
                  <ChatFeed />

                  {/* Info Sidebar */}
                  <div className="hidden xl:block">
                    <ChatInfoSidebar />
                  </div>
                </>
              )}
            </div>
          </motion.div>

          {/* ─── Create Group Dialog ───────────────────────────────────── */}
          <CreateGroupDialog
            isOpen={isCreateGroupOpen}
            onClose={() => setIsCreateGroupOpen(false)}
          />
        </>
      )}
    </AnimatePresence>
  )
}
