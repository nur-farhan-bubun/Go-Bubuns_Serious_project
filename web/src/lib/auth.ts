// ─── Auth Service ───────────────────────────────────────────────────────
// Handles authentication flow: Google OAuth, JWT token management,
// and user identity for the chat and other services.
//
// For development/demo: supports simulated login with proper HS256 JWTs.
// For production: redirects to the user-service Google OAuth endpoint,
// stores the returned JWT token, and passes it to API calls.
//
// ─── Dev JWT Secret ────────────────────────────────────────────────────
// This secret is used by loginSimulated() to sign HS256 JWTs on the client.
// For end-to-end auth in development, set the same value as CLERK_JWT_KEY
// in the API Gateway's environment:
//
//   CLERK_JWT_KEY=dev-jwt-secret-do-not-use-in-production
//
// The gateway's HS256 validation will accept tokens signed with this secret.

// ─── Configuration ──────────────────────────────────────────────────────

// API base URL. Defaults to same-origin (empty string) so requests go through
// Next.js rewrites (next.config.ts) which proxy to the API gateway on port 8080.
const API_BASE = process.env.NEXT_PUBLIC_API_URL || ""
const AUTH_API = `${API_BASE}/v1/auth`

/** Dev secret for signing simulated JWTs. Must match API Gateway's CLERK_JWT_KEY. */
const DEV_JWT_SECRET = "dev-jwt-secret-do-not-use-in-production"

// ─── Token Storage Keys ─────────────────────────────────────────────────

const TOKEN_KEY = "chat_auth_token"
const USER_ID_KEY = "chat_user_id"
const USER_NAME_KEY = "chat_user_name"
const USER_EMAIL_KEY = "chat_user_email"

/** localStorage flag set by logout() so AuthInitializer skips auto-login. */
const LOGGED_OUT_FLAG_KEY = "chat_logged_out"

// ─── Types ──────────────────────────────────────────────────────────────

export interface AuthUser {
  id: string
  email: string
  name: string
  avatar_url: string
}

export interface AuthState {
  token: string | null
  user: AuthUser | null
  isAuthenticated: boolean
}

// ─── Token Management ───────────────────────────────────────────────────

export function getToken(): string | null {
  if (typeof window === "undefined") return null
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

export function getStoredUserId(): string | null {
  if (typeof window === "undefined") return null
  return localStorage.getItem(USER_ID_KEY)
}

export function getStoredUserName(): string | null {
  if (typeof window === "undefined") return null
  return localStorage.getItem(USER_NAME_KEY)
}

function storeUserInfo(user: AuthUser): void {
  localStorage.setItem(USER_ID_KEY, user.id)
  localStorage.setItem(USER_NAME_KEY, user.name)
  localStorage.setItem(USER_EMAIL_KEY, user.email)
}

export function clearUserInfo(): void {
  localStorage.removeItem(USER_ID_KEY)
  localStorage.removeItem(USER_NAME_KEY)
  localStorage.removeItem(USER_EMAIL_KEY)
}

export function isAuthenticated(): boolean {
  return !!getToken()
}

export function getAuthHeaders(): Record<string, string> {
  const token = getToken()
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  }
  if (token) {
    headers["Authorization"] = `Bearer ${token}`
  }
  return headers
}

// ─── JWT Signing (HS256 via Web Crypto API) ─────────────────────────────

/**
 * Base64url encode a string or buffer (RFC 4648 §5).
 * Uses URL-safe characters and strips padding.
 */
function base64urlEncode(data: Uint8Array): string {
  return btoa(String.fromCharCode(...data))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "")
}

/**
 * Base64url encode a JSON-serializable object.
 */
function base64urlEncodeJSON(obj: Record<string, unknown>): string {
  const json = JSON.stringify(obj)
  const bytes = new TextEncoder().encode(json)
  return base64urlEncode(bytes)
}

/**
 * Sign a payload with HS256 using the Web Crypto API.
 *
 * Returns a complete JWT string: header.payload.signature
 */
