"use client"

import { useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { registerUser, loginWithGoogle, clearLoggedOutFlag } from "../../lib/auth"
import { useChatStore } from "../../components/chat/ChatStore"

// ─── SVG Icons ──────────────────────────────────────────────────────────

function GoogleIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none">
      <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4" />
      <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853" />
      <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05" />
      <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335" />
    </svg>
  )
}

function MailIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect x="2" y="4" width="20" height="16" rx="2" />
      <path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7" />
    </svg>
  )
}

function LockIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
      <path d="M7 11V7a5 5 0 0 1 10 0v4" />
    </svg>
  )
}

function EyeIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  )
}

function EyeOffIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M9.88 9.88a3 3 0 1 0 4.24 4.24" />
      <path d="M10.73 5.08A10.43 10.43 0 0 1 12 5c7 0 10 7 10 7a13.16 13.16 0 0 1-1.67 2.68" />
      <path d="M6.61 6.61A13.526 13.526 0 0 0 2 12s3 7 10 7a9.74 9.74 0 0 0 5.39-1.61" />
      <line x1="2" x2="22" y1="2" y2="22" />
    </svg>
  )
}

function UserIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2" />
      <circle cx="12" cy="7" r="4" />
    </svg>
  )
}

function CheckIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M20 6 9 17l-5-5" />
    </svg>
  )
}

function ArrowRightIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M5 12h14M12 5l7 7-7 7" />
    </svg>
  )
}

// ─── Password requirements ──────────────────────────────────────────────

const PASSWORD_REQUIREMENTS = [
  { label: "At least 8 characters", test: (v: string) => v.length >= 8 },
  { label: "Contains a number", test: (v: string) => /\d/.test(v) },
  { label: "Contains a letter", test: (v: string) => /[a-zA-Z]/.test(v) },
]

// ─── Component ──────────────────────────────────────────────────────────

