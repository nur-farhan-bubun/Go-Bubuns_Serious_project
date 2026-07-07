// ─── Store / State Types ────────────────────────────────────────────────

import type { MapPost } from "./location"

export interface ThreadComment {
  id: string
  userId: string
  userName: string
  content: string
  createdAt: string
  parentId?: string      // null = top-level comment
  replyToName?: string   // name of the user being replied to (for display)
}

export interface DrawerState {
  // ── Post / Thread ──────────────────────────────────────────────────
  selectedPost: MapPost | null
  isDrawerOpen: boolean

  // ── Snap & transform ───────────────────────────────────────────────
  activeSnapPoint: number | string
  mapScale: number
  mapRotateX: number
  mapBlur: number
  isExpanded: boolean

  // ── Comments ───────────────────────────────────────────────────────
  comments: ThreadComment[]

  // ── Voting ─────────────────────────────────────────────────────────
  postVotes: Record<string, number>
  postUserVote: Record<string, "up" | "down" | null>
  commentVotes: Record<string, number>
  commentUserVote: Record<string, "up" | "down" | null>

  // ── Likes, Shares & Create Post ──────────────────────────────────
  postLikes: Record<string, boolean>
  shareCount: number
  isCreatePostOpen: boolean
  createPostLat: number
  createPostLng: number

  // ── Actions ────────────────────────────────────────────────────────
  openPost: (post: MapPost) => void
  closeDrawer: () => void
  setSnapPoint: (snap: number | string | null) => void
  addComment: (content: string, parentId?: string, replyToName?: string) => void
  openCreatePost: (lat: number, lng: number) => void
  closeCreatePost: () => void

  upvotePost: (postId: string) => void
  downvotePost: (postId: string) => void
  upvoteComment: (commentId: string) => void
  downvoteComment: (commentId: string) => void

  toggleLikePost: (postId: string) => void
  sharePost: () => void
}
