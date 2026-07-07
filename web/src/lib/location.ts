// ─── Location Service API Client ─────────────────────────────────────────
// Sends real-time location updates to the backend and simulates movement
// when no real GPS is available.

import type { LocationUpdate, MapPost, SimulatedUser, PostCategory } from "../types/location"

// ─── Configuration ──────────────────────────────────────────────────────

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"
const DEFAULT_USER_ID = "user-sim-001"

// ─── Logging ────────────────────────────────────────────────────────────
// Logs go to the Next.js dev server terminal via console.log.
// Each log is prefixed with [LOCATION] for easy filtering.

function log(type: string, data: Record<string, unknown>) {
  const entry = {
    ts: new Date().toISOString(),
    type,
    ...data,
  }
  // Server-side terminal log (Next.js dev server stdout)
  console.log(`\n📍 [LOCATION] ${JSON.stringify(entry, null, 2)}`)
}

// ─── API Calls ──────────────────────────────────────────────────────────

/**
 * Update the user's location on the backend.
 * PUT /v1/location → Location Service (via API Gateway)
 */
export async function updateLocation(
  latitude: number,
  longitude: number,
  userID: string = DEFAULT_USER_ID,
): Promise<boolean> {
  const body: LocationUpdate = { user_id: userID, latitude, longitude }

  log("UPDATE_LOCATION", { user_id: userID, latitude, longitude })

  try {
    const res = await fetch(`${API_BASE}/v1/location`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    })

    if (!res.ok) {
      console.warn(`⚠️ [LOCATION] PUT /v1/location failed: ${res.status}`)
      return false
    }

    log("UPDATE_SUCCESS", { status: res.status })
    return true
  } catch (err) {
    console.warn(`⚠️ [LOCATION] Network error: ${err}`)
    return false
  }
}

/**
 * Get the user's current location from the backend.
 * GET /v1/location?user_id=xxx
 */
export async function getLocation(
  userID: string = DEFAULT_USER_ID,
): Promise<LocationUpdate | null> {
  log("FETCH_LOCATION", { user_id: userID })

  try {
    const res = await fetch(`${API_BASE}/v1/location?user_id=${userID}`)
    if (!res.ok) return null

    const data: LocationUpdate = await res.json()
    log("FETCH_SUCCESS", { latitude: data.latitude, longitude: data.longitude })
    return data
  } catch (err) {
    console.warn(`⚠️ [LOCATION] Network error fetching location: ${err}`)
    return null
  }
}

export async function listPosts(
  lat: number,
  lng: number,
  radius: number = 5000,
  category?: string,
): Promise<MapPost[]> {
  const params = new URLSearchParams({
    lat: lat.toString(),
    lng: lng.toString(),
    radius: radius.toString(),
  })
  if (category) params.set("category", category)

  try {
    const res = await fetch(`${API_BASE}/v1/posts?${params}`)
    if (!res.ok) return []
    return await res.json()
  } catch {
    return []
  }
}

// ─── Simulated Movement ─────────────────────────────────────────────────
// When real GPS is not available, we can simulate walking around the map.

const SF_STREETS = [
  // A loop around downtown San Francisco streets
  { lat: 37.7749, lng: -122.4194 },
  { lat: 37.7755, lng: -122.4180 },
  { lat: 37.7762, lng: -122.4165 },
  { lat: 37.7770, lng: -122.4150 },
  { lat: 37.7778, lng: -122.4135 },
  { lat: 37.7785, lng: -122.4120 },
  { lat: 37.7790, lng: -122.4105 },
  { lat: 37.7792, lng: -122.4088 },
  { lat: 37.7790, lng: -122.4070 },
  { lat: 37.7785, lng: -122.4055 },
  { lat: 37.7778, lng: -122.4040 },
  { lat: 37.7770, lng: -122.4025 },
  { lat: 37.7762, lng: -122.4010 },
  { lat: 37.7755, lng: -122.4000 },
  { lat: 37.7745, lng: -122.3995 },
  { lat: 37.7735, lng: -122.4000 },
  { lat: 37.7725, lng: -122.4010 },
  { lat: 37.7718, lng: -122.4025 },
  { lat: 37.7710, lng: -122.4040 },
  { lat: 37.7705, lng: -122.4055 },
  { lat: 37.7700, lng: -122.4070 },
  { lat: 37.7698, lng: -122.4088 },
  { lat: 37.7700, lng: -122.4105 },
  { lat: 37.7705, lng: -122.4120 },
  { lat: 37.7712, lng: -122.4135 },
  { lat: 37.7720, lng: -122.4150 },
  { lat: 37.7728, lng: -122.4165 },
  { lat: 37.7738, lng: -122.4180 },
]

