"use client"

import Image from "next/image"
import { useState, useRef, useEffect } from "react"
import {
  motion,
  AnimatePresence,
} from "framer-motion"
import { useMapStore, getUserColor, getInitials } from "../lib/useMapStore"
import { POST_CATEGORIES, createPost } from "../lib/location"
import type { MapPost } from "../types/location"
import { cn } from "../lib/utils"

// ─── Animation ──────────────────────────────────────────────────────────

const OVERLAY_TRANSITION = {
  duration: 0.25,
  ease: [0.4, 0, 0.2, 1] as [number, number, number, number],
}

const SPRING_UP = {
  type: "spring" as const,
  damping: 22,
  stiffness: 280,
  mass: 0.9,
}

// ─── Image URL input helper ─────────────────────────────────────────────

function isValidUrl(str: string): boolean {
  try {
    new URL(str)
    return true
  } catch {
    return false
  }
}

// ─── Component ──────────────────────────────────────────────────────────

interface CreatePostDrawerProps {
  onPostCreated?: (post: MapPost) => void
}

export default function CreatePostDrawer({ onPostCreated }: CreatePostDrawerProps) {
  const {
    isCreatePostOpen,
    createPostLat,
    createPostLng,
    closeCreatePost,
  } = useMapStore()

  const [title, setTitle] = useState("")
  const [content, setContent] = useState("")
  const [category, setCategory] = useState<"RESTAURANT" | "SOCIAL_LIFE" | "EVENT">("RESTAURANT")
  const [imageUrlInput, setImageUrlInput] = useState("")
  const [imageUrls, setImageUrls] = useState<string[]>([])
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const titleRef = useRef<HTMLInputElement>(null)

  // Focus title input when opened
  useEffect(() => {
    if (isCreatePostOpen) {
      setTimeout(() => titleRef.current?.focus(), 300)
    }
  }, [isCreatePostOpen])

  // Reset form when closed
  useEffect(() => {
    if (!isCreatePostOpen) {
      setTitle("")
      setContent("")
      setCategory("RESTAURANT")
      setImageUrlInput("")
      setImageUrls([])
      setSubmitting(false)
      setError(null)
    }
  }, [isCreatePostOpen])

  function addImageUrl() {
    const trimmed = imageUrlInput.trim()
    if (!trimmed) return
    if (!isValidUrl(trimmed)) {
      setError("Please enter a valid URL")
      return
    }
    setImageUrls((prev) => [...prev, trimmed])
    setImageUrlInput("")
    setError(null)
  }

  function removeImageUrl(index: number) {
    setImageUrls((prev) => prev.filter((_, i) => i !== index))
  }

  function handleImageUrlKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter") {
      e.preventDefault()
      addImageUrl()
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!title.trim()) {
      setError("Title is required")
      return
    }

    setSubmitting(true)
    setError(null)

    const result = await createPost({
      title: title.trim(),
      content: content.trim() || undefined,
      category,
      image_urls: imageUrls.length > 0 ? imageUrls : undefined,
      latitude: createPostLat,
      longitude: createPostLng,
    })

    if (result) {
      onPostCreated?.(result)
      closeCreatePost()
    } else {
      setError("Failed to create post. Please try again.")
    }

    setSubmitting(false)
  }

  const userColor = getUserColor("user-sim-001")

  return (
    <AnimatePresence>
      {isCreatePostOpen && (
        <>
          {/* ─── Overlay ────────────────────────────────────────────── */}
          <motion.div
            key="create-overlay"
            className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={OVERLAY_TRANSITION}
            onClick={closeCreatePost}
          />

          {/* ─── Sheet ──────────────────────────────────────────────── */}
          <motion.div
            key="create-sheet"
            className="fixed bottom-0 left-0 right-0 z-50 mx-auto flex flex-col rounded-t-[20px] bg-slate-900/95 backdrop-blur-2xl shadow-2xl border-t border-slate-700/60 outline-none overflow-hidden"
            style={{
              maxHeight: "92dvh",
              height: "85dvh",
            }}
            initial={{ y: "100%" }}
            animate={{ y: 0 }}
            exit={{ y: "100%" }}
            transition={SPRING_UP}
          >
            {/* ── Drag Handle ───────────────────────────────────────── */}
            <div className="flex items-center justify-center pt-3 pb-2 cursor-grab active:cursor-grabbing z-10 shrink-0">
              <div className="h-1.5 w-12 rounded-full bg-slate-600/60" />
            </div>

            {/* ── Header ────────────────────────────────────────────── */}
            <div className="flex items-center justify-between px-5 pb-3 border-b border-slate-800/80 shrink-0">
              <div className="flex items-center gap-3">
                <div className="w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold text-white shrink-0"
                  style={{ backgroundColor: userColor }}
                >
                  {getInitials("You")}
                </div>
                <div>
                  <h2 className="text-base font-bold text-white">Create Post</h2>
                  <p className="text-[11px] text-slate-400">
                    {createPostLat.toFixed(4)}, {createPostLng.toFixed(4)}
                  </p>
                </div>
              </div>

              <button
                onClick={closeCreatePost}
                className="w-8 h-8 flex items-center justify-center rounded-full hover:bg-slate-800 transition-colors text-slate-400 hover:text-white"
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M18 6 6 18M6 6l12 12" />
                </svg>
              </button>
            </div>

            {/* ── Form ────────────────────────────────────────────────── */}
            <form onSubmit={handleSubmit} className="flex-1 overflow-y-auto overscroll-contain px-5 py-4 space-y-4">
              {/* Title */}
              <div>
                <label className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1.5 block">
                  Title <span className="text-red-400">*</span>
                </label>
                <input
                  ref={titleRef}
                  type="text"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder="What's the name of this place?"
                  maxLength={255}
                  className="w-full bg-slate-800/80 text-white text-sm rounded-xl px-4 py-3 outline-none border border-slate-700/50 focus:border-slate-500/60 focus:bg-slate-800 placeholder:text-slate-500 transition-colors"
                />
                <p className="text-[10px] text-slate-600 mt-1 text-right">{title.length}/255</p>
              </div>

              {/* Content */}
              <div>
                <label className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1.5 block">
                  Description
                </label>
                <textarea
                  value={content}
                  onChange={(e) => setContent(e.target.value)}
                  placeholder="What's special about this place? Any recommendations?"
                  rows={3}
                  className="w-full bg-slate-800/80 text-white text-sm rounded-xl px-4 py-3 outline-none border border-slate-700/50 focus:border-slate-500/60 focus:bg-slate-800 placeholder:text-slate-500 transition-colors resize-none"
                />
              </div>

              {/* Category */}
              <div>
                <label className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1.5 block">
                  Category <span className="text-red-400">*</span>
                </label>
                <div className="grid grid-cols-3 gap-2">
                  {(Object.entries(POST_CATEGORIES) as [string, typeof POST_CATEGORIES[keyof typeof POST_CATEGORIES]][]).map(([key, val]) => (
                    <button
                      key={key}
                      type="button"
                      onClick={() => setCategory(key as "RESTAURANT" | "SOCIAL_LIFE" | "EVENT")}
                      className={cn(
                        "flex flex-col items-center gap-1.5 px-3 py-3 rounded-xl border transition-all duration-200",
                        category === key
                          ? "border-slate-400/60 bg-slate-800/80"
                          : "border-slate-700/50 bg-slate-800/40 hover:bg-slate-800/60",
                      )}
                    >
                      <span className="text-lg">{val.icon}</span>
                      <span className="text-[11px] font-medium text-slate-300">{val.label}</span>
                      {category === key && (
                        <motion.div
                          layoutId="category-dot"
                          className="w-1.5 h-1.5 rounded-full mt-0.5"
                          style={{ backgroundColor: val.color }}
                        />
                      )}
                    </button>
                  ))}
                </div>
              </div>

              {/* Image URLs */}
              <div>
                <label className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1.5 block">
                  Images
                </label>
                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    value={imageUrlInput}
                    onChange={(e) => setImageUrlInput(e.target.value)}
                    onKeyDown={handleImageUrlKeyDown}
                    placeholder="Paste an image URL..."
                    className="flex-1 bg-slate-800/80 text-white text-sm rounded-xl px-4 py-2.5 outline-none border border-slate-700/50 focus:border-slate-500/60 focus:bg-slate-800 placeholder:text-slate-500 transition-colors"
                  />
                  <button
                    type="button"
                    onClick={addImageUrl}
                    disabled={!imageUrlInput.trim()}
                    className="w-9 h-9 flex items-center justify-center rounded-xl bg-blue-500 text-white shadow-lg shadow-blue-500/25 hover:bg-blue-400 disabled:bg-slate-800 disabled:text-slate-600 disabled:shadow-none transition-all shrink-0"
                  >
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <path d="M12 5v14M5 12h14" />
                    </svg>
                  </button>
                </div>

                {/* Image previews */}
                {imageUrls.length > 0 && (
                  <div className="mt-2 flex flex-wrap gap-2">
                    {imageUrls.map((url, i) => (
                      <div
                        key={i}
                        className="relative group w-16 h-16 rounded-lg overflow-hidden border border-slate-700/50 bg-slate-800"
                      >
                        <Image
                          src={url}
                          alt=""
                          width={64}
                          height={64}
                          className="w-full h-full object-cover"
                          onError={(e) => {
                            const target = e.currentTarget
                            target.style.display = "none"
                          }}
                        />
                        <button
                          type="button"
                          onClick={() => removeImageUrl(i)}
                          className="absolute top-0.5 right-0.5 w-5 h-5 rounded-full bg-black/60 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity"
                        >
                          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="white" strokeWidth="2">
                            <path d="M18 6 6 18M6 6l12 12" />
                          </svg>
                        </button>
                      </div>
                    ))}
                  </div>
                )}
                <p className="text-[10px] text-slate-600 mt-1">
                  {imageUrls.length} image{imageUrls.length !== 1 ? "s" : ""} added
                </p>
              </div>

              {/* Location */}
              <div>
                <label className="text-xs font-semibold text-slate-400 uppercase tracking-wider mb-1.5 block">
                  Location
                </label>
                <div className="bg-slate-800/60 rounded-xl px-4 py-3 border border-slate-700/50 flex items-center gap-2">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#64748b" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M12 2C7.6 2 4 5.6 4 10c0 5.5 7.3 11.5 7.6 11.7.2.2.6.2.8 0C12.7 21.5 20 15.5 20 10c0-4.4-3.6-8-8-8z" />
                    <circle cx="12" cy="10" r="3" />
                  </svg>
                  <span className="text-sm text-slate-300 font-mono">
                    {createPostLat.toFixed(6)}, {createPostLng.toFixed(6)}
                  </span>
                </div>
              </div>

              {/* Error */}
              {error && (
                <motion.p
                  initial={{ opacity: 0, y: -4 }}
                  animate={{ opacity: 1, y: 0 }}
                  className="text-xs text-red-400 bg-red-500/10 rounded-lg px-3 py-2 border border-red-500/20"
                >
                  {error}
                </motion.p>
              )}

              {/* Submit */}
              <div className="pt-2 pb-4">
                <button
                  type="submit"
                  disabled={submitting || !title.trim()}
                  className={cn(
                    "w-full py-3 rounded-xl text-sm font-bold transition-all duration-200 flex items-center justify-center gap-2",
                    submitting || !title.trim()
                      ? "bg-slate-800 text-slate-500 cursor-not-allowed"
                      : "bg-blue-500 text-white shadow-lg shadow-blue-500/25 hover:bg-blue-400 active:scale-[0.98]",
                  )}
                >
                  {submitting ? (
                    <>
                      <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                      Creating...
                    </>
                  ) : (
                    <>
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                        <path d="M12 5v14M5 12h14" />
                      </svg>
                      Post to Map
                    </>
                  )}
                </button>
              </div>
            </form>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  )
}
