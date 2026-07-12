"use client"

import dynamic from "next/dynamic"
import { useCallback, useEffect, useRef, useState } from "react"
import Map, {
  Marker,
  NavigationControl,
  ScaleControl,
  MapRef,
  Popup,
} from "react-map-gl/maplibre"
import "maplibre-gl/dist/maplibre-gl.css"
import type { MapLayerMouseEvent, MarkerDragEvent } from "react-map-gl/maplibre"
import { motion } from "framer-motion"
import { MAP_STYLE } from "../constants"
import {
  updateLocation,
  getLocation,
  startSimulatedWalk,
  startOtherUsers,
  getOtherUsersInitial,
  fetchPosts,
  mergePosts,
  SEED_POSTS,
  POST_CATEGORIES,
} from "../lib/location"
import type { LocationLogEntry, SimulatedUser, MapPost } from "../types/location"
import { useLocationWebSocket } from "../lib/useWebSocket"
import type { WSLocationMessage } from "../types/websocket"
import { useMapStore } from "../lib/useMapStore"
import {
  OtherUserMarkerSVG,
  PostPinSVG,
  HapticUserMarkerSVG,
} from "./markers"
import ThreadDrawer from "./ThreadDrawer"
import CreatePostDrawer from "./CreatePostDrawer"



// ─── Constants ──────────────────────────────────────────────────────────

const INITIAL_VIEW = {
  latitude: 37.7749,
  longitude: -122.4194,
  zoom: 13,
}

const SIM_INTERVAL_MS = 2500
const OTHER_USERS_INTERVAL_MS = 3200
const MAX_LOG_ENTRIES = 30
const USER_ID = "user-sim-001"

// ─── User info map (for display) ────────────────────────────────────────

const USER_INFO: Record<string, { name: string; color: string }> = {
  "user-sim-001": { name: "You", color: "#3b82f6" }, // blue
  "user-sim-alice": { name: "Alice", color: "#f59e0b" }, // amber
  "user-sim-bob": { name: "Bob", color: "#10b981" }, // emerald
  "user-sim-carol": { name: "Carol", color: "#8b5cf6" }, // violet
}

// ─── Component ──────────────────────────────────────────────────────────