async function signHS256(payload: Record<string, unknown>, secret: string): Promise<string> {
  const header = { alg: "HS256", typ: "JWT" }
  const headerEncoded = base64urlEncodeJSON(header)
  const payloadEncoded = base64urlEncodeJSON(payload)

  const signingInput = `${headerEncoded}.${payloadEncoded}`

  // Import the secret key
  const keyData = new TextEncoder().encode(secret)
  const cryptoKey = await crypto.subtle.importKey(
    "raw",
    keyData,
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  )

  // Compute HMAC-SHA256 signature
  const signatureBytes = await crypto.subtle.sign(
    "HMAC",
    cryptoKey,
    new TextEncoder().encode(signingInput),
  )

  const signatureEncoded = base64urlEncode(new Uint8Array(signatureBytes))

  return `${signingInput}.${signatureEncoded}`
}

// ─── Google OAuth Login ─────────────────────────────────────────────────

/**
 * Redirect the user to Google OAuth login page.
 * The user-service backend handles the OAuth flow.
 */
export function loginWithGoogle(): void {
  window.location.href = `${AUTH_API}/google/login`
}

/**
 * Handle the OAuth callback from Google.
 * Called after the user is redirected back from Google with a code and state.
 * Exchanges the code for a JWT token and stores it.
 */
export async function handleAuthCallback(code: string, state: string): Promise<AuthUser | null> {
  try {
    const res = await fetch(`${AUTH_API}/google/callback?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`, {
      headers: { "Content-Type": "application/json" },
    })
    if (!res.ok) {
      console.error("[Auth] Callback failed:", res.status)
      return null
    }

    const data = await res.json()
    const user: AuthUser = {
      id: data.user?.id || "",
      email: data.user?.email || "",
      name: data.user?.name || "",
      avatar_url: data.user?.avatar_url || "",
    }
    const token: string = data.token || ""

    if (token && user.id) {
      setToken(token)
      storeUserInfo(user)
      addLocalRegisteredUser(user)
      return user
    }

    return null
  } catch (err) {
    console.error("[Auth] Callback error:", err)
    return null
  }
}

/**
 * Simulated login for development — creates a proper HS256 JWT signed with
 * a known dev secret and stores user info.
 *
 * The generated JWT can be validated by the API Gateway's auth middleware
 * when CLERK_JWT_KEY is set to the same DEV_JWT_SECRET value.
 *
 * This lets you test auth end-to-end without a Clerk account.
 *
 * @param userId - User ID. If not provided, a new one is generated.
 * @param displayName - Optional display name.
 * @param email - Optional email address.
 */
export async function loginSimulated(userId?: string, displayName?: string, email?: string): Promise<AuthUser> {
  const finalEmail = email || (userId ? `${userId}@demo.local` : `user-${Date.now()}@demo.local`)
  const finalName = displayName || (userId ? userId.replace(/^user-/, "").replace(/[-_]/g, " ").replace(/\b\w/g, (c) => c.toUpperCase()) : "User")

  // Try to register with the user-service to get a real UUID assigned.
  // This UUID is used as the user's ID in the JWT so frontend and backend
  // agree on identities — critical for conversation sharing.
  // If registration fails (e.g. user already exists), look up the UUID by email.
  let finalId = userId || `user-${Date.now()}`
  const registeredUuid = await registerUserWithService(finalEmail, finalName)
  if (registeredUuid) {
    finalId = registeredUuid
  } else {
    // User already exists — look up their UUID from the user list
    const existingUuid = await findUserIdByEmail(finalEmail)
    if (existingUuid) {
      finalId = existingUuid
    }
  }

  const user: AuthUser = {
    id: finalId,
    email: finalEmail,
    name: finalName,
    avatar_url: "",
  }

  const now = Math.floor(Date.now() / 1000)

  const token = await signHS256(
    {
      sub: user.id,
      email: user.email,
      name: user.name,
      iat: now,
      exp: now + 86400, // 24 hours
      iss: "dev-simulated",
    },
    DEV_JWT_SECRET,
  )

  setToken(token)
  storeUserInfo(user)
  clearLoggedOutFlag()

  // Save to local registered users registry so other users can see this user
  addLocalRegisteredUser(user)

  return user
}

