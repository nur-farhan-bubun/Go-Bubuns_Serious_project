"use client"

import { useState, useCallback, useRef, useEffect } from "react"
import { motion, AnimatePresence } from "framer-motion"
import { useChatStore } from "./ChatStore"
import { searchUsers, type SearchUserResult } from "../../lib/chat"

// ─── SVG Icons ──────────────────────────────────────────────────────────

function CloseIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M18 6 6 18M6 6l12 12" />
    </svg>
  )
}

function SearchIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="11" cy="11" r="8" />
      <path d="m21 21-4.35-4.35" />
    </svg>
  )
}

function PlusIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 5v14M5 12h14" />
    </svg>
  )
}

function SpinnerIcon() {
  return (
    <svg className="animate-spin" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 12a9 9 0 1 1-6.219-8.56" />
    </svg>
  )
}

// ─── Helpers ────────────────────────────────────────────────────────────

function userColorFromId(id: string): string {
  const colors = ["#06D6A0", "#ED4245", "#57F287", "#FEE75C", "#EB459E", "#1ABC9C", "#9B59B6", "#3498DB", "#E67E22", "#00BCD4"]
  let hash = 0
  for (let i = 0; i < id.length; i++) {
    hash = id.charCodeAt(i) + ((hash << 5) - hash)
  }
  return colors[Math.abs(hash) % colors.length]
}

// ─── Selected User Chip ─────────────────────────────────────────────────

function UserChip({ user, onRemove }: { user: SearchUserResult; onRemove: (id: string) => void }) {
  const initials = user.display_name
    .split(" ")
    .map((w) => w[0])
    .join("")
    .toUpperCase()
    .slice(0, 2)
  const color = userColorFromId(user.user_id)

  return (
    <motion.span
      initial={{ scale: 0.8, opacity: 0 }}
      animate={{ scale: 1, opacity: 1 }}
      exit={{ scale: 0.8, opacity: 0 }}
      className="inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-full text-xs font-medium text-white"
      style={{ backgroundColor: color + "30", border: `1px solid ${color}60` }}
    >
      <span
        className="w-4 h-4 rounded-full flex items-center justify-center text-[8px] font-bold text-white shrink-0"
        style={{ backgroundColor: color }}
      >
        {initials}
      </span>
      <span className="truncate max-w-[120px]">{user.display_name}</span>
      <button
        onClick={() => onRemove(user.user_id)}
        className="ml-0.5 w-3.5 h-3.5 rounded-full flex items-center justify-center hover:bg-white/20 transition-colors"
      >
        <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
          <path d="M18 6 6 18M6 6l12 12" />
        </svg>
      </button>
    </motion.span>
  )
}

// ─── Component ──────────────────────────────────────────────────────────

interface CreateGroupDialogProps {
  isOpen: boolean
  onClose: () => void
}