/**
 * Start a simulated walk around the city.
 * Returns a cleanup function to stop the simulation.
 */
export function startSimulatedWalk(
  onStep: (lat: number, lng: number) => void,
  intervalMs: number = 2000,
): () => void {
  let index = 0
  let running = true

  function step() {
    if (!running) return
    const point = SF_STREETS[index]
    onStep(point.lat, point.lng)
    index = (index + 1) % SF_STREETS.length
    const jitter = Math.random() * 300
    setTimeout(step, intervalMs + jitter)
  }

  step()
  return () => { running = false }
}

// ─── Simulated Other Users ──────────────────────────────────────────────
// These are virtual users that move around the map to demonstrate
// real-time multi-user tracking via WebSocket.
// Each has their own route, pace, and marker color.

const OTHER_USERS_ROUTES = [
  {
    user_id: "user-sim-alice",
    name: "Alice",
    color: "#f59e0b" as const, // amber
    startIndex: 0,
    route: [
      { lat: 37.7810, lng: -122.4110 },
      { lat: 37.7820, lng: -122.4090 },
      { lat: 37.7835, lng: -122.4070 },
      { lat: 37.7840, lng: -122.4045 },
      { lat: 37.7838, lng: -122.4015 },
      { lat: 37.7825, lng: -122.4000 },
      { lat: 37.7810, lng: -122.4010 },
      { lat: 37.7800, lng: -122.4035 },
      { lat: 37.7795, lng: -122.4060 },
      { lat: 37.7800, lng: -122.4085 },
      { lat: 37.7810, lng: -122.4110 },
    ],
  },
  {
    user_id: "user-sim-bob",
    name: "Bob",
    color: "#10b981" as const, // emerald
    startIndex: 3,
    route: [
      { lat: 37.7680, lng: -122.4150 },
      { lat: 37.7670, lng: -122.4130 },
      { lat: 37.7665, lng: -122.4105 },
      { lat: 37.7670, lng: -122.4080 },
      { lat: 37.7680, lng: -122.4060 },
      { lat: 37.7695, lng: -122.4045 },
      { lat: 37.7710, lng: -122.4055 },
      { lat: 37.7715, lng: -122.4080 },
      { lat: 37.7710, lng: -122.4105 },
      { lat: 37.7695, lng: -122.4130 },
      { lat: 37.7680, lng: -122.4150 },
    ],
  },
  {
    user_id: "user-sim-carol",
    name: "Carol",
    color: "#8b5cf6" as const, // violet
    startIndex: 5,
    route: [
      { lat: 37.7730, lng: -122.4250 },
      { lat: 37.7745, lng: -122.4240 },
      { lat: 37.7760, lng: -122.4225 },
      { lat: 37.7775, lng: -122.4210 },
      { lat: 37.7780, lng: -122.4190 },
      { lat: 37.7775, lng: -122.4170 },
      { lat: 37.7760, lng: -122.4155 },
      { lat: 37.7745, lng: -122.4170 },
      { lat: 37.7735, lng: -122.4190 },
      { lat: 37.7730, lng: -122.4215 },
      { lat: 37.7730, lng: -122.4250 },
    ],
  },
]

/**
 * Start simulated movement for all "other users".
 * Each walks their own route at their own pace.
 * Returns a cleanup function to stop all.
 */
