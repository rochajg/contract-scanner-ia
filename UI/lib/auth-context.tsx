"use client"

import {
  createContext,
  useContext,
  useEffect,
  useState,
  useCallback,
  type ReactNode,
} from "react"
import { useRouter } from "next/navigation"
import { loginUser, registerUser } from "@/lib/api"

export interface AuthUser {
  userId: string
  username: string
  token: string
}

interface AuthContextType {
  user: AuthUser | null
  isLoading: boolean
  login: (email: string, password: string) => Promise<void>
  register: (username: string, email: string, password: string) => Promise<void>
  logout: () => void
}

const AuthContext = createContext<AuthContextType | null>(null)

const TOKEN_KEY = "auth_token"
const USER_KEY = "auth_user"

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const router = useRouter()

  useEffect(() => {
    try {
      const token = localStorage.getItem(TOKEN_KEY)
      const userRaw = localStorage.getItem(USER_KEY)
      if (token && userRaw) {
        const parsed = JSON.parse(userRaw) as AuthUser
        setUser({ ...parsed, token })
      }
    } catch {
      // corrupted storage — ignore
    } finally {
      setIsLoading(false)
    }
  }, [])

  const persistAuth = useCallback((data: { token: string; user_id: string; username: string }) => {
    const authUser: AuthUser = {
      userId: data.user_id,
      username: data.username,
      token: data.token,
    }
    localStorage.setItem(TOKEN_KEY, data.token)
    localStorage.setItem(USER_KEY, JSON.stringify(authUser))
    setUser(authUser)
  }, [])

  const login = useCallback(
    async (email: string, password: string) => {
      const data = await loginUser(email, password)
      persistAuth(data)
      router.push("/")
    },
    [persistAuth, router]
  )

  const register = useCallback(
    async (username: string, email: string, password: string) => {
      const data = await registerUser(username, email, password)
      persistAuth(data)
      router.push("/")
    },
    [persistAuth, router]
  )

  const logout = useCallback(() => {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_KEY)
    setUser(null)
    router.push("/login")
  }, [router])

  return (
    <AuthContext.Provider value={{ user, isLoading, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthContextType {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error("useAuth must be used inside AuthProvider")
  }
  return ctx
}
