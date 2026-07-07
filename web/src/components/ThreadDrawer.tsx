"use client"

import { useState, useRef, useEffect } from "react"
import Image from "next/image"
import {
  motion,
  AnimatePresence,
  useMotionValue,
  useAnimation,
  useMotionValueEvent,
} from "framer-motion"
import { useMapStore, getUserColor, getInitials } from "../lib/useMapStore"
import type { ThreadComment } from "../types/store"
import { POST_CATEGORIES } from "../lib/location"
import { cn } from "../lib/utils"

// ─── Time formatting ────────────────────────────────────────────────────

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return "just now"
  if (mins < 60) return `${mins}m ago`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.floor(hours / 24)}d ago`
}

// ─── Snap points ────────────────────────────────────────────────────────

const SNAP_PEEK = 0.35 // 35% — initial peek
const SNAP_FULL = 0.92 // 92% — expanded

// ─── Animation curves ───────────────────────────────────────────────────

const SPRING_OPEN = {
  type: "spring" as const,
  damping: 18,
  stiffness: 250,
  mass: 0.9,
}

const SPRING_SNAP = {
  type: "spring" as const,
  damping: 28,
  stiffness: 400,
  mass: 0.6,
}

const SPRING_DRAG = {
  type: "spring" as const,
  damping: 35,
  stiffness: 500,
  mass: 0.5,
}

const OVERLAY_TRANSITION = {
  duration: 0.3,
  ease: [0.4, 0, 0.2, 1] as [number, number, number, number],
}

// ─── Snap interpolation ─────────────────────────────────────────────────

function snapToY(snap: number): number {
  if (typeof window === "undefined") return 0
  return Math.round((1 - snap) * window.innerHeight)
}

function yToSnap(y: number): number {
  if (typeof window === "undefined") return 0
  return 1 - y / window.innerHeight
}

// ─── Vote arrows SVG ────────────────────────────────────────────────────

function UpArrow({ active }: { active: boolean }) {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill={active ? "#FF4500" : "none"} stroke={active ? "#FF4500" : "#64748b"} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="transition-colors">
      <path d="M12 19V5M5 12l7-7 7 7" />
    </svg>
  )
}

function DownArrow({ active }: { active: boolean }) {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill={active ? "#7193FF" : "none"} stroke={active ? "#7193FF" : "#64748b"} strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="transition-colors">
      <path d="M12 5v14M19 12l-7 7-7-7" />
    </svg>
  )
}

// ─── Reddit-style number formatting ─────────────────────────────────────

function formatScore(n: number): string {
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`
  return String(n)
}

// ─── Component ──────────────────────────────────────────────────────────