export default function CreateGroupDialog({ isOpen, onClose }: CreateGroupDialogProps) {
  const { currentUser, createGroup } = useChatStore()

  const [roomName, setRoomName] = useState("")
  const [searchQuery, setSearchQuery] = useState("")
  const [searchResults, setSearchResults] = useState<SearchUserResult[]>([])
  const [selectedUsers, setSelectedUsers] = useState<SearchUserResult[]>([])
  const [isSearching, setIsSearching] = useState(false)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  const searchInputRef = useRef<HTMLInputElement>(null)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Focus search input when dialog opens
  useEffect(() => {
    if (isOpen) {
      setTimeout(() => searchInputRef.current?.focus(), 150)
    }
  }, [isOpen])

  // Reset state when dialog opens
  useEffect(() => {
    if (isOpen) {
      setRoomName("")
      setSearchQuery("")
      setSearchResults([])
      setSelectedUsers([])
      setError(null)
      setSuccess(null)
    }
  }, [isOpen])

  // Debounced search
  const handleSearchChange = useCallback((value: string) => {
    setSearchQuery(value)

    if (debounceRef.current) {
      clearTimeout(debounceRef.current)
    }

    if (!value.trim()) {
      setSearchResults([])
      setIsSearching(false)
      return
    }

    setIsSearching(true)
    debounceRef.current = setTimeout(async () => {
      try {
        const results = await searchUsers(value, 15)
        // Filter out already-selected users and the current user
        const selectedIds = new Set(selectedUsers.map((u) => u.user_id))
        selectedIds.add(currentUser.id)
        setSearchResults(results.filter((u) => !selectedIds.has(u.user_id)))
      } catch {
        setSearchResults([])
      } finally {
        setIsSearching(false)
      }
    }, 300)
  }, [selectedUsers, currentUser.id])

  // Re-filter when selectedUsers changes (so selected users disappear from results)
  useEffect(() => {
    if (searchQuery.trim() && searchResults.length > 0) {
      const selectedIds = new Set(selectedUsers.map((u) => u.user_id))
      selectedIds.add(currentUser.id)
      setSearchResults((prev) => prev.filter((u) => !selectedIds.has(u.user_id)))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedUsers.length])

  const addUser = (user: SearchUserResult) => {
    setSelectedUsers((prev) => [...prev, user])
    setSearchQuery("")
    setSearchResults([])
    searchInputRef.current?.focus()
  }

  const removeUser = (userId: string) => {
    setSelectedUsers((prev) => prev.filter((u) => u.user_id !== userId))
  }

  const handleSubmit = async () => {
    setError(null)
    setSuccess(null)

    // Validation
    if (!roomName.trim()) {
      setError("Please enter a group name")
      return
    }
    if (selectedUsers.length === 0) {
      setError("Please select at least one member")
      return
    }

    setIsSubmitting(true)
    try {
      const memberIds = selectedUsers.map((u) => u.user_id)
      const convId = await createGroup(roomName.trim(), memberIds)

      if (!convId) {
        setError("Failed to create group. Please try again.")
        setIsSubmitting(false)
        return
      }

      setSuccess("Group created successfully!")

      // Close the dialog after a brief delay
      setTimeout(() => {
        onClose()
      }, 600)
    } catch {
      setError("Something went wrong. Please try again.")
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <AnimatePresence>
      {isOpen && (
        <>
          {/* Backdrop */}
          <motion.div
            className="fixed inset-0 z-[100] bg-black/60 backdrop-blur-sm"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={onClose}
          />

          {/* Dialog */}
          <motion.div
            className="fixed inset-0 z-[110] flex items-center justify-center pointer-events-none"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
          >
            <motion.div
              className="pointer-events-auto w-full max-w-lg mx-4 bg-chat-panel border border-chat-border rounded-2xl shadow-2xl overflow-hidden"
              initial={{ scale: 0.92, opacity: 0, y: 20 }}
              animate={{ scale: 1, opacity: 1, y: 0 }}
              exit={{ scale: 0.92, opacity: 0, y: 20 }}
              transition={{ type: "spring", damping: 26, stiffness: 320, mass: 0.7 }}
              onClick={(e) => e.stopPropagation()}
            >
              {/* ── Header ────────────────────────────────────────────── */}
              <div className="flex items-center justify-between px-5 py-4 border-b border-chat-border">
                <div>
                  <h2 className="text-base font-bold text-white">Create Group</h2>
                  <p className="text-xs text-chat-muted mt-0.5">Start a new group conversation</p>
                </div>
                <button
                  onClick={onClose}
                  className="w-8 h-8 rounded-lg flex items-center justify-center text-chat-muted hover:text-white hover:bg-chat-card transition-all"
                >
                  <CloseIcon />
                </button>
              </div>

              {/* ── Body ──────────────────────────────────────────────── */}
              <div className="px-5 py-4 space-y-4">
                {/* Room Name */}
                <div>
                  <label className="text-xs font-medium text-chat-muted mb-1.5 block">
                    Group Name
                  </label>
                  <input
                    type="text"
                    value={roomName}
                    onChange={(e) => setRoomName(e.target.value)}
                    placeholder="e.g. Trip Planning Squad"
                    className="w-full bg-chat-inner text-white text-sm rounded-xl px-4 py-2.5 outline-none border border-chat-border focus:border-chat-accent/50 placeholder:text-chat-muted transition-colors"
                    maxLength={100}
                  />
                </div>

                {/* User Search */}
                <div>
                  <label className="text-xs font-medium text-chat-muted mb-1.5 block">
                    Add Members
                  </label>

                  {/* Search input */}
                  <div className="relative mb-2">
                    <span className="absolute left-3 top-1/2 -translate-y-1/2 text-chat-muted">
                      {isSearching ? <SpinnerIcon /> : <SearchIcon />}
                    </span>
                    <input
                      ref={searchInputRef}
                      type="text"
                      value={searchQuery}
                      onChange={(e) => handleSearchChange(e.target.value)}
                      placeholder="Search users by name or email..."
                      className="w-full bg-chat-inner text-white text-sm rounded-xl pl-9 pr-4 py-2.5 outline-none border border-chat-border focus:border-chat-accent/50 placeholder:text-chat-muted transition-colors"
                    />
                  </div>

                  {/* Selected users as chips */}
                  {selectedUsers.length > 0 && (
                    <div className="flex flex-wrap gap-1.5 mb-2">
                      <AnimatePresence>
                        {selectedUsers.map((user) => (
                          <UserChip key={user.user_id} user={user} onRemove={removeUser} />
                        ))}
                      </AnimatePresence>
                    </div>
                  )}

                  {/* Search results dropdown */}
                  <AnimatePresence>
                    {searchQuery.trim() && searchResults.length > 0 && (
                      <motion.div
                        initial={{ opacity: 0, y: -4 }}
                        animate={{ opacity: 1, y: 0 }}
                        exit={{ opacity: 0, y: -4 }}
                        className="bg-chat-card border border-chat-border rounded-xl overflow-hidden max-h-48 overflow-y-auto scrollbar-thin"
                      >
                        {searchResults.map((user) => {
                          const initials = user.display_name
                            .split(" ")
                            .map((w) => w[0])
                            .join("")
                            .toUpperCase()
                            .slice(0, 2)
                          const color = userColorFromId(user.user_id)

                          return (
                            <button
                              key={user.user_id}
                              onClick={() => addUser(user)}
                              className="w-full flex items-center gap-3 px-3 py-2.5 hover:bg-chat-inner transition-colors text-left"
                            >
                              <div
                                className="w-8 h-8 rounded-full flex items-center justify-center text-[9px] font-bold text-white shrink-0"
                                style={{ backgroundColor: color }}
                              >
                                {initials}
                              </div>
                              <div className="flex-1 min-w-0">
                                <p className="text-sm text-slate-200 font-medium truncate">
                                  {user.display_name}
                                </p>
                              </div>
                              <span className="w-6 h-6 rounded-full bg-chat-accent/10 flex items-center justify-center text-chat-accent shrink-0">
                                <PlusIcon />
                              </span>
                            </button>
                          )
                        })}
                      </motion.div>
                    )}
                  </AnimatePresence>

                  {searchQuery.trim() && !isSearching && searchResults.length === 0 && (
                    <p className="text-xs text-chat-muted/60 text-center py-2">
                      No users found matching "{searchQuery}"
                    </p>
                  )}
                </div>

                {/* Error / Success */}
                {error && (
                  <motion.p
                    initial={{ opacity: 0, y: -4 }}
                    animate={{ opacity: 1, y: 0 }}
                    className="text-xs text-red-400 bg-red-400/10 rounded-lg px-3 py-2 border border-red-400/20"
                  >
                    {error}
                  </motion.p>
                )}
                {success && (
                  <motion.p
                    initial={{ opacity: 0, y: -4 }}
                    animate={{ opacity: 1, y: 0 }}
                    className="text-xs text-green-400 bg-green-400/10 rounded-lg px-3 py-2 border border-green-400/20"
                  >
                    {success}
                  </motion.p>
                )}
              </div>

              {/* ── Footer ────────────────────────────────────────────── */}
              <div className="flex items-center justify-end gap-3 px-5 py-4 border-t border-chat-border bg-chat-card/30">
                <button
                  onClick={onClose}
                  className="px-4 py-2 rounded-xl text-sm text-chat-muted hover:text-white hover:bg-chat-card transition-all"
                >
                  Cancel
                </button>
                <button
                  onClick={handleSubmit}
                  disabled={isSubmitting || !roomName.trim() || selectedUsers.length === 0}
                  className="px-5 py-2 rounded-xl text-sm font-semibold text-black bg-chat-accent hover:opacity-90 disabled:opacity-40 disabled:cursor-not-allowed transition-all flex items-center gap-2"
                >
                  {isSubmitting && <SpinnerIcon />}
                  {isSubmitting ? "Creating..." : "Create Group"}
                </button>
              </div>
            </motion.div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  )
}