export function startOtherUsers(
  onUserStep: (userID: string, lat: number, lng: number) => void,
  intervalMs: number = 3500,
): () => void {
  const runners: { stop: () => void }[] = []

  for (const profile of OTHER_USERS_ROUTES) {
    let index = profile.startIndex
    let running = true

    function step() {
      if (!running) return
      const point = profile.route[index]
      onUserStep(profile.user_id, point.lat, point.lng)
      index = (index + 1) % profile.route.length
      // Each user moves at a slightly different speed
      const variation = 500 - index * 50
      setTimeout(step, intervalMs + variation)
    }

    step()
    runners.push({
      stop: () => {
        running = false
      },
    })
  }

  return () => {
    runners.forEach((r) => r.stop())
  }
}

/** Returns the initial positions of all simulated other users */
export function getOtherUsersInitial(): SimulatedUser[] {
  return OTHER_USERS_ROUTES.map((p) => ({
    user_id: p.user_id,
    name: p.name,
    color: p.color,
    lat: p.route[p.startIndex].lat,
    lng: p.route[p.startIndex].lng,
  }))
}

// ─── Map Posts ────────────────────────────────────────────────────────────
// Categories with their display colors

export const POST_CATEGORIES: Record<PostCategory, { label: string; color: string; icon: string }> = {
  RESTAURANT: { label: "Restaurant", color: "#e11d48", icon: "🍽️" },
  SOCIAL_LIFE: { label: "Social Life", color: "#f59e0b", icon: "🎉" },
  EVENT: { label: "Event", color: "#8b5cf6", icon: "🎪" },
} as const

/**
 * Fetch map posts near a location.
 * GET /v1/posts?lat=...&lng=...&radius=...
 */
/**
 * Create a new map post.
 * POST /v1/posts → Location Service (via API Gateway)
 */
export async function createPost(post: {
  title: string
  content?: string
  category: "RESTAURANT" | "SOCIAL_LIFE" | "EVENT"
  image_urls?: string[]
  latitude: number
  longitude: number
}): Promise<MapPost | null> {
  const body = {
    user_id: "user-sim-001",
    title: post.title,
    content: post.content,
    category: post.category,
    image_urls: post.image_urls ?? [],
    latitude: post.latitude,
    longitude: post.longitude,
  }

  log("CREATE_POST", { title: post.title, category: post.category })

  try {
    const res = await fetch(`${API_BASE}/v1/posts`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    })

    if (res.ok) {
      const data: MapPost = await res.json()
      log("CREATE_POST_SUCCESS", { id: data.id })
      return data
    }

    // If backend fails, create a client-side post so the UI still works
    console.warn(`⚠️ [POSTS] POST /v1/posts failed: ${res.status} — creating client-side post`)
    return createLocalPost(post)
  } catch (err) {
    console.warn(`⚠️ [POSTS] Network error creating post: ${err} — creating client-side post`)
    return createLocalPost(post)
  }
}

