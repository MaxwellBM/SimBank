import { createContext, useContext, useState, useCallback, useEffect } from 'react'
import client, { setTokenProvider } from '../api/client'

const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [user, setUser] = useState(() => {
    try {
      const saved = sessionStorage.getItem('simbank_user')
      return saved ? JSON.parse(saved) : null
    } catch {
      return null
    }
  })
  const [token, setToken] = useState(() => sessionStorage.getItem('simbank_token') || null)
  const [loading, setLoading] = useState(false)

  const updateSession = useCallback((newToken, newUser) => {
    if (newToken) {
      sessionStorage.setItem('simbank_token', newToken)
      sessionStorage.setItem('simbank_user', JSON.stringify(newUser))
    } else {
      sessionStorage.removeItem('simbank_token')
      sessionStorage.removeItem('simbank_user')
    }
    setToken(newToken)
    setUser(newUser)
  }, [])

  useEffect(() => {
    setTokenProvider(() => token)
  }, [token])

  const login = useCallback(async (email, password) => {
    setLoading(true)
    try {
      const { data } = await client.post('/auth/login', { email, password })
      updateSession(data.token, data.user)
      return data.user
    } finally {
      setLoading(false)
    }
  }, [updateSession])

  const register = useCallback(async (email, password, fullName) => {
    setLoading(true)
    try {
      const { data } = await client.post('/auth/register', {
        email, password, full_name: fullName,
      })
      updateSession(data.token, data.user)
      return data.user
    } finally {
      setLoading(false)
    }
  }, [updateSession])

  const logout = useCallback(async () => {
    try {
      await client.post('/auth/logout')
    } catch {
      // ignore
    }
    updateSession(null, null)
  }, [updateSession])

  return (
    <AuthContext.Provider value={{ user, token, loading, login, register, logout, isAuthenticated: !!token }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