export default function ThreadDrawer() {
  const {
    selectedPost,
    isDrawerOpen,
    comments,
    closeDrawer,
    addComment,
    setSnapPoint,
    postVotes,
    postUserVote,
    commentVotes,
    commentUserVote,
    postLikes,
    upvotePost,
    downvotePost,
    upvoteComment,
    downvoteComment,
    toggleLikePost,
    sharePost,
  } = useMapStore()

  const [replyText, setReplyText] = useState("")
  const [expandedImage, setExpandedImage] = useState<string | null>(null)
  const [replyTarget, setReplyTarget] = useState<{ id: string; name: string } | null>(null)
  const replyInputRef = useRef<HTMLInputElement>(null)
  const commentListRef = useRef<HTMLDivElement>(null)
  const vhRef = useRef(typeof window !== "undefined" ? window.innerHeight : 900)

  // ─── Framer Motion Controls ───────────────────────────────────────────
  const y = useMotionValue(0)
  const controls = useAnimation()

  const [exitY, setExitY] = useState(snapToY(SNAP_FULL))
  const updateMapRef = useRef(setSnapPoint)

  useEffect(() => {
    updateMapRef.current = setSnapPoint
  }, [setSnapPoint])

  useMotionValueEvent(y, "change", (latest) => {
    if (!isDrawerOpen) return
    const snap = Math.max(0, Math.min(1, yToSnap(latest)))
    updateMapRef.current(snap)
  })

  // ─── Auto-scroll to latest comment ────────────────────────────────────
  useEffect(() => {
    if (commentListRef.current) {
      commentListRef.current.scrollTop = commentListRef.current.scrollHeight
    }
  }, [comments.length])

  useEffect(() => {
    if (!isDrawerOpen) {
      setReplyText("")
      setExpandedImage(null)
      setReplyTarget(null)
    }
  }, [isDrawerOpen])

  useEffect(() => {
    function handleResize() {
      vhRef.current = window.innerHeight
      if (isDrawerOpen) {
        const currentSnap = yToSnap(y.get())
        controls.start({ y: snapToY(currentSnap) })
      }
    }
    window.addEventListener("resize", handleResize)
    return () => window.removeEventListener("resize", handleResize)
  }, [isDrawerOpen, controls, y])

  useEffect(() => {
    if (isDrawerOpen) {
      const peekY = snapToY(SNAP_PEEK)
      setExitY(vhRef.current * 1.2)
      controls.start({ y: peekY }, SPRING_OPEN)
    }
  }, [isDrawerOpen, controls])

  const cat = selectedPost
    ? POST_CATEGORIES[selectedPost.category]
    : null

  // ─── Derived post vote state ──────────────────────────────────────────
  const postId = selectedPost?.id ?? ""
  const postScore = postVotes[postId] ?? 0
  const userPostVote = postUserVote[postId] ?? null
  const isLiked = postLikes[postId] ?? false

  function handleReplyTo(commentId: string, userName: string) {
    setReplyTarget({ id: commentId, name: userName })
    setReplyText(`@${userName} `)
    setTimeout(() => replyInputRef.current?.focus(), 100)
  }

  function cancelReply() {
    setReplyTarget(null)
    setReplyText("")
  }

  function handleSubmitReply(e: React.FormEvent) {
    e.preventDefault()
    const trimmed = replyText.trim()
    if (!trimmed) return

    // Strip the @mention prefix for cleanliness in the stored content
    const cleanContent = replyTarget
      ? trimmed.replace(new RegExp(`^@${replyTarget.name}\s*`), "").trim()
      : trimmed

    addComment(cleanContent || trimmed, replyTarget?.id, replyTarget?.name)
    setReplyText("")
    setReplyTarget(null)
  }

  function handleDragEnd(
    _: unknown,
    info: { offset: { y: number }; velocity: { y: number } },
  ) {
    const currentY = y.get()
    const currentSnap = yToSnap(currentY)

    if (info.velocity.y > 300 || info.offset.y > vhRef.current * 0.2) {
      closeDrawer()
      return
    }

    const midpoint = (SNAP_PEEK + SNAP_FULL) / 2
    const targetSnap = currentSnap < midpoint ? SNAP_PEEK : SNAP_FULL
    const targetY = snapToY(targetSnap)

    setSnapPoint(targetSnap)
    const dist = Math.abs(currentY - targetY)
    const spring = dist < 80 ? SPRING_SNAP : SPRING_DRAG
    controls.start({ y: targetY }, spring)
  }

  function handleClose() {
    closeDrawer()
  }

  function handleOverlayClick() {
    handleClose()
  }

  const sheetHeight = SNAP_FULL * vhRef.current

  return (
    <AnimatePresence>
      {isDrawerOpen && selectedPost && (
        <>
          {/* ─── Overlay ────────────────────────────────────────────── */}
          <motion.div
            key="drawer-overlay"
            className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={OVERLAY_TRANSITION}
            onClick={handleOverlayClick}
          />

          {/* ─── Full-Screen Lightbox ───────────────────────────────── */}
          <AnimatePresence>
            {expandedImage && (
              <motion.div
                key="image-lightbox"
                className="fixed inset-0 z-[60] flex items-center justify-center bg-black/90 backdrop-blur-xl"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                transition={{ duration: 0.2 }}
                onClick={() => setExpandedImage(null)}
              >
                <motion.button
                  className="absolute top-6 right-6 z-10 w-10 h-10 rounded-full bg-white/10 hover:bg-white/20 flex items-center justify-center text-white transition-colors"
                  onClick={() => setExpandedImage(null)}
                  whileHover={{ scale: 1.1 }}
                  whileTap={{ scale: 0.9 }}
                >
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M18 6 6 18M6 6l12 12" />
                  </svg>
                </motion.button>
                <Image
                  src={expandedImage}
                  alt=""
                  width={1200}
                  height={900}
                  className="max-h-[90vh] max-w-[90vw] object-contain rounded-2xl shadow-2xl"
                  style={{ width: "auto", height: "auto" }}
                  onClick={(e) => e.stopPropagation()}
                />
              </motion.div>
            )}
          </AnimatePresence>

          {/* ─── Sheet ──────────────────────────────────────────────── */}
          <motion.div
            key="drawer-sheet"
            className="fixed bottom-0 left-0 right-0 z-50 mx-auto flex flex-col rounded-t-[20px] bg-slate-900/95 backdrop-blur-2xl shadow-2xl border-t border-slate-700/60 outline-none overflow-hidden"
            style={{
              height: sheetHeight,
              maxHeight: "96dvh",
              y,
            }}
            drag="y"
            dragConstraints={{ top: 0, bottom: sheetHeight }}
            dragElastic={0.05}
            dragMomentum={false}
            onDragEnd={handleDragEnd}
            initial={{ y: vhRef.current }}
            animate={controls}
            exit={{ y: exitY, transition: { type: "tween", duration: 0.25, ease: [0.4, 0, 0.2, 1] } }}
            transition={SPRING_DRAG}
          >
            {/* ── Drag Handle ───────────────────────────────────────── */}
            <div className="flex items-center justify-center pt-3 pb-2 cursor-grab active:cursor-grabbing z-10 shrink-0">
              <div className="h-1.5 w-12 rounded-full bg-slate-600/60" />
            </div>

            {/* ─── REDDIT-STYLE POST ──────────────────────────────────── */}
            <div className="flex-1 overflow-y-auto overscroll-contain">
              <div className="flex">
                {/* ── Vote Column ──────────────────────────────────────── */}
                <div className="flex flex-col items-center gap-0.5 pt-4 px-3 shrink-0 w-12">
                  <button
                    onClick={() => upvotePost(postId)}
                    className={cn(
                      "p-1 rounded transition-all duration-150 hover:scale-110 active:scale-95",
                      userPostVote === "up"
                        ? "text-orange-500"
                        : "text-slate-500 hover:text-orange-400 hover:bg-orange-500/10",
                    )}
                    title="Upvote"
                  >
                    <UpArrow active={userPostVote === "up"} />
                  </button>

                  <span
                    className={cn(
                      "text-xs font-bold tabular-nums leading-none py-0.5 select-none",
                      userPostVote === "up"
                        ? "text-orange-500"
                        : userPostVote === "down"
                          ? "text-blue-400"
                          : "text-slate-400",
                    )}
                  >
                    {formatScore(postScore)}
                  </span>

                  <button
                    onClick={() => downvotePost(postId)}
                    className={cn(
                      "p-1 rounded transition-all duration-150 hover:scale-110 active:scale-95",
                      userPostVote === "down"
                        ? "text-blue-400"
                        : "text-slate-500 hover:text-blue-400 hover:bg-blue-500/10",
                    )}
                    title="Downvote"
                  >
                    <DownArrow active={userPostVote === "down"} />
                  </button>
                </div>

                {/* ── Post Content ──────────────────────────────────── */}
                <div className="flex-1 min-w-0 pt-3 pr-4 pb-2">
                  {/* Category Badge + Close */}
                  <div className="flex items-center justify-between mb-1">
                    <div className="flex items-center gap-2">
                      <div
                        className="w-2.5 h-2.5 rounded-full shrink-0"
                        style={{ backgroundColor: cat?.color ?? "#94a3b8" }}
                      />
                      <span
                        className="text-[10px] font-semibold uppercase tracking-widest"
                        style={{ color: cat?.color ?? "#94a3b8" }}
                      >
                        {cat?.icon} {cat?.label}
                      </span>
                    </div>

                    <button
                      onClick={handleClose}
                      className="w-7 h-7 flex items-center justify-center rounded-full hover:bg-slate-800 transition-colors text-slate-400 hover:text-white"
                    >
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                        <path d="M18 6 6 18M6 6l12 12" />
                      </svg>
                    </button>
                  </div>

                  {/* Title */}
                  <h2 className="text-lg font-bold text-white leading-tight">
                    {selectedPost.title}
                  </h2>

                  {/* Content */}
                  {selectedPost.content && (
                    <p className="text-sm text-slate-300 mt-1.5 leading-relaxed">
                      {selectedPost.content}
                    </p>
                  )}

                  {/* ── Image Gallery ───────────────────────────────── */}
                  {selectedPost.image_urls && selectedPost.image_urls.length > 0 && (
                    <div className="mt-3 flex flex-wrap gap-2">
                      {selectedPost.image_urls.map((url, i) => (
                        <motion.button
                          key={i}
                          className="relative group overflow-hidden rounded-xl border border-slate-700/50 bg-slate-800 cursor-pointer"
                          whileHover={{ scale: 1.03 }}
                          whileTap={{ scale: 0.97 }}
                          onClick={() => setExpandedImage(url)}
                        >
                          <Image
                            src={url}
                            alt=""
                            width={128}
                            height={128}
                            className="w-28 h-28 object-cover sm:w-32 sm:h-32"
                          />
                          <div className="absolute inset-0 bg-black/0 group-hover:bg-black/20 transition-colors flex items-center justify-center">
                            <svg
                              className="opacity-0 group-hover:opacity-100 transition-opacity text-white"
                              width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"
                            >
                              <circle cx="11" cy="11" r="8" />
                              <path d="m21 21-4.35-4.35" />
                              <path d="M11 8v6M8 11h6" />
                            </svg>
                          </div>
                        </motion.button>
                      ))}
                    </div>
                  )}

                  {/* ── Action Bar ───────────────────────────────────── */}
                  <div className="flex items-center gap-1 mt-3 -ml-1">
                    {/* Comments count */}
                    <button
                      className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
                      onClick={() => commentListRef.current?.scrollIntoView({ behavior: "smooth" })}
                    >
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                        <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
                      </svg>
                      {comments.length}
                    </button>

                    {/* Like button */}
                    <button
                      onClick={() => toggleLikePost(postId)}
                      className={cn(
                        "flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium transition-all duration-200",
                        isLiked
                          ? "text-red-400 hover:bg-red-500/10"
                          : "text-slate-400 hover:text-white hover:bg-slate-800",
                      )}
                    >
                      <motion.div
                        animate={isLiked ? { scale: [1, 1.3, 1] } : { scale: 1 }}
                        transition={{ duration: 0.3 }}
                      >
                        <svg
                          width="16" height="16"
                          viewBox="0 0 24 24"
                          fill={isLiked ? "currentColor" : "none"}
                          stroke="currentColor"
                          strokeWidth="2"
                          strokeLinecap="round"
                          strokeLinejoin="round"
                        >
                          <path d="M19 14c1.49-1.46 3-3.21 3-5.5A5.5 5.5 0 0 0 16.5 3c-1.76 0-3 .5-4.5 2-1.5-1.5-2.74-2-4.5-2A5.5 5.5 0 0 0 2 8.5c0 2.3 1.5 4.05 3 5.5l7 7Z" />
                        </svg>
                      </motion.div>
                      {isLiked ? "Liked" : "Like"}
                    </button>

                    {/* Share button */}
                    <button
                      onClick={sharePost}
                      className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium text-slate-400 hover:text-blue-400 hover:bg-blue-500/10 transition-colors"
                    >
                      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                        <circle cx="18" cy="5" r="3" />
                        <circle cx="6" cy="12" r="3" />
                        <circle cx="18" cy="19" r="3" />
                        <path d="m8.59 13.51 6.83 3.98M15.41 6.51 8.59 10.49" />
                      </svg>
                      Share
                    </button>

                    {/* Coordinates (metadata) */}
                    <span className="ml-auto text-[10px] text-slate-600 font-mono hidden sm:block">
                      {selectedPost.latitude.toFixed(4)}, {selectedPost.longitude.toFixed(4)}
                    </span>
                  </div>

                  {/* ── Divider ──────────────────────────────────────── */}
                  <div className="h-px bg-slate-800/80 mt-3" />
                </div>
              </div>

              {/* ── Comments Thread ────────────────────────────────── */}
              <div
                ref={commentListRef}
                className="px-4 py-2 space-y-0.5 scroll-smooth overscroll-contain"
              >
                {comments.length === 0 ? (
                  <div className="flex items-center justify-center py-8 text-slate-600 text-sm">
                    <p>No comments yet. Be the first!</p>
                  </div>
                ) : (
                  <>
                    {/* Build a map of parentId → children for rendering */}
                    {(() => {
                      // Sort top-level: newest first; replies: oldest first (chronological thread)
                      const topLevel = comments
                        .filter((c) => !c.parentId)
                        .sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())

                      const childrenByParent = new Map<string, ThreadComment[]>()
                      for (const c of comments) {
                        if (c.parentId) {
                          const list = childrenByParent.get(c.parentId) || []
                          list.push(c)
                          list.sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime())
                          childrenByParent.set(c.parentId, list)
                        }
                      }

                      // Recursively render a comment and its children
                      function renderComment(comment: ThreadComment, depth: number = 0) {
                        const cId = comment.id
                        const cScore = commentVotes[cId] ?? 0
                        const userCVote = commentUserVote[cId] ?? null
                        const color = getUserColor(comment.userId)
                        const children = childrenByParent.get(cId) || []

                        return (
                          <div key={cId}>
                            <motion.div
                              initial={{ opacity: 0, y: 8, scale: 0.97 }}
                              animate={{ opacity: 1, y: 0, scale: 1 }}
                              transition={{ duration: 0.2, ease: "easeOut" }}
                              className="flex gap-1.5 group py-1"
                            >
                              {/* Thread line for nested comments */}
                              {depth > 0 && (
                                <div className="relative w-5 shrink-0 flex justify-center">
                                  <div className="w-px h-full bg-slate-700/40" />
                                </div>
                              )}

                              {/* ── Vote column (hide at depth > 2 for space) ── */}
                              {depth <= 2 ? (
                                <div className="flex flex-col items-center gap-0.5 pt-1 shrink-0 w-6">
                                  <button
                                    onClick={() => upvoteComment(cId)}
                                    className={cn(
                                      "p-0.5 rounded transition-all hover:scale-110",
                                      userCVote === "up"
                                        ? "text-orange-500"
                                        : "text-slate-600 hover:text-orange-400",
                                    )}
                                  >
                                    <UpArrow active={userCVote === "up"} />
                                  </button>
                                  <span
                                    className={cn(
                                      "text-[9px] font-bold tabular-nums leading-none",
                                      userCVote === "up"
                                        ? "text-orange-500"
                                        : userCVote === "down"
                                          ? "text-blue-400"
                                          : "text-slate-500",
                                    )}
                                  >
                                    {formatScore(cScore)}
                                  </span>
                                  <button
                                    onClick={() => downvoteComment(cId)}
                                    className={cn(
                                      "p-0.5 rounded transition-all hover:scale-110",
                                      userCVote === "down"
                                        ? "text-blue-400"
                                        : "text-slate-600 hover:text-blue-400",
                                    )}
                                  >
                                    <DownArrow active={userCVote === "down"} />
                                  </button>
                                </div>
                              ) : (
                                <div className="w-6 shrink-0" />
                              )}

                              {/* ── Comment Content ──────────────── */}
                              <div className="flex-1 min-w-0">
                                <div className="flex items-center gap-1.5 flex-wrap">
                                  {/* Discord-style avatar */}
                                  <div
                                    className="w-5 h-5 rounded-full flex items-center justify-center text-[8px] font-bold text-white shrink-0 ring-1 ring-offset-1 ring-offset-slate-900"
                                    style={{ backgroundColor: color }}
                                  >
                                    {getInitials(comment.userName)}
                                  </div>
                                  {/* Username */}
                                  <span
                                    className="text-[11px] font-semibold"
                                    style={{ color }}
                                  >
                                    {comment.userName}
                                  </span>
                                  {/* Replying to indicator */}
                                  {comment.replyToName && depth <= 1 && (
                                    <span className="text-[10px] text-slate-500">
                                      replying to{' '}
                                      <span className="text-slate-400">@{comment.replyToName}</span>
                                    </span>
                                  )}
                                  <span className="text-[9px] text-slate-600 ml-auto">
                                    {timeAgo(comment.createdAt)}
                                  </span>
                                </div>
                                <p className="text-sm text-slate-300 mt-0.5 leading-relaxed">
                                  {comment.content}
                                </p>
                                {/* Reply button */}
                                <button
                                  onClick={() => handleReplyTo(cId, comment.userName)}
                                  className="text-[10px] font-medium text-slate-500 hover:text-slate-300 mt-0.5 opacity-0 group-hover:opacity-100 transition-opacity"
                                >
                                  Reply
                                </button>
                              </div>
                            </motion.div>

                            {/* Nested children */}
                            {children.length > 0 && (
                              <div className={depth === 0 ? "ml-1 border-l-2 border-slate-800/60 pl-1" : ""}>
                                {children.map((child) => renderComment(child, depth + 1))}
                              </div>
                            )}
                          </div>
                        )
                      }

                      return (
                        <AnimatePresence initial={false}>
                          {topLevel.map((comment) => renderComment(comment, 0))}
                        </AnimatePresence>
                      )
                    })()}
                  </>
                )}
              </div>
            </div>

            {/* ── Reply Input ────────────────────────────────────────── */}
            <div className="px-4 py-3 border-t border-slate-800/80 bg-slate-900/50 shrink-0">
              {/* Reply context bar */}
              {replyTarget && (
                <div className="flex items-center gap-2 mb-2 px-1">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="#64748b" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
                  </svg>
                  <span className="text-[11px] text-slate-400">
                    Replying to <span className="font-semibold text-slate-300">@{replyTarget.name}</span>
                  </span>
                  <button
                    type="button"
                    onClick={cancelReply}
                    className="ml-auto text-slate-500 hover:text-slate-300 transition-colors"
                  >
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                      <path d="M18 6 6 18M6 6l12 12" />
                    </svg>
                  </button>
                </div>
              )}
              <form onSubmit={handleSubmitReply} className="flex items-center gap-2">
                <input
                  ref={replyInputRef}
                  type="text"
                  value={replyText}
                  onChange={(e) => setReplyText(e.target.value)}
                  placeholder={replyTarget ? `Reply to @${replyTarget.name}...` : "Add a comment..."}
                  className="flex-1 bg-slate-800/80 text-white text-sm rounded-xl px-4 py-2.5 outline-none border border-slate-700/50 focus:border-slate-500/60 focus:bg-slate-800 placeholder:text-slate-500 transition-colors"
                />
                <button
                  type="submit"
                  disabled={!replyText.trim()}
                  className={cn(
                    "w-9 h-9 flex items-center justify-center rounded-xl transition-all duration-200",
                    replyText.trim()
                      ? "bg-blue-500 text-white shadow-lg shadow-blue-500/25 hover:bg-blue-400"
                      : "bg-slate-800 text-slate-600",
                  )}
                >
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M22 2 11 13" />
                    <path d="m22 2-7 20-4-9-9-4 20-7z" />
                  </svg>
                </button>
              </form>
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  )
}
