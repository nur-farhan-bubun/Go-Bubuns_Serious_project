"use client"

import { create } from "zustand"
import type { ThreadComment, DrawerState } from "../types/store"

// ─── Discord-Inspired Color Palette ─────────────────────────────────────
// These mimic Discord's vibrant role colors — each user gets a unique one.

const DISCORD_COLORS = [
  "#FF73FA", // Pink
  "#ED4245", // Red
  "#57F287", // Green
  "#FEE75C", // Yellow
  "#5865F2", // Blurple (Discord's brand color)
  "#EB459E", // Fuchsia
  "#00D26A", // Dark Green
  "#FFA06A", // Salmon
  "#9B59B6", // Purple
  "#1ABC9C", // Teal
  "#3498DB", // Blue
  "#E67E22", // Orange
  "#2ECC71", // Emerald
  "#E91E63", // Hot Pink
  "#00BCD4", // Cyan
  "#FF5722", // Deep Orange
  "#8E44AD", // Dark Purple
  "#2ECC71", // Mint
  "#F39C12", // Gold
  "#00A8FF", // Bright Blue
]

function getUserColor(userId: string): string {
  let hash = 0
  for (let i = 0; i < userId.length; i++) {
    hash = userId.charCodeAt(i) + ((hash << 5) - hash)
  }
  return DISCORD_COLORS[Math.abs(hash) % DISCORD_COLORS.length]
}

function getInitials(name: string): string {
  return name
    .split(" ")
    .map((w) => w[0])
    .join("")
    .toUpperCase()
    .slice(0, 2)
}

export { getUserColor, getInitials }

// ─── Seed Comments (per post) ───────────────────────────────────────────