/**
 * Log out: clear stored token and user info, and set a flag so
 * AuthInitializer skips auto-login on next page load.
 */
export function logout(): void {
  clearToken()
  clearUserInfo()
  // Set flag so AuthInitializer won't auto-sign-in on reload
  localStorage.setItem(LOGGED_OUT_FLAG_KEY, Date.now().toString())
  // Redirect to login page
  if (typeof window !== "undefined") {
    window.location.href = "/login"
  }
}

/**
 * Check whether the user recently logged out (to skip auto-login).
 */
export function wasRecentlyLoggedOut(): boolean {
  const val = localStorage.getItem(LOGGED_OUT_FLAG_KEY)
  if (!val) return false
  const elapsed = Date.now() - Number(val)
  // Flag expires after 10 seconds — long enough to survive a page reload
  if (elapsed > 10_000) {
    localStorage.removeItem(LOGGED_OUT_FLAG_KEY)
    return false
  }
  return true
}

/**
 * Clear the logged-out flag (called after a successful login).
 */
export function clearLoggedOutFlag(): void {
  localStorage.removeItem(LOGGED_OUT_FLAG_KEY)
}

// ─── User-Service API helpers ───────────────────────────────────────────

/**
 * Register a user via the user-service API and return the UUID assigned by the backend.
 * POST /v1/users — proxied to user-service by the API gateway.
 * Returns the user ID (UUID) on success, or null if registration fails.
 */
export async function registerUserWithService(email: string, displayName: string): Promise<string | null> {
  try {
    const res = await fetch(`${API_BASE}/v1/users`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, display_name: displayName }),
    })
    if (!res.ok) return null
    const data = await res.json()
    return (data.id as string) || null
  } catch {
    return null
  }
}

/**
 * Look up a user's UUID by email from the user-service.
 * Used when registration fails because the user already exists.
 * GET /v1/users returns all users; we find the matching email.
 */
export async function findUserIdByEmail(email: string): Promise<string | null> {
  try {
    const res = await fetch(`${API_BASE}/v1/users`, {
      headers: { "Content-Type": "application/json" },
    })
    if (!res.ok) return null
    const data = await res.json()
    const usersList: Record<string, unknown>[] = data.users || []
    if (!Array.isArray(usersList)) return null
    const user = usersList.find((u) => u.email === email)
    return (user?.id as string) || null
  } catch {
    return null
  }
}

// ─── Local Registered Users Registry ────────────────────────────────────
// Tracks all users who have ever logged in via this browser, so they
// appear in each other's user directory even without a running backend.

const LOCAL_REGISTERED_USERS_KEY = "chat_local_registered_users"

/**
 * Get all registered users from localStorage.
 */
export function getLocalRegisteredUsers(): AuthUser[] {
  if (typeof window === "undefined") return []
  try {
    const raw = localStorage.getItem(LOCAL_REGISTERED_USERS_KEY)
    if (!raw) return []
    return JSON.parse(raw) as AuthUser[]
  } catch {
    return []
  }
}

/**
 * Add or update a user in the local registered users registry.
 * Deduplicates by user ID (keeps the latest entry).
 */
export function addLocalRegisteredUser(user: AuthUser): void {
  if (typeof window === "undefined") return
  try {
    const users = getLocalRegisteredUsers()
    const idx = users.findIndex((u) => u.id === user.id)
    if (idx >= 0) {
      users[idx] = user
    } else {
      users.push(user)
    }
    localStorage.setItem(LOCAL_REGISTERED_USERS_KEY, JSON.stringify(users))
  } catch {
    // Silently fail
  }
}

// ─── Email/Password Registration ─────────────────────────────────────────

/**
 * Register a new user via the user-service API, falling back to creating a
 * simulated user when the backend is unavailable.
 *
 * POST /v1/users  → creates the user, then generates a dev JWT.
 */
