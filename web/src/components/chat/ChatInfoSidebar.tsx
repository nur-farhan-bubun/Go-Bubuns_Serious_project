"use client"

import { useState } from "react"
import { useChatStore } from "./ChatStore"
import UserProfileView from "./UserProfileView"
import Image from "next/image"

// ─── SVG Icons ──────────────────────────────────────────────────────────

function PhoneIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z" />
    </svg>
  )
}

function VideoIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <polygon points="23 7 16 12 23 17 23 7" />
      <rect x="1" y="5" width="15" height="14" rx="2" ry="2" />
    </svg>
  )
}

function PinIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 2C7.6 2 4 5.6 4 10c0 5.5 7.3 11.5 7.6 11.7.2.2.6.2.8 0C12.7 21.5 20 15.5 20 10c0-4.4-3.6-8-8-8z" />
      <circle cx="12" cy="10" r="3" />
    </svg>
  )
}

function ChevronDown() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="m6 9 6 6 6-6" />
    </svg>
  )
}

function ChevronUp() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="m18 15-6-6-6 6" />
    </svg>
  )
}

// ─── Accordion Section ─────────────────────────────────────────────────

function AccordionSection({
  title,
  count,
  defaultOpen = true,
  children,
}: {
  title: string
  count?: number
  defaultOpen?: boolean
  children: React.ReactNode
}) {
  const [open, setOpen] = useState(defaultOpen)

  return (
    <div className="border-b border-chat-border pb-3">
      <button
        onClick={() => setOpen(!open)}
        className="w-full flex items-center justify-between py-2 text-xs font-semibold text-chat-muted uppercase tracking-wider"
      >
        <span className="flex items-center gap-1.5">
          {title}
          {count !== undefined && (
            <span className="text-[10px] font-normal text-chat-muted bg-chat-card px-1.5 py-0.5 rounded-md normal-case">
              {count}
            </span>
          )}
        </span>
        {open ? <ChevronUp /> : <ChevronDown />}
      </button>
      {open && <div className="space-y-1.5">{children}</div>}
    </div>
  )
}

// ─── Component ──────────────────────────────────────────────────────────