function generateSeedComments(postId: string): ThreadComment[] {
  const baseComments: Record<string, ThreadComment[]> = {
    // ── RESTAURANTS ──────────────────────────────────────────────────
    "seed-rest-01": [
      {
        id: "c1",
        userId: "user-localguide",
        userName: "Local Guide",
        content: "Their latte art is incredible! The barista made a perfect swan today 🦢",
        createdAt: new Date(Date.now() - 3600000).toISOString(),
      },
      {
        id: "c2",
        userId: "user-coffeelover",
        userName: "Coffee Lover",
        content: "Don't miss the pistachio croissant 🥐 Best pastry in the city!",
        createdAt: new Date(Date.now() - 1800000).toISOString(),
      },
      {
        id: "c3",
        userId: "user-foodie",
        userName: "Foodie SF",
        content: "Pro tip: go on weekdays before 10am to avoid the line",
        createdAt: new Date(Date.now() - 600000).toISOString(),
      },
      {
        id: "c3-reply-1",
        parentId: "c3",
        replyToName: "Foodie SF",
        userId: "user-coffeelover",
        userName: "Coffee Lover",
        content: "Seconded! The line moves fast too, never waited more than 10 min ☕",
        createdAt: new Date(Date.now() - 300000).toISOString(),
      },
      {
        id: "c3-reply-2",
        parentId: "c3",
        replyToName: "Foodie SF",
        userId: "user-baker",
        userName: "Baker Betty",
        content: "And their seasonal specials are always worth trying! The pumpkin latte is 🔥",
        createdAt: new Date(Date.now() - 120000).toISOString(),
      },
    ],
    "seed-rest-02": [
      {
        id: "c4",
        userId: "user-foodie",
        userName: "Foodie SF",
        content: "The quail is life-changing 🤯 I dream about it",
        createdAt: new Date(Date.now() - 7200000).toISOString(),
      },
      {
        id: "c5",
        userId: "user-chefextra",
        userName: "Chef Extra",
        content: "Make a reservation weeks in advance! It's always booked solid.",
        createdAt: new Date(Date.now() - 3600000).toISOString(),
      },
      {
        id: "c6",
        userId: "user-localguide",
        userName: "Local Guide",
        content: "The chef's tasting menu is an absolute steal at $89",
        createdAt: new Date(Date.now() - 2400000).toISOString(),
      },
      {
        id: "c6-reply-1",
        parentId: "c6",
        replyToName: "Local Guide",
        userId: "user-foodie",
        userName: "Foodie SF",
        content: "Right? 7 courses for $89 is insane value for that quality 🤤",
        createdAt: new Date(Date.now() - 1200000).toISOString(),
      },
      {
        id: "c6-reply-2",
        parentId: "c6-reply-1",
        replyToName: "Foodie SF",
        userId: "user-chefextra",
        userName: "Chef Extra",
        content: "The wine pairing is another $45 but SO worth it. Trust me.",
        createdAt: new Date(Date.now() - 600000).toISOString(),
      },
      {
        id: "c7",
        userId: "user-nightowl",
        userName: "Night Owl",
        content: "Does anyone know if they accommodate gluten-free?",
        createdAt: new Date(Date.now() - 1200000).toISOString(),
      },
      {
        id: "c8",
        userId: "user-chefextra",
        userName: "Chef Extra",
        parentId: "c7",
        replyToName: "Night Owl",
        content: "Yes! They have an amazing GF bread alternative. Just mention it when booking.",
        createdAt: new Date(Date.now() - 600000).toISOString(),
      },
      {
        id: "c8-reply-1",
        parentId: "c8",
        replyToName: "Chef Extra",
        userId: "user-nightowl",
        userName: "Night Owl",
        content: "Amazing, thanks! Booking now 🙌",
        createdAt: new Date(Date.now() - 300000).toISOString(),
      },
    ],
    "seed-rest-03": [
      {
        id: "c9",
        userId: "user-baker",
        userName: "Baker Betty",
        content: "The morning bun is pure heaven. Get there early before they sell out!",
        createdAt: new Date(Date.now() - 10800000).toISOString(),
      },
      {
        id: "c10",
        userId: "user-localguide",
        userName: "Local Guide",
        content: "Best sourdough in SF, no contest. Take a loaf to go 🥖",
        createdAt: new Date(Date.now() - 7200000).toISOString(),
      },
      {
        id: "c11",
        userId: "user-foodie",
        userName: "Foodie SF",
        content: "Weekend lines are crazy but they move fast. Totally worth it.",
        createdAt: new Date(Date.now() - 1800000).toISOString(),
      },
    ],
    "seed-rest-04": [
      {
        id: "c12",
        userId: "user-pizzalover",
        userName: "Pizza Fanatic",
        content: "This is the most authentic Neapolitan pizza outside of Italy 🇮🇹",
        createdAt: new Date(Date.now() - 14400000).toISOString(),
      },
      {
        id: "c13",
        userId: "user-nightowl",
        userName: "Night Owl",
        content: "The Margherita D.O.C. is perfection. Simple ingredients done right.",
        createdAt: new Date(Date.now() - 7200000).toISOString(),
      },
      {
        id: "c14",
        userId: "user-chefextra",
        userName: "Chef Extra",
        content: "Tony won 13 international pizza awards! You can taste why.",
        createdAt: new Date(Date.now() - 3600000).toISOString(),
      },
      {
        id: "c15",
        userId: "user-baker",
        userName: "Baker Betty",
        content: "Cash only! There's an ATM inside but save the fee",
        createdAt: new Date(Date.now() - 600000).toISOString(),
      },
    ],
    "seed-rest-05": [
      {
        id: "c16",
        userId: "user-foodie",
        userName: "Foodie SF",
        content: "The tea leaf salad is an absolute must-order. Life-changing flavors!",
        createdAt: new Date(Date.now() - 7200000).toISOString(),
      },
      {
        id: "c17",
        userId: "user-localguide",
        userName: "Local Guide",
        content: "Get the rainbow salad and coconut rice. You'll thank me later.",
        createdAt: new Date(Date.now() - 5400000).toISOString(),
      },
      {
        id: "c18",
        userId: "user-pizzalover",
        userName: "Pizza Fanatic",
        content: "The line wraps around the block but it's 100% worth the wait",
        createdAt: new Date(Date.now() - 3600000).toISOString(),
      },
    ],
    "seed-rest-06": [
      {
        id: "c19",
        userId: "user-coffeelover",
        userName: "Coffee Lover",
        content: "Old-school SF at its finest. Sit at the counter and chat with the staff!",
        createdAt: new Date(Date.now() - 10800000).toISOString(),
      },
      {
        id: "c20",
        userId: "user-foodie",
        userName: "Foodie SF",
        content: "Cracked crab and a glass of white wine = the perfect afternoon",
        createdAt: new Date(Date.now() - 7200000).toISOString(),
      },
      {
        id: "c21",
        userId: "user-chefextra",
        userName: "Chef Extra",
        content: "Been coming here for 20 years. Never changed, never needed to.",
        createdAt: new Date(Date.now() - 1800000).toISOString(),
      },
    ],
    "seed-rest-07": [
      {
        id: "c22",
        userId: "user-chefextra",
        userName: "Chef Extra",
        content: "The roast chicken takes an hour to prepare. ORDER IT FIRST THING.",
        createdAt: new Date(Date.now() - 14400000).toISOString(),
      },
      {
        id: "c23",
        userId: "user-nightowl",
        userName: "Night Owl",
        content: "Their burger is also incredible — don't sleep on it",
        createdAt: new Date(Date.now() - 7200000).toISOString(),
      },
      {
        id: "c24",
        userId: "user-localguide",
        userName: "Local Guide",
        content: "The Caesar salad is made tableside. Old Hollywood vibes 🎬",
        createdAt: new Date(Date.now() - 2400000).toISOString(),
      },
    ],

    // ── SOCIAL LIFE ──────────────────────────────────────────────────
    "seed-social-01": [
      {
        id: "c25",
        userId: "user-parkgoer",
        userName: "Park Goer",
        content: "Best sunset spot in the city 🌅 Bring a blanket and some wine!",
        createdAt: new Date(Date.now() - 14400000).toISOString(),
      },
      {
        id: "c26",
        userId: "user-localguide",
        userName: "Local Guide",
        content: "Sunday afternoons are magical — everyone's out playing music and having picnics",
        createdAt: new Date(Date.now() - 7200000).toISOString(),
      },
      {
        id: "c27",
        userId: "user-baker",
        userName: "Baker Betty",
        content: "The Bi-Rite ice cream across the street is the perfect park snack 🍦",
        createdAt: new Date(Date.now() - 3600000).toISOString(),
      },
      {
        id: "c28",
        userId: "user-nightowl",
        userName: "Night Owl",
        content: "Great people-watching. You'll see everything here!",
        createdAt: new Date(Date.now() - 600000).toISOString(),
      },
    ],
    "seed-social-02": [
      {
        id: "c29",
        userId: "user-coffeelover",
        userName: "Coffee Lover",
        content: "Saturday market is the best. Get the mushroom brie sandwich 🧀",
        createdAt: new Date(Date.now() - 21600000).toISOString(),
      },
      {
        id: "c30",
        userId: "user-foodie",
        userName: "Foodie SF",
        content: "Blue Bottle inside, Hog Island oysters outside. Perfect date spot.",
        createdAt: new Date(Date.now() - 10800000).toISOString(),
      },
      {
        id: "c31",
        userId: "user-localguide",
        userName: "Local Guide",
        content: "Don't miss the sourdough bread bowl from the vendor near the water",
        createdAt: new Date(Date.now() - 5400000).toISOString(),
      },
      {
        id: "c32",
        userId: "user-pizzalover",
        userName: "Pizza Fanatic",
        content: "Tuesday is the quietest day to go if you want to avoid crowds",
        createdAt: new Date(Date.now() - 1800000).toISOString(),
      },
    ],
    "seed-social-03": [
      {
        id: "c33",
        userId: "user-parkgoer",
        userName: "Park Goer",
        content: "The Painted Ladies view is even better at golden hour 🌇",
        createdAt: new Date(Date.now() - 14400000).toISOString(),
      },
      {
        id: "c34",
        userId: "user-localguide",
        userName: "Local Guide",
        content: "Pro tip: bring a picnic and watch the sunset over the city skyline",
        createdAt: new Date(Date.now() - 7200000).toISOString(),
      },
      {
        id: "c35",
        userId: "user-baker",
        userName: "Baker Betty",
        content: "The grass is always filled with people playing frisbee and lounging",
        createdAt: new Date(Date.now() - 1200000).toISOString(),
      },
    ],
    "seed-social-04": [
      {
        id: "c36",
        userId: "user-localguide",
        userName: "Local Guide",
        content: "Clarion Alley has the most incredible political murals. Take your time!",
        createdAt: new Date(Date.now() - 21600000).toISOString(),
      },
      {
        id: "c37",
        userId: "user-nightowl",
        userName: "Night Owl",
        content: "Go on a Saturday when all the local artist studios are open 🎨",
        createdAt: new Date(Date.now() - 10800000).toISOString(),
      },
      {
        id: "c38",
        userId: "user-foodie",
        userName: "Foodie SF",
        content: "Grab a burrito from Taqueria El Farolito after the walk! 🌯",
        createdAt: new Date(Date.now() - 3600000).toISOString(),
      },
    ],
    "seed-social-05": [
      {
        id: "c39",
        userId: "user-parkgoer",
        userName: "Park Goer",
        content: "This rooftop is SF's best kept secret! The views are stunning 🤫",
        createdAt: new Date(Date.now() - 14400000).toISOString(),
      },
      {
        id: "c40",
        userId: "user-baker",
        userName: "Baker Betty",
        content: "Free yoga on Wednesdays at noon. Best lunch break ever.",
        createdAt: new Date(Date.now() - 7200000).toISOString(),
      },
      {
        id: "c41",
        userId: "user-coffeelover",
        userName: "Coffee Lover",
        content: "The native plant garden is beautiful and educational",
        createdAt: new Date(Date.now() - 1800000).toISOString(),
      },
    ],

    // ── EVENTS ───────────────────────────────────────────────────────
    "seed-event-01": [
      {
        id: "c42",
        userId: "user-jazzfan",
        userName: "Jazz Fan",
        content: "The acoustics are phenomenal! Every seat has a great view.",
        createdAt: new Date(Date.now() - 86400000).toISOString(),
      },
      {
        id: "c43",
        userId: "user-nightowl",
        userName: "Night Owl",
        content: "Going tonight — anyone want to join? Got an extra ticket 🎵",
        createdAt: new Date(Date.now() - 43200000).toISOString(),
      },
      {
        id: "c44",
        userId: "user-musiclover",
        userName: "Music Lover",
        content: "Check out their Sunday brunch jazz series — it's incredible",
        createdAt: new Date(Date.now() - 7200000).toISOString(),
      },
      {
        id: "c45",
        userId: "user-localguide",
        userName: "Local Guide",
        content: "Student discounts available at the door with ID!",
        createdAt: new Date(Date.now() - 3600000).toISOString(),
      },
    ],
    "seed-event-02": [
      {
        id: "c46",
        userId: "user-baseballfan",
        userName: "Baseball Fan",
        content: "Fireworks nights are the BEST. Get seats on the 3rd base side 🎆",
        createdAt: new Date(Date.now() - 14400000).toISOString(),
      },
      {
        id: "c47",
        userId: "user-foodie",
        userName: "Foodie SF",
        content: "Garlic fries and a Ghirardelli sundae = the real reason I come",
        createdAt: new Date(Date.now() - 7200000).toISOString(),
      },
      {
        id: "c48",
        userId: "user-nightowl",
        userName: "Night Owl",
        content: "Pro tip: bring a jacket even in summer. The fog rolls in fast!",
        createdAt: new Date(Date.now() - 2400000).toISOString(),
      },
      {
        id: "c49",
        userId: "user-localguide",
        userName: "Local Guide",
        content: "Knothole tickets are cheapest and still have great views",
        createdAt: new Date(Date.now() - 600000).toISOString(),
      },
    ],
    "seed-event-03": [
      {
        id: "c50",
        userId: "user-nightowl",
        userName: "Night Owl",
        content: "After Hours at SFMOMA is the best date night in the city 🖼️",
        createdAt: new Date(Date.now() - 10800000).toISOString(),
      },
      {
        id: "c51",
        userId: "user-musiclover",
        userName: "Music Lover",
        content: "The DJ sets are surprisingly good! Great vibe all around.",
        createdAt: new Date(Date.now() - 7200000).toISOString(),
      },
      {
        id: "c52",
        userId: "user-baker",
        userName: "Baker Betty",
        content: "The rooftop sculpture garden is open during these events too",
        createdAt: new Date(Date.now() - 3600000).toISOString(),
      },
    ],
    "seed-event-04": [
      {
        id: "c53",
        userId: "user-musiclover",
        userName: "Music Lover",
        content: "Three days of FREE music in Golden Gate Park. Unreal lineup this year!",
        createdAt: new Date(Date.now() - 86400000).toISOString(),
      },
      {
        id: "c54",
        userId: "user-parkgoer",
        userName: "Park Goer",
        content: "Bring sunscreen, a blanket, and snacks. There are 6 stages!",
        createdAt: new Date(Date.now() - 43200000).toISOString(),
      },
      {
        id: "c55",
        userId: "user-foodie",
        userName: "Foodie SF",
        content: "The food vendor selection is incredible. Best festival food in SF.",
        createdAt: new Date(Date.now() - 7200000).toISOString(),
      },
      {
        id: "c56",
        userId: "user-localguide",
        userName: "Local Guide",
        content: "Bike there! They have free bike valet parking 🚲",
        createdAt: new Date(Date.now() - 1800000).toISOString(),
      },
    ],
    "seed-event-05": [
      {
        id: "c57",
        userId: "user-musiclover",
        userName: "Music Lover",
        content: "Casablanca at the Castro is a religious experience. Go! 🎬",
        createdAt: new Date(Date.now() - 7200000).toISOString(),
      },
      {
        id: "c58",
        userId: "user-nightowl",
        userName: "Night Owl",
        content: "The organ player before the show is worth the ticket alone",
        createdAt: new Date(Date.now() - 3600000).toISOString(),
      },
      {
        id: "c59",
        userId: "user-localguide",
        userName: "Local Guide",
        content: "Arrive early to admire the architecture. It's a stunning venue.",
        createdAt: new Date(Date.now() - 1200000).toISOString(),
      },
    ],
    "seed-event-06": [
      {
        id: "c60",
        userId: "user-foodie",
        userName: "Foodie SF",
        content: "Best food truck gathering in the city! The lobster roll truck is elite 🦞",
        createdAt: new Date(Date.now() - 14400000).toISOString(),
      },
      {
        id: "c61",
        userId: "user-musiclover",
        userName: "Music Lover",
        content: "Live music + bay view + amazing food = perfect Friday night",
        createdAt: new Date(Date.now() - 7200000).toISOString(),
      },
      {
        id: "c62",
        userId: "user-parkgoer",
        userName: "Park Goer",
        content: "The sunset views over the Golden Gate Bridge from here are unreal 🌉",
        createdAt: new Date(Date.now() - 3600000).toISOString(),
      },
      {
        id: "c63",
        userId: "user-baker",
        userName: "Baker Betty",
        content: "Bring the whole family! They have a kids' zone and lawn games",
        createdAt: new Date(Date.now() - 600000).toISOString(),
      },
    ],
  }

  // Fallback for any post without seed data
  return baseComments[postId] ?? []
}