/** Creates a post on the client side when the backend is unavailable */
function createLocalPost(post: {
  title: string
  content?: string
  category: "RESTAURANT" | "SOCIAL_LIFE" | "EVENT"
  image_urls?: string[]
  latitude: number
  longitude: number
}): MapPost {
  const now = new Date().toISOString()
  return {
    id: `client-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    user_id: "user-sim-001",
    title: post.title,
    content: post.content,
    category: post.category,
    image_urls: post.image_urls ?? [],
    latitude: post.latitude,
    longitude: post.longitude,
    created_at: now,
    updated_at: now,
  }
}

/**
 * Fetch map posts near a location.
 * GET /v1/posts?lat=...&lng=...&radius=...
 */
export async function fetchPosts(
  lat: number,
  lng: number,
  radius: number = 3000,
): Promise<MapPost[]> {
  const params = new URLSearchParams({
    lat: lat.toString(),
    lng: lng.toString(),
    radius: radius.toString(),
  })

  try {
    const res = await fetch(`${API_BASE}/v1/posts?${params}`)
    if (!res.ok) return []
    return await res.json()
  } catch {
    return []
  }
}

// ─── Seed Map Posts (for demo when backend has no data) ────────────────
// These posts appear around downtown SF so the map always has pins to show.

export const SEED_POSTS: MapPost[] = [
  // ── RESTAURANTS ──────────────────────────────────────────────────────
  {
    id: "seed-rest-01",
    user_id: "user-sys",
    title: "Blue Bottle Coffee",
    content: "Amazing pour-over coffee and pastries in a minimalist space.",
    category: "RESTAURANT",
    image_urls: [
      "https://images.unsplash.com/photo-1495474472287-4d71bcdd2085?w=400&h=400&fit=crop",
      "https://images.unsplash.com/photo-1509042239860-f550ce710b93?w=400&h=400&fit=crop",
    ],
    latitude: 37.7765,
    longitude: -122.4180,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: "seed-rest-02",
    user_id: "user-sys",
    title: "State Bird Provisions",
    content: "Michelin-starred American dim-sum style dining. Must try the fried quail!",
    category: "RESTAURANT",
    image_urls: [
      "https://images.unsplash.com/photo-1414235077428-338989a2e8c0?w=400&h=400&fit=crop",
      "https://images.unsplash.com/photo-1546069901-ba9599a7e63c?w=400&h=400&fit=crop",
    ],
    latitude: 37.7800,
    longitude: -122.4090,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: "seed-rest-03",
    user_id: "user-sys",
    title: "Tartine Bakery",
    content: "Legendary sourdough bread, croissants, and morning buns.",
    category: "RESTAURANT",
    image_urls: [
      "https://images.unsplash.com/photo-1555507036-ab1f4038028a?w=400&h=400&fit=crop",
    ],
    latitude: 37.7710,
    longitude: -122.4220,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: "seed-rest-04",
    user_id: "user-sys",
    title: "Tony's Pizza Napoletana",
    content: "Award-winning Neapolitan pizza with a coal-fired crust.",
    category: "RESTAURANT",
    image_urls: [
      "https://images.unsplash.com/photo-1565299624946-b28f40a0ae38?w=400&h=400&fit=crop",
    ],
    latitude: 37.7715,
    longitude: -122.4090,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: "seed-rest-05",
    user_id: "user-sys",
    title: "Burma Superstar",
    content: "Incredible Burmese tea leaf salad and coconut noodles. Always a line!",
    category: "RESTAURANT",
    image_urls: [
      "https://images.unsplash.com/photo-1455619452474-d2be8b1e70cd?w=400&h=400&fit=crop",
    ],
    latitude: 37.7738,
    longitude: -122.4140,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: "seed-rest-06",
    user_id: "user-sys",
    title: "Swan Oyster Depot",
    content: "Old-school seafood counter. The crab louie and clam chowder are legendary.",
    category: "RESTAURANT",
    image_urls: [
      "https://images.unsplash.com/photo-1559742811-822f4580b12e?w=400&h=400&fit=crop",
    ],
    latitude: 37.7740,
    longitude: -122.4100,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: "seed-rest-07",
    user_id: "user-sys",
    title: "Zuni Cafe",
    content: "Famous roast chicken with bread salad. Cozy brick-walled atmosphere.",
    category: "RESTAURANT",
    image_urls: [],
    latitude: 37.7820,
    longitude: -122.4150,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },

  // ── SOCIAL LIFE ─────────────────────────────────────────────────────
  {
    id: "seed-social-01",
    user_id: "user-sys",
    title: "Dolores Park",
    content: "Sunny spot for picnics, people-watching, and amazing city views.",
    category: "SOCIAL_LIFE",
    image_urls: [
      "https://images.unsplash.com/photo-1534008897995-27a23e859048?w=400&h=400&fit=crop",
      "https://images.unsplash.com/photo-1582213782179-e0d53f98f2ca?w=400&h=400&fit=crop",
    ],
    latitude: 37.7690,
    longitude: -122.4170,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: "seed-social-02",
    user_id: "user-sys",
    title: "Ferry Building Marketplace",
    content: "Weekend farmers market with local produce, artisan cheeses, and fresh seafood.",
    category: "SOCIAL_LIFE",
    image_urls: [
      "https://images.unsplash.com/photo-1488459716781-31db52582fe9?w=400&h=400&fit=crop",
    ],
    latitude: 37.7785,
    longitude: -122.4130,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: "seed-social-03",
    user_id: "user-sys",
    title: "Alamo Square Park",
    content: "Famous Painted Ladies view. Perfect spot for sunset photos 📸",
    category: "SOCIAL_LIFE",
    image_urls: [
      "https://images.unsplash.com/photo-1580131732415-63c3c0360b78?w=400&h=400&fit=crop",
    ],
    latitude: 37.7764,
    longitude: -122.4340,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: "seed-social-04",
    user_id: "user-sys",
    title: "Mission District Mural Walk",
    content: "Self-guided tour of incredible street art on Clarion Alley and Balmy Alley.",
    category: "SOCIAL_LIFE",
    image_urls: [
      "https://images.unsplash.com/photo-1578301978693-85fa9c0320b9?w=400&h=400&fit=crop",
    ],
    latitude: 37.7605,
    longitude: -122.4145,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: "seed-social-05",
    user_id: "user-sys",
    title: "Salesforce Park Rooftop",
    content: "Stunning elevated park with native plants, yoga classes, and city panoramas.",
    category: "SOCIAL_LIFE",
    image_urls: [
      "https://images.unsplash.com/photo-1574362848149-11496d93a7c7?w=400&h=400&fit=crop",
    ],
    latitude: 37.7794,
    longitude: -122.4160,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },

  // ── EVENTS ──────────────────────────────────────────────────────────
  {
    id: "seed-event-01",
    user_id: "user-sys",
    title: "SF Jazz Center",
    content: "World-class jazz performances in an intimate venue. Check tonight's show!",
    category: "EVENT",
    image_urls: [
      "https://images.unsplash.com/photo-1511192336575-5a79af67a629?w=400&h=400&fit=crop",
    ],
    latitude: 37.7750,
    longitude: -122.4210,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: "seed-event-02",
    user_id: "user-sys",
    title: "Oracle Park Night Game",
    content: "Evening Giants game with fireworks. Gates open at 6 PM.",
    category: "EVENT",
    image_urls: [
      "https://images.unsplash.com/photo-1472457897821-70d3819a0e24?w=400&h=400&fit=crop",
      "https://images.unsplash.com/photo-1566577739112-5180d4bf9391?w=400&h=400&fit=crop",
    ],
    latitude: 37.7790,
    longitude: -122.4070,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: "seed-event-03",
    user_id: "user-sys",
    title: "SFMOMA After Hours",
    content: "Late-night museum access with DJ sets, cocktails, and contemporary art.",
    category: "EVENT",
    image_urls: [
      "https://images.unsplash.com/photo-1531913764164-f85c9e7e5c7f?w=400&h=400&fit=crop",
    ],
    latitude: 37.7732,
    longitude: -122.4018,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: "seed-event-04",
    user_id: "user-sys",
    title: "Hardly Strictly Bluegrass",
    content: "Free 3-day music festival in Golden Gate Park. Bluegrass, folk & more.",
    category: "EVENT",
    image_urls: [
      "https://images.unsplash.com/photo-1506157786151-b8491531f063?w=400&h=400&fit=crop",
    ],
    latitude: 37.7685,
    longitude: -122.4520,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: "seed-event-05",
    user_id: "user-sys",
    title: "Castro Theatre Film Night",
    content: "Classic film screening at the historic Castro Theatre. Tonight: Casablanca.",
    category: "EVENT",
    image_urls: [
      "https://images.unsplash.com/photo-1489599849927-2ee91cede3ba?w=400&h=400&fit=crop",
    ],
    latitude: 37.7620,
    longitude: -122.4350,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: "seed-event-06",
    user_id: "user-sys",
    title: "Off the Grid: Fort Mason",
    content: "Weekly food truck gathering with live music overlooking the bay.",
    category: "EVENT",
    image_urls: [
      "https://images.unsplash.com/photo-1565123409695-7b5ef63a2efb?w=400&h=400&fit=crop",
    ],
    latitude: 37.7812,
    longitude: -122.4240,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
]

/**
 * Combines API posts with seed posts (deduped by id prefix).
 * If the API returns results, those take priority.
 */
export function mergePosts(apiPosts: MapPost[]): MapPost[] {
  const apiIds = new Set(apiPosts.map((p) => p.id))
  const seeds = SEED_POSTS.filter((s) => !apiIds.has(s.id))
  return [...apiPosts, ...seeds]
}