export default function ChatInfoSidebar() {
  const { activeConversationId, conversations, assets, users, selectedProfileUserId, setSelectedProfileUser } = useChatStore()

  const conv = conversations.find((c) => c.id === activeConversationId)

  const photos = assets.filter((a) => a.type === "photo")
  const files = assets.filter((a) => a.type === "file" || a.type === "link")

  // If a profile is selected, show the profile view instead of the sidebar
  if (selectedProfileUserId) {
    return (
      <div className="w-80 bg-chat-panel border-l border-chat-border shrink-0 h-full flex flex-col">
        <UserProfileView />
      </div>
    )
  }

  return (
    <div className="w-80 bg-chat-panel border-l border-chat-border shrink-0 h-full flex flex-col">
      {/* ── Media Controls ───────────────────────────────────────────── */}
      <div className="px-4 pt-4 pb-3 border-b border-chat-border shrink-0">
        <div className="grid grid-cols-3 gap-2">
          <button className="flex flex-col items-center gap-1.5 px-3 py-3 rounded-xl bg-chat-card hover:bg-chat-card/80 transition-colors">
            <div className="w-9 h-9 rounded-full bg-chat-accent/10 flex items-center justify-center text-chat-accent">
              <PhoneIcon />
            </div>
            <span className="text-[10px] text-chat-muted font-medium">Voice</span>
          </button>
          <button className="flex flex-col items-center gap-1.5 px-3 py-3 rounded-xl bg-chat-card hover:bg-chat-card/80 transition-colors">
            <div className="w-9 h-9 rounded-full bg-blue-500/10 flex items-center justify-center text-blue-400">
              <VideoIcon />
            </div>
            <span className="text-[10px] text-chat-muted font-medium">Video</span>
          </button>
          <button className="flex flex-col items-center gap-1.5 px-3 py-3 rounded-xl bg-chat-card hover:bg-chat-card/80 transition-colors">
            <div className="w-9 h-9 rounded-full bg-amber-500/10 flex items-center justify-center text-amber-400">
              <PinIcon />
            </div>
            <span className="text-[10px] text-chat-muted font-medium">Pins</span>
          </button>
        </div>
      </div>

      {/* ── Content ─────────────────────────────────────────────────── */}
      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-3 scrollbar-thin">
        {/* Members */}
        {conv && (
          <AccordionSection title="Members" count={conv.members.length}>
            {conv.members.map((member) => {
              const user = users.find((u) => u.id === member.id)
              if (!user) return null
              const statusColors = {
                online: "bg-chat-online",
                idle: "bg-yellow-400",
                dnd: "bg-red-500",
                offline: "bg-slate-600",
              }
              return (
                <div
                  key={user.id}
                  onClick={() => setSelectedProfileUser(user.id)}
                  className="flex items-center gap-2.5 px-2 py-1.5 rounded-lg hover:bg-chat-card/60 transition-colors cursor-pointer group"
                >
                  <div className="relative shrink-0">
                    <div
                      className="w-8 h-8 rounded-full flex items-center justify-center text-[9px] font-bold text-white"
                      style={{ backgroundColor: user.color }}
                    >
                      {user.name.split(" ").map((w) => w[0]).join("").toUpperCase().slice(0, 2)}
                    </div>
                    <span className={`absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 rounded-full border-2 border-chat-panel ${statusColors[user.status]}`} />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-xs font-medium text-slate-200 truncate group-hover:text-white transition-colors">
                      {user.name}
                    </p>
                    {user.email && (
                      <p className="text-[9px] text-chat-muted/60 truncate leading-tight">{user.email}</p>
                    )}
                    <p className="text-[10px] text-chat-muted capitalize">{user.role}</p>
                  </div>
                  {/* View profile indicator */}
                  <svg
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    className="text-chat-muted opacity-0 group-hover:opacity-100 transition-opacity shrink-0"
                  >
                    <path d="M9 18l6-6-6-6" />
                  </svg>
                </div>
              )
            })}
          </AccordionSection>
        )}

        {/* Photos */}
        {photos.length > 0 && (
          <AccordionSection title="Photos" count={photos.length} defaultOpen={photos.length <= 3}>
            <div className="grid grid-cols-3 gap-1.5">
              {photos.slice(0, 9).map((photo) => (
                <div
                  key={photo.id}
                  className="aspect-square rounded-lg overflow-hidden bg-chat-card cursor-pointer hover:opacity-80 transition-opacity"
                >
                  <Image
                    src={photo.url}
                    alt={photo.name}
                    width={200}
                    height={200}
                    className="w-full h-full object-cover"
                  />
                </div>
              ))}
            </div>
          </AccordionSection>
        )}

        {/* Files */}
        {files.length > 0 && (
          <AccordionSection title="Files" count={files.length}>
            {files.map((file) => (
              <div
                key={file.id}
                className="flex items-center gap-2.5 px-2 py-2 rounded-lg hover:bg-chat-card/60 transition-colors cursor-pointer group"
              >
                <div className="w-8 h-8 rounded-lg bg-chat-card flex items-center justify-center text-chat-muted shrink-0">
                  {file.type === "link" ? (
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
                      <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
                    </svg>
                  ) : (
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
                      <polyline points="14 2 14 8 20 8" />
                      <line x1="16" x2="8" y1="13" y2="13" />
                      <line x1="16" x2="8" y1="17" y2="17" />
                    </svg>
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-xs text-slate-300 truncate font-medium">{file.name}</p>
                  <p className="text-[10px] text-chat-muted">{file.size}</p>
                </div>
                <button className="shrink-0 w-6 h-6 rounded flex items-center justify-center text-chat-muted hover:text-white hover:bg-chat-card transition-all opacity-0 group-hover:opacity-100">
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M7 10l5 5 5-5M12 15V3" />
                  </svg>
                </button>
              </div>
            ))}
          </AccordionSection>
        )}
      </div>
    </div>
  )
}