export default function RegisterPage() {
  const router = useRouter()
  const setCurrentUserId = useChatStore((s) => s.setCurrentUserId)

  const [name, setName] = useState("")
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [showPassword, setShowPassword] = useState(false)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState("")
  const [success, setSuccess] = useState(false)

  const passwordValid = PASSWORD_REQUIREMENTS.map((r) => r.test(password))

  async function handleRegister(e: React.FormEvent) {
    e.preventDefault()
    setError("")

    if (!name.trim()) {
      setError("Please enter your name")
      return
    }
    if (!email.trim()) {
      setError("Please enter your email")
      return
    }
    if (!passwordValid.every(Boolean)) {
      setError("Password does not meet all requirements")
      return
    }

    setIsSubmitting(true)
    try {
      clearLoggedOutFlag()
      const user = await registerUser(email.trim(), password, name.trim())
      if (user) {
        setCurrentUserId(user.id, user.name)
        setSuccess(true)
        setTimeout(() => router.push("/"), 1000)
      } else {
        setError("Registration failed. Please try again.")
      }
    } catch {
      setError("Registration failed. Please try again.")
    } finally {
      setIsSubmitting(false)
    }
  }

  function handleGoogleLogin() {
    clearLoggedOutFlag()
    loginWithGoogle()
  }

  // ─── Success state ──────────────────────────────────────────────────
  if (success) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950 flex items-center justify-center p-4">
        <div className="text-center">
          <div className="w-16 h-16 rounded-full bg-chat-accent/20 flex items-center justify-center mx-auto mb-4">
            <CheckIcon />
          </div>
          <h2 className="text-xl font-bold text-white">Account created!</h2>
          <p className="text-sm text-slate-400 mt-1">Redirecting you to the app...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        {/* ── Logo / Brand ────────────────────────────────────────── */}
        <div className="text-center mb-8">
          <div className="w-14 h-14 rounded-2xl bg-gradient-to-br from-chat-accent to-amber-400 flex items-center justify-center text-2xl font-bold text-black mx-auto mb-4 shadow-lg shadow-chat-accent/20">
            RS
          </div>
          <h1 className="text-2xl font-bold text-white">Create an account</h1>
          <p className="text-sm text-slate-400 mt-1">Get started with RideShare</p>
        </div>

        {/* ── Card ────────────────────────────────────────────────── */}
        <div className="bg-slate-900/80 backdrop-blur-xl border border-slate-800 rounded-2xl p-6 shadow-2xl">
          {/* ── Registration Form ─────────────────────────────────── */}
          <form onSubmit={handleRegister} className="space-y-4">
            <div>
              <label htmlFor="name" className="block text-xs font-medium text-slate-400 mb-1.5">
                Display name
              </label>
              <div className="relative">
                <span className="absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-500">
                  <UserIcon />
                </span>
                <input
                  id="name"
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Your name"
                  className="w-full bg-slate-800/80 text-white text-sm rounded-xl pl-10 pr-4 py-3 outline-none border border-slate-700/50 focus:border-chat-accent/50 focus:bg-slate-800 placeholder:text-slate-600 transition-all"
                  autoComplete="name"
                />
              </div>
            </div>

            <div>
              <label htmlFor="email" className="block text-xs font-medium text-slate-400 mb-1.5">
                Email
              </label>
              <div className="relative">
                <span className="absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-500">
                  <MailIcon />
                </span>
                <input
                  id="email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="you@example.com"
                  className="w-full bg-slate-800/80 text-white text-sm rounded-xl pl-10 pr-4 py-3 outline-none border border-slate-700/50 focus:border-chat-accent/50 focus:bg-slate-800 placeholder:text-slate-600 transition-all"
                  autoComplete="email"
                />
              </div>
            </div>

            <div>
              <label htmlFor="password" className="block text-xs font-medium text-slate-400 mb-1.5">
                Password
              </label>
              <div className="relative">
                <span className="absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-500">
                  <LockIcon />
                </span>
                <input
                  id="password"
                  type={showPassword ? "text" : "password"}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Create a password"
                  className="w-full bg-slate-800/80 text-white text-sm rounded-xl pl-10 pr-10 py-3 outline-none border border-slate-700/50 focus:border-chat-accent/50 focus:bg-slate-800 placeholder:text-slate-600 transition-all"
                  autoComplete="new-password"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3.5 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300 transition-colors"
                >
                  {showPassword ? <EyeOffIcon /> : <EyeIcon />}
                </button>
              </div>

              {/* ── Password requirements ──────────────────────────── */}
              {password.length > 0 && (
                <div className="mt-2 space-y-1">
                  {PASSWORD_REQUIREMENTS.map((req, i) => (
                    <div key={i} className="flex items-center gap-2">
                      <div
                        className={`w-3.5 h-3.5 rounded-full flex items-center justify-center transition-colors ${
                          passwordValid[i] ? "bg-chat-accent/20 text-chat-accent" : "bg-slate-800 text-slate-600"
                        }`}
                      >
                        {passwordValid[i] ? (
                          <CheckIcon />
                        ) : (
                          <span className="text-[8px] font-bold">•</span>
                        )}
                      </div>
                      <span className={`text-[11px] ${passwordValid[i] ? "text-slate-400" : "text-slate-600"}`}>
                        {req.label}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {error && (
              <div className="bg-red-500/10 border border-red-500/20 rounded-xl px-4 py-2.5">
                <p className="text-xs text-red-400">{error}</p>
              </div>
            )}

            <button
              type="submit"
              disabled={isSubmitting}
              className="w-full flex items-center justify-center gap-2 bg-chat-accent text-black font-semibold rounded-xl py-3 text-sm hover:opacity-90 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isSubmitting ? (
                <span className="w-4 h-4 border-2 border-black/30 border-t-black rounded-full animate-spin" />
              ) : (
                <>
                  Create account <ArrowRightIcon />
                </>
              )}
            </button>
          </form>

          {/* ── Divider ────────────────────────────────────────────── */}
          <div className="flex items-center gap-3 my-5">
            <div className="flex-1 h-px bg-slate-800" />
            <span className="text-[11px] text-slate-500 font-medium">or</span>
            <div className="flex-1 h-px bg-slate-800" />
          </div>

          {/* ── Google OAuth ───────────────────────────────────────── */}
          <button
            onClick={handleGoogleLogin}
            className="w-full flex items-center justify-center gap-3 bg-white/5 hover:bg-white/10 border border-slate-700/50 text-white rounded-xl py-3 text-sm font-medium transition-all"
          >
            <GoogleIcon />
            Sign up with Google
          </button>
        </div>

        {/* ── Footer ──────────────────────────────────────────────── */}
        <p className="text-center text-sm text-slate-500 mt-6">
          Already have an account?{" "}
          <Link href="/login" className="text-chat-accent hover:underline font-medium">
            Sign in
          </Link>
        </p>
      </div>
    </div>
  )
}
