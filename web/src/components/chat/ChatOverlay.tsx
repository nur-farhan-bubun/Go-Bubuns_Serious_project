"use client"

import { motion, AnimatePresence } from "framer-motion"
import { useChatStore } from "./ChatStore"
import ChatConversations from "./ChatConversations"
import ChatFeed from "./ChatFeed"
import ChatInfoSidebar from "./ChatInfoSidebar"

// ─── Component ──────────────────────────────────────────────────────────

export default function ChatOverlay() {
  const { isChatOpen, closeChat } = useChatStore()

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
            className="fixed inset-0 z-[95] flex pointer-events-none"
            initial={{ x: "-100%" }}
            animate={{ x: 0 }}
            exit={{ x: "-100%" }}
            transition={{ type: "spring", damping: 28, stiffness: 300, mass: 0.8 }}
          >
            {/* Left spacer for workspace bar (w-20) */}
            <div className="w-20 shrink-0 pointer-events-none" />

            {/* Remaining 3 columns */}
            <div className="flex flex-1 pointer-events-auto h-full overflow-hidden">
              {/* Conversations */}
              <div className="hidden md:block">
                <ChatConversations />
              </div>

              {/* Main Feed */}
              <ChatFeed />

              {/* Info Sidebar */}
              <div className="hidden xl:block">
                <ChatInfoSidebar />
              </div>
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  )
}