export default function MapView() {
  const mapRef = useRef<MapRef>(null)

  // ─── Local user state ───────────────────────────────────────────────
  const [position, setPosition] = useState({
    latitude: INITIAL_VIEW.latitude,
    longitude: INITIAL_VIEW.longitude,
  })
  const [showPopup, setShowPopup] = useState(false)
  const [isDragging, setIsDragging] = useState(false)

  // ─── Other users (simulated + from WebSocket) ──────────────────────
  const [otherUsers, setOtherUsers] = useState<SimulatedUser[]>(() =>
    getOtherUsersInitial(),
  )
  const [selectedUser, setSelectedUser] = useState<string | null>(null)

  // ─── Haptic click state ───────────────────────────────────────────
  const [clickedPinId, setClickedPinId] = useState<string | null>(null)
  const hapticTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  function triggerHaptic(id: string) {
    if (hapticTimer.current) clearTimeout(hapticTimer.current)
    setClickedPinId(id)
    hapticTimer.current = setTimeout(() => {
      setClickedPinId(null)
    }, 500)
  }

  // ─── Map posts ───────────────────────────────────────────────────────
  const [mapPosts, setMapPosts] = useState<MapPost[]>([])
  const openPost = useMapStore((s) => s.openPost)
  const openCreatePost = useMapStore((s) => s.openCreatePost)

  // ─── Simulation ──────────────────────────────────────────────────────
  const [simulating, setSimulating] = useState(false)
  const simCleanup = useRef<(() => void) | null>(null)
  const [otherUsersRunning, setOtherUsersRunning] = useState(false)
  const otherUsersCleanup = useRef<(() => void) | null>(null)

  // ─── Connection status ───────────────────────────────────────────────
  const [connected, setConnected] = useState(false)
  const [checking, setChecking] = useState(true)

  // ─── Location log ────────────────────────────────────────────────────
  const [log, setLog] = useState<LocationLogEntry[]>([])
  const logEndRef = useRef<HTMLDivElement>(null)

  // Auto-scroll log
  useEffect(() => {
    logEndRef.current?.scrollIntoView({ behavior: "smooth" })
  }, [log])

  // ─── WebSocket ───────────────────────────────────────────────────────
  const handleWSMessage = useCallback((msg: WSLocationMessage) => {
    addLog(msg.latitude, msg.longitude, "websocket")

    if (msg.user_id !== USER_ID) {
      setOtherUsers((prev) => {
        const exists = prev.find((u) => u.user_id === msg.user_id)
        if (exists) {
          return prev.map((u) =>
            u.user_id === msg.user_id
              ? { ...u, lat: msg.latitude, lng: msg.longitude }
              : u,
          )
        }
        const info = USER_INFO[msg.user_id]
        return [
          ...prev,
          {
            user_id: msg.user_id,
            name: info?.name ?? msg.user_id,
            color: (info?.color ?? "#94a3b8") as SimulatedUser["color"],
            lat: msg.latitude,
            lng: msg.longitude,
          },
        ]
      })
    }
  }, [])

  const ws = useLocationWebSocket(USER_ID, handleWSMessage)

  // ─── Load map posts near current position ──────────────────────────
  const loadPosts = useCallback(async (lat: number, lng: number) => {
    const apiPosts = await fetchPosts(lat, lng, 3000)
    if (apiPosts.length > 0) {
      setMapPosts(apiPosts)
      console.log(`📍 [POSTS] Fetched ${apiPosts.length} posts from API near (${lat.toFixed(4)}, ${lng.toFixed(4)})`)
    } else {
      setMapPosts(mergePosts([]))
      console.log(`📍 [POSTS] Using ${SEED_POSTS.length} seed posts for demo`)
    }
  }, [])

  // ─── Check connection to location service on mount ──────────────────
  useEffect(() => {
    let mounted = true
    async function checkConnection() {
      const pos = await getLocation(USER_ID)
      if (!mounted) return
      if (pos) {
        setConnected(true)
        if (pos.latitude && pos.longitude) {
          setPosition({ latitude: pos.latitude, longitude: pos.longitude })
          addLog(pos.latitude, pos.longitude, "api")
        }
      } else {
        setConnected(false)
      }
      setChecking(false)
    }
    checkConnection()
    return () => {
      mounted = false
    }
  }, [])

  // ─── Load posts on mount ───────────────────────────────────────────
  useEffect(() => {
    loadPosts(position.latitude, position.longitude)
  }, [])

  // ─── Reload posts when user moves (debounced) ───────────────────────
  const postLoadTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  useEffect(() => {
    if (postLoadTimer.current) clearTimeout(postLoadTimer.current)
    postLoadTimer.current = setTimeout(() => {
      loadPosts(position.latitude, position.longitude)
    }, 3000)
    return () => {
      if (postLoadTimer.current) clearTimeout(postLoadTimer.current)
    }
  }, [position.latitude, position.longitude, loadPosts])

  // ─── Add log entry ──────────────────────────────────────────────────
  function addLog(lat: number, lng: number, source: LocationLogEntry["source"]) {
    const entry: LocationLogEntry = {
      timestamp: new Date().toISOString(),
      latitude: lat,
      longitude: lng,
      source,
    }
    setLog((prev) => [entry, ...prev].slice(0, MAX_LOG_ENTRIES))
  }

  // ─── Send location to backend ───────────────────────────────────────
  const sendLocation = useCallback(
    async (lat: number, lng: number, source: LocationLogEntry["source"]) => {
      setPosition({ latitude: lat, longitude: lng })
      addLog(lat, lng, source)

      console.log(
        `📍 [LOCATION] ${source.toUpperCase()} → (${lat.toFixed(6)}, ${lng.toFixed(6)})`,
      )

      const ok = await updateLocation(lat, lng, USER_ID)
      setConnected(ok || connected)
    },
    [connected],
  )

  // ─── Map click handler ──────────────────────────────────────────────
  const handleMapClick = useCallback(
    (e: MapLayerMouseEvent) => {
      sendLocation(e.lngLat.lat, e.lngLat.lng, "click")
      setShowPopup(true)
      setSelectedUser(null)
    },
    [sendLocation],
  )

  // ─── Marker drag handler ────────────────────────────────────────────
  const handleMarkerDrag = useCallback((e: MarkerDragEvent) => {
    const { lat, lng } = e.lngLat
    setPosition({ latitude: lat, longitude: lng })
    setIsDragging(true)
  }, [])

  const handleMarkerDragEnd = useCallback(
    (e: MarkerDragEvent) => {
      setIsDragging(false)
      setSelectedUser(null)
      sendLocation(e.lngLat.lat, e.lngLat.lng, "drag")
    },
    [sendLocation],
  )

  // ─── Toggle your own walk simulation ────────────────────────────────
  const toggleSimulation = useCallback(() => {
    if (simulating) {
      simCleanup.current?.()
      simCleanup.current = null
      setSimulating(false)
      console.log("📍 [LOCATION] Your walk STOPPED")
    } else {
      setSimulating(true)
      console.log("📍 [LOCATION] Your walk STARTED — walking around SF")
      const cleanup = startSimulatedWalk(
        (lat, lng) => sendLocation(lat, lng, "simulation"),
        SIM_INTERVAL_MS,
      )
      simCleanup.current = cleanup
    }
  }, [simulating, sendLocation])

  // ─── Toggle other users simulation ──────────────────────────────────
  const toggleOtherUsers = useCallback(() => {
    if (otherUsersRunning) {
      otherUsersCleanup.current?.()
      otherUsersCleanup.current = null
      setOtherUsersRunning(false)
      console.log("📍 [LOCATION] Other users simulation STOPPED")
    } else {
      setOtherUsers(getOtherUsersInitial())
      setOtherUsersRunning(true)
      console.log(
        "📍 [LOCATION] Other users simulation STARTED — Alice, Bob, Carol walking",
      )
      const cleanup = startOtherUsers(
        (userID, lat, lng) => {
          setOtherUsers((prev) =>
            prev.map((u) =>
              u.user_id === userID ? { ...u, lat, lng } : u,
            ),
          )
          updateLocation(lat, lng, userID)
          console.log(
            `👤 [OTHER] ${USER_INFO[userID]?.name ?? userID} → ` +
              `(${lat.toFixed(6)}, ${lng.toFixed(6)})`,
          )
        },
        OTHER_USERS_INTERVAL_MS,
      )
      otherUsersCleanup.current = cleanup
    }
  }, [otherUsersRunning])

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      simCleanup.current?.()
      otherUsersCleanup.current?.()
      if (hapticTimer.current) clearTimeout(hapticTimer.current)
    }
  }, [])

  // ─── Format timestamp ───────────────────────────────────────────────
  function formatTime(iso: string) {
    return new Date(iso).toLocaleTimeString("en-US", {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    })
  }

  // ─── Get source display info ────────────────────────────────────────
  function sourceBadge(source: string) {
    const colors: Record<string, string> = {
      simulation: "bg-blue-500/10 text-blue-400 border-blue-500/20",
      drag: "bg-amber-500/10 text-amber-400 border-amber-500/20",
      click: "bg-emerald-500/10 text-emerald-400 border-emerald-500/20",
      api: "bg-slate-500/10 text-slate-400 border-slate-500/20",
      websocket: "bg-purple-500/10 text-purple-400 border-purple-500/20",
    }
    return colors[source] ?? "bg-slate-500/10 text-slate-400 border-slate-500/20"
  }

  // ─── Store values for 3D map transform ──────────────────────────────
  const mapScale = useMapStore((s) => s.mapScale)
  const mapRotateX = useMapStore((s) => s.mapRotateX)
  const mapBlur = useMapStore((s) => s.mapBlur)

  // ─── Loading state ──────────────────────────────────────────────────
  if (checking) {
    return (
      <div className="h-full w-full flex items-center justify-center bg-slate-900 text-white">
        <div className="text-center space-y-4">
          <div className="animate-spin rounded-full h-10 w-10 border-t-2 border-b-2 border-blue-400 mx-auto" />
          <p className="text-slate-400 text-sm">Connecting to Location Service...</p>
        </div>
      </div>
    )
  }

  // ─── Render ─────────────────────────────────────────────────────────
  return (
    <div className="h-full w-full flex overflow-hidden">
      {/* ─── Main Content Area (map + log panel) ─────────────────── */}
      <div className="flex-1 flex min-w-0">
        {/* ─── Main Map with 3D Transform ──────────────────────────────── */}
        <motion.div
          className="relative flex-1 origin-center"
          animate={{
            scale: mapScale,
            rotateX: mapRotateX,
            filter: `blur(${mapBlur}px)`,
          }}
          transition={{ duration: 0.5, ease: [0.32, 0.72, 0, 1] }}
          style={{ perspective: 1200 }}
        >
          <Map
            ref={mapRef}
            mapStyle={MAP_STYLE}
            initialViewState={INITIAL_VIEW}
            style={{ height: "100%", width: "100%" }}
            onClick={handleMapClick}
            cursor={isDragging ? "grabbing" : "crosshair"}
          >
            <NavigationControl position="top-right" />
            <ScaleControl position="bottom-left" unit="metric" />

            {/* ── Other users' markers (non-draggable) ── */}
            {otherUsers.map((u) => (
              <Marker
                key={u.user_id}
                longitude={u.lng}
                latitude={u.lat}
                anchor="bottom"
                onClick={(e) => {
                  e.originalEvent.stopPropagation()
                  setSelectedUser(u.user_id)
                  setShowPopup(false)
                }}
              >
                <OtherUserMarkerSVG color={u.color} />
              </Marker>
            ))}

            {/* ── Other user popup ── */}
            {selectedUser &&
              (() => {
                const u = otherUsers.find((o) => o.user_id === selectedUser)
                if (!u) return null
                return (
                  <Popup
                    longitude={u.lng}
                    latitude={u.lat}
                    closeButton={true}
                    closeOnClick={false}
                    onClose={() => setSelectedUser(null)}
                    anchor="top"
                  >
                    <div className="text-xs space-y-1 min-w-[140px]">
                      <p
                        className="font-semibold flex items-center gap-1.5"
                        style={{ color: u.color }}
                      >
                        <span
                          className="w-2 h-2 rounded-full inline-block"
                          style={{ backgroundColor: u.color }}
                        />
                        {u.name}
                      </p>
                      <p className="font-mono text-gray-700">
                        {u.lat.toFixed(6)}, {u.lng.toFixed(6)}
                      </p>
                    </div>
                  </Popup>
                )
              })()}

            {/* ── Map Post markers → open drawer ── */}
            {mapPosts.map((post) => (
              <Marker
                key={post.id}
                longitude={post.longitude}
                latitude={post.latitude}
                anchor="bottom"
                onClick={(e) => {
                  e.originalEvent.stopPropagation()
                  triggerHaptic(post.id)
                  openPost(post)
                  setSelectedUser(null)
                  setShowPopup(false)
                }}
              >
                <PostPinSVG
                  color={POST_CATEGORIES[post.category]?.color ?? "#94a3b8"}
                  category={post.category}
                  isClicked={clickedPinId === post.id}
                />
              </Marker>
            ))}

            {/* ── Local user marker (draggable) ── */}
            <Marker
              longitude={position.longitude}
              latitude={position.latitude}
              anchor="bottom"
              draggable={!simulating}
              onDrag={handleMarkerDrag}
              onDragEnd={handleMarkerDragEnd}
              onClick={() => {
                triggerHaptic("user-local")
                setSelectedUser(null)
                setShowPopup((p) => !p)
              }}
            >
              <HapticUserMarkerSVG
                color="#3b82f6"
                pulse={simulating || isDragging}
                isClicked={clickedPinId === "user-local"}
              />
            </Marker>

            {/* ── Local user popup ── */}
            {showPopup && (
              <Popup
                longitude={position.longitude}
                latitude={position.latitude}
                closeButton={true}
                closeOnClick={false}
                onClose={() => setShowPopup(false)}
                anchor="top"
              >
                <div className="text-xs space-y-1 min-w-[170px]">
                  <p className="font-semibold text-blue-600 flex items-center gap-1.5">
                    <span className="w-2 h-2 rounded-full bg-blue-500 inline-block" />
                    You
                  </p>
                  <p className="font-mono text-gray-700">
                    {position.latitude.toFixed(6)}, {position.longitude.toFixed(6)}
                  </p>
                  <p className="text-gray-400 flex items-center gap-1">
                    <span className={`w-1.5 h-1.5 rounded-full ${connected ? "bg-green-400" : "bg-amber-400"}`} />
                    {connected ? "Live" : "Offline"}
                  </p>
                </div>
              </Popup>
            )}
          </Map>

          {/* ─── Top-left Status Panel ────────────────────────────────── */}
          <div className="absolute top-4 left-4 z-10 space-y-2">
            <div
              className={`px-3 py-1.5 rounded-full text-xs font-semibold shadow-lg backdrop-blur-sm border ${
                connected
                  ? "bg-green-500/20 text-green-300 border-green-500/30"
                  : "bg-amber-500/20 text-amber-300 border-amber-500/30"
              }`}
            >
              {connected ? "● REST Connected" : "○ Offline (cached)"}
            </div>

            <div
              className={`px-3 py-1.5 rounded-full text-xs font-semibold shadow-lg backdrop-blur-sm border flex items-center gap-1.5 ${
                ws.status === "connected"
                  ? "bg-purple-500/20 text-purple-300 border-purple-500/30"
                  : ws.status === "connecting"
                    ? "bg-blue-500/20 text-blue-300 border-blue-500/30"
                    : "bg-slate-500/20 text-slate-400 border-slate-500/30"
              }`}
            >
              {ws.status === "connected" ? (
                <>
                  <span className="w-1.5 h-1.5 rounded-full bg-purple-400 animate-pulse" />
                  WS Connected
                  <span className="text-[10px] opacity-60">({ws.messageCount})</span>
                </>
              ) : ws.status === "connecting" ? (
                <>
                  <span className="w-1.5 h-1.5 rounded-full bg-blue-400 animate-pulse" />
                  WS Connecting...
                </>
              ) : (
                <>
                  <span className="w-1.5 h-1.5 rounded-full bg-slate-500" />
                  WS Disconnected
                </>
              )}
            </div>

            <div className="px-3 py-1.5 rounded-full text-xs font-mono shadow-lg backdrop-blur-sm bg-slate-900/70 text-slate-200 border border-slate-700/50">
              {position.latitude.toFixed(6)}, {position.longitude.toFixed(6)}
            </div>
          </div>

          {/* ─── User & Posts Legend (top-right) ──────────────────────── */}
          <div className="absolute top-4 right-4 z-10 px-3 py-2 rounded-xl shadow-lg backdrop-blur-sm bg-slate-900/80 border border-slate-700/50 space-y-2">
            <div>
              <p className="text-[10px] uppercase tracking-wider text-slate-500 mb-1.5 font-semibold">
                Users
              </p>
              <div className="space-y-1">
                <div className="flex items-center gap-2 text-xs">
                  <span className="w-2.5 h-2.5 rounded-full bg-blue-500 inline-block" />
                  <span className="text-slate-300">You</span>
                </div>
                {otherUsers.map((u) => (
                  <div key={u.user_id} className="flex items-center gap-2 text-xs">
                    <span
                      className="w-2.5 h-2.5 rounded-full inline-block"
                      style={{ backgroundColor: u.color }}
                    />
                    <span className="text-slate-300">{u.name}</span>
                    <span className="text-[10px] text-slate-500">
                      {u.lat.toFixed(4)}, {u.lng.toFixed(4)}
                    </span>
                  </div>
                ))}
              </div>
            </div>

            {mapPosts.length > 0 && (
              <>
                <div className="h-px bg-slate-700/50" />
                <div>
                  <div className="flex items-center justify-between mb-1.5">
                    <p className="text-[10px] uppercase tracking-wider text-slate-500 font-semibold">
                      Places
                    </p>
                    <span className="text-[10px] text-slate-600">{mapPosts.length}</span>
                  </div>
                  <div className="space-y-1">
                    {Object.entries(POST_CATEGORIES).map(([key, val]) => {
                      const count = mapPosts.filter((p) => p.category === key).length
                      if (count === 0) return null
                      return (
                        <div key={key} className="flex items-center gap-2 text-xs">
                          <span
                            className="w-2.5 h-2.5 rounded-full inline-block"
                            style={{ backgroundColor: val.color }}
                          />
                          <span className="text-slate-300">
                            {val.icon} {val.label}
                          </span>
                          <span className="text-[10px] text-slate-500 ml-auto">{count}</span>
                        </div>
                      )
                    })}
                  </div>
                </div>
              </>
            )}
          </div>

          {/* ─── Create Post FAB ──────────────────────────────────────── */}
          <div className="absolute bottom-6 right-6 z-10">
            <motion.button
              onClick={() => {
                openCreatePost(position.latitude, position.longitude)
                setShowPopup(false)
                setSelectedUser(null)
              }}
              className="w-14 h-14 rounded-full bg-blue-500 text-white shadow-2xl shadow-blue-500/30 flex items-center justify-center"
              whileHover={{ scale: 1.1, rotate: 90 }}
              whileTap={{ scale: 0.9 }}
              transition={{ type: "spring", stiffness: 400, damping: 17 }}
              title="Create Post"
            >
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                <path d="M12 5v14M5 12h14" />
              </svg>
            </motion.button>
          </div>

          {/* ─── Bottom Controls ───────────────────────────────────────── */}
          <div className="absolute bottom-6 left-1/2 -translate-x-1/2 z-10 flex items-center gap-2 px-4 py-3 rounded-2xl shadow-2xl backdrop-blur-md bg-slate-900/80 border border-slate-700/50">
            <button
              onClick={toggleSimulation}
              className={`flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-semibold transition-all duration-200 ${
                simulating
                  ? "bg-red-500/20 text-red-300 hover:bg-red-500/30 border border-red-500/30"
                  : "bg-blue-500/20 text-blue-300 hover:bg-blue-500/30 border border-blue-500/30"
              }`}
            >
              <span
                className={`w-2 h-2 rounded-full ${simulating ? "bg-red-400 animate-pulse" : "bg-blue-400"}`}
              />
              {simulating ? "Stop Walk" : "Walk (You)"}
            </button>

            <button
              onClick={toggleOtherUsers}
              className={`flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-semibold transition-all duration-200 ${
                otherUsersRunning
                  ? "bg-red-500/20 text-red-300 hover:bg-red-500/30 border border-red-500/30"
                  : "bg-emerald-500/20 text-emerald-300 hover:bg-emerald-500/30 border border-emerald-500/30"
              }`}
            >
              <span
                className={`w-2 h-2 rounded-full ${otherUsersRunning ? "bg-red-400 animate-pulse" : "bg-emerald-400"}`}
              />
              {otherUsersRunning ? "Stop Others" : "Simulate Others"}
            </button>

            <div className="w-px h-6 bg-slate-700/50" />

            <p className="text-xs text-slate-400 hidden sm:block">
              {simulating && otherUsersRunning
                ? "Everyone is walking around SF..."
                : simulating
                  ? "You are walking..."
                  : otherUsersRunning
                    ? "Others are walking..."
                    : "Click or drag to move"}
            </p>
          </div>
        </motion.div>

        {/* ─── Log Panel ──────────────────────────────────────────────── */}
        <div className="w-80 bg-slate-900 border-l border-slate-800 flex flex-col">
          {/* Header */}
          <div className="px-4 py-3 border-b border-slate-800 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className={`w-2 h-2 rounded-full ${ws.status === "connected" ? "bg-purple-400 animate-pulse" : "bg-slate-500"}`} />
              <h2 className="text-sm font-semibold text-slate-200">
                Location Log
                {ws.messageCount > 0 && (
                  <span className="ml-1.5 text-[10px] text-slate-500 font-normal">
                    ({ws.messageCount} WS)
                  </span>
                )}
              </h2>
            </div>
            <span className="text-xs text-slate-500">{log.length} entries</span>
          </div>

          {/* Connection info */}
          <div className="px-4 py-2 text-xs text-slate-500 border-b border-slate-800/50 flex flex-wrap items-center gap-x-2 gap-y-1">
            <span>User:</span>
            <code className="px-1.5 py-0.5 rounded bg-slate-800 text-slate-300 font-mono">
              {USER_ID}
            </code>
            <span className="text-slate-600">|</span>
            <span>WS:</span>
            <code className="text-slate-400">{ws.status}</code>
          </div>

          {/* Log entries */}
          <div className="flex-1 overflow-y-auto p-2 space-y-1">
            {log.length === 0 ? (
              <div className="flex items-center justify-center h-full text-slate-600 text-xs">
                <p>No location updates yet</p>
              </div>
            ) : (
              log.map((entry, i) => (
                <div
                  key={`${entry.timestamp}-${i}`}
                  className={`px-3 py-2 rounded-lg text-xs font-mono border ${sourceBadge(entry.source)}`}
                >
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-[10px] uppercase tracking-wider font-semibold">
                      {entry.source}
                    </span>
                    <span className="text-[10px] text-slate-500">
                      {formatTime(entry.timestamp)}
                    </span>
                  </div>
                  <p className="text-slate-300">
                    {entry.latitude.toFixed(6)}, {entry.longitude.toFixed(6)}
                  </p>
                </div>
              ))
            )}
            <div ref={logEndRef} />
          </div>

          {/* Footer with user legend */}
          <div className="px-4 py-2 border-t border-slate-800 text-[10px] text-slate-600">
            <div className="flex items-center justify-between mb-1">
              <span>Click · Drag · Walk · Others</span>
              {log.length > 0 && (
                <button
                  onClick={() => setLog([])}
                  className="text-slate-500 hover:text-slate-300 transition-colors"
                >
                  Clear
                </button>
              )}
            </div>
            <div className="flex items-center gap-3 text-slate-500">
              {[
                { color: "#3b82f6", label: "You" },
                { color: "#f59e0b", label: "Alice" },
                { color: "#10b981", label: "Bob" },
                { color: "#8b5cf6", label: "Carol" },
              ].map((u) => (
                <span key={u.label} className="flex items-center gap-1">
                  <span
                    className="w-1.5 h-1.5 rounded-full inline-block"
                    style={{ backgroundColor: u.color }}
                  />
                  {u.label}
                </span>
              ))}
            </div>
          </div>
        </div>
      </div>

      {/* ─── ThreadDrawer (bottom-sheet overlay, fixed position) ──── */}
      <ThreadDrawer />

      {/* ─── CreatePostDrawer (fixed position) ──── */}
      <CreatePostDrawer
        onPostCreated={(newPost) => {
          setMapPosts((prev) => [newPost, ...prev])
          // Open the new post immediately
          openPost(newPost)
        }}
      />
    </div>
  )
}