export async function registerUser(email: string, password: string, displayName: string): Promise<AuthUser | null> {
  // First try the real API
  try {
    const res = await fetch(`${API_BASE}/v1/users`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, display_name: displayName }),
    })
    if (res.ok) {
      const data = await res.json()
      const user: AuthUser = {
        id: data.id || `user-${Date.now()}`,
        email: data.email || email,
        name: data.profile?.display_name || displayName,
        avatar_url: data.profile?.avatar_url || "",
      }
      // Simulate a JWT since the user-service doesn't issue one on registration
      const now = Math.floor(Date.now() / 1000)
      const token = await signHS256(
        { sub: user.id, email: user.email, name: user.name, iat: now, exp: now + 86400, iss: "dev-simulated" },
        DEV_JWT_SECRET,
      )
      setToken(token)
      storeUserInfo(user)
      clearLoggedOutFlag()
      addLocalRegisteredUser(user)
      return user
    }
  } catch {
    // Fall through to simulated login
  }

  // Fallback: create a simulated user
  const fallbackId = `user-${Date.now()}`
  const fallbackUser: AuthUser = {
    id: fallbackId,
    email,
    name: displayName,
    avatar_url: "",
  }
  const now = Math.floor(Date.now() / 1000)
  const token = await signHS256(
    { sub: fallbackId, email, name: displayName, iat: now, exp: now + 86400, iss: "dev-simulated" },
    DEV_JWT_SECRET,
  )
  setToken(token)
  storeUserInfo(fallbackUser)
  clearLoggedOutFlag()
  addLocalRegisteredUser(fallbackUser)
  return fallbackUser
}

// ─── Token Status (expiry checking) ────────────────────────────────────

export type TokenStatus = "valid" | "expiring-soon" | "expired" | "no-token"

export interface TokenInfo {
  status: TokenStatus
  /** Seconds until expiry. Negative if expired. */
  secondsRemaining: number
  /** ISO string of the expiry time. */
  expiresAt?: string
  /** Subject (user ID) from the token, if any. */
  sub?: string
}

/**
 * Decode a JWT token's payload without verification.
 * Returns null if the token is not a valid 3-part JWT.
 *
 * NOTE: This only base64-decodes the payload — it does NOT verify
 * the signature. Use this only for UI display purposes (expiry,
 * user ID). The actual token validation happens server-side.
 */
export function decodeToken(token: string): Record<string, unknown> | null {
  try {
    const parts = token.split(".")
    if (parts.length !== 3) return null

    // Base64url-decode the payload (middle part)
    let base64 = parts[1].replace(/-/g, "+").replace(/_/g, "/")
    // Add padding if needed
    while (base64.length % 4 !== 0) base64 += "="

    const json = atob(base64)
    return JSON.parse(json)
  } catch {
    return null
  }
}

/**
 * Get the status and expiry info for the stored JWT token.
 * Returns "no-token" if no token is stored.
 */
export function getTokenStatus(): TokenInfo {
  const token = getToken()
  if (!token) {
    return { status: "no-token", secondsRemaining: 0 }
  }

  const payload = decodeToken(token)
  if (!payload) {
    return { status: "no-token", secondsRemaining: 0 }
  }

  const exp = payload.exp as number | undefined
  if (!exp) {
    // Token has no expiry — treat as valid
    return {
      status: "valid",
      secondsRemaining: Infinity,
      sub: payload.sub as string | undefined,
    }
  }

  const now = Math.floor(Date.now() / 1000)
  const secondsRemaining = exp - now

  let status: TokenStatus
  if (secondsRemaining <= 0) {
    status = "expired"
  } else if (secondsRemaining < 300) {
    // Less than 5 minutes remaining
    status = "expiring-soon"
  } else {
    status = "valid"
  }

  return {
    status,
    secondsRemaining,
    expiresAt: new Date(exp * 1000).toISOString(),
    sub: payload.sub as string | undefined,
  }
}

/**
 * Get the current auth state from storage.
 */
export function getAuthState(): AuthState {
  const token = getToken()
  const userId = getStoredUserId()
  const userName = getStoredUserName()

  if (!token || !userId) {
    return { token: null, user: null, isAuthenticated: false }
  }

  return {
    token,
    user: { id: userId, name: userName || "", email: "", avatar_url: "" },
    isAuthenticated: true,
  }
}