// ─── Helper: toggle vote logic ──────────────────────────────────────────

function toggleVote(
  currentVote: "up" | "down" | null,
  newVote: "up" | "down",
  currentScore: number,
): { vote: "up" | "down" | null; score: number } {
  if (currentVote === newVote) {
    // Un-vote
    return { vote: null, score: currentScore + (newVote === "up" ? -1 : 1) }
  }
  if (currentVote === null) {
    // Fresh vote
    return { vote: newVote, score: currentScore + (newVote === "up" ? 1 : -1) }
  }
  // Switching vote
  return { vote: newVote, score: currentScore + (newVote === "up" ? 2 : -2) }
}

// ─── Store ──────────────────────────────────────────────────────────────

export const useMapStore = create<DrawerState>((set, get) => ({
  // ── Defaults ───────────────────────────────────────────────────────
  selectedPost: null,
  isDrawerOpen: false,
  activeSnapPoint: 0.35,
  mapScale: 1,
  mapRotateX: 0,
  mapBlur: 0,
  isExpanded: false,
  comments: [],

  // ── Voting defaults ────────────────────────────────────────────────
  postVotes: {},
  postUserVote: {},
  commentVotes: {},
  commentUserVote: {},
  postLikes: {},
  shareCount: 0,
  isCreatePostOpen: false,
  createPostLat: 37.7749,
  createPostLng: -122.4194,

  // ── Actions ────────────────────────────────────────────────────────

  openPost: (post) =>
    set({
      selectedPost: post,
      isDrawerOpen: true,
      activeSnapPoint: 0.35,
      mapScale: 0.92,
      mapRotateX: 4,
      mapBlur: 1,
      isExpanded: false,
      comments: generateSeedComments(post.id),
    }),

  closeDrawer: () =>
    set({
      selectedPost: null,
      isDrawerOpen: false,
      activeSnapPoint: 0.35,
      mapScale: 1,
      mapRotateX: 0,
      mapBlur: 0,
      isExpanded: false,
      comments: [],
    }),

  setSnapPoint: (snap) => {
    if (snap === null) return

    // Ignore snap updates during exit animation (AnimatePresence)
    if (!get().isDrawerOpen) return

    const numericSnap = typeof snap === "number" ? snap : parseFloat(snap)
    const isExpanded = numericSnap >= 0.85

    set({
      activeSnapPoint: snap,
      isExpanded,
      mapScale: isExpanded ? 0.82 : 0.92,
      mapRotateX: isExpanded ? 15 : 4,
      mapBlur: isExpanded ? 4 : 1,
    })
  },

  addComment: (content: string, parentId?: string, replyToName?: string) => {
    const { comments, selectedPost } = get()
    if (!selectedPost) return

    const newComment: ThreadComment = {
      id: `c-${Date.now()}`,
      userId: "user-sim-001",
      userName: "You",
      content,
      createdAt: new Date().toISOString(),
      parentId,
      replyToName,
    }

    set({ comments: [...comments, newComment] })
  },

  // ── Vote actions ──────────────────────────────────────────────────

  upvotePost: (postId) =>
    set((state) => {
      const current = state.postUserVote[postId] ?? null
      const currentScore = state.postVotes[postId] ?? 0
      const { vote, score } = toggleVote(current, "up", currentScore)
      return {
        postUserVote: { ...state.postUserVote, [postId]: vote },
        postVotes: { ...state.postVotes, [postId]: score },
      }
    }),

  downvotePost: (postId) =>
    set((state) => {
      const current = state.postUserVote[postId] ?? null
      const currentScore = state.postVotes[postId] ?? 0
      const { vote, score } = toggleVote(current, "down", currentScore)
      return {
        postUserVote: { ...state.postUserVote, [postId]: vote },
        postVotes: { ...state.postVotes, [postId]: score },
      }
    }),

  upvoteComment: (commentId) =>
    set((state) => {
      const current = state.commentUserVote[commentId] ?? null
      const currentScore = state.commentVotes[commentId] ?? 0
      const { vote, score } = toggleVote(current, "up", currentScore)
      return {
        commentUserVote: { ...state.commentUserVote, [commentId]: vote },
        commentVotes: { ...state.commentVotes, [commentId]: score },
      }
    }),

  downvoteComment: (commentId) =>
    set((state) => {
      const current = state.commentUserVote[commentId] ?? null
      const currentScore = state.commentVotes[commentId] ?? 0
      const { vote, score } = toggleVote(current, "down", currentScore)
      return {
        commentUserVote: { ...state.commentUserVote, [commentId]: vote },
        commentVotes: { ...state.commentVotes, [commentId]: score },
      }
    }),

  // ── Like / Share ──────────────────────────────────────────────────

  toggleLikePost: (postId) =>
    set((state) => ({
      postLikes: {
        ...state.postLikes,
        [postId]: !state.postLikes[postId],
      },
    })),

  // ── Create Post ──────────────────────────────────────────────────

  openCreatePost: (lat, lng) =>
    set({ isCreatePostOpen: true, createPostLat: lat, createPostLng: lng }),

  closeCreatePost: () =>
    set({ isCreatePostOpen: false }),

  sharePost: () => {
    const { selectedPost } = get()
    if (!selectedPost) return

    const text = `📍 ${selectedPost.title}\n${selectedPost.content || ""}\n\n${window.location.origin}/?post=${selectedPost.id}`

    if (navigator.share) {
      navigator.share({
        title: selectedPost.title,
        text: selectedPost.content || "",
        url: `${window.location.origin}/?post=${selectedPost.id}`,
      }).catch((e) => {
        if (e instanceof DOMException && e.name === "AbortError") return // user cancelled
        console.warn("Share failed:", e)
      })
    } else {
      navigator.clipboard.writeText(text).catch((e) => console.warn("Clipboard write failed:", e))
    }

    set((state) => ({ shareCount: state.shareCount + 1 }))
  },
}))
