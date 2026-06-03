import { createContext, useState, useEffect, useCallback, type ReactNode } from 'react'
import apiClient, { setUnauthorizedHandler, resetUnauthorizedHandler, resetUnauthorizedDebounce } from '@/lib/apiClient'

export interface CurrentUser {
  id: string
  email: string
  active_workspace_id: string
  role: string
  name?: string
}

interface MeContextValue {
  user: CurrentUser | null
  loading: boolean
  error: string | null
  refresh: () => Promise<void>
  reauthenticate: () => Promise<boolean>
  login: (user: CurrentUser) => void
  logout: () => Promise<void>
  switchWorkspace: (workspaceId: string) => Promise<{ active_workspace_id: string; role: string } | null>
}

const MeContext = createContext<MeContextValue | null>(null)

export function MeProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<CurrentUser | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    setLoading(true)
    setError(null)

    const res = await apiClient.get<CurrentUser>('/me')

    if (res.error) {
      if (res.error.code === 'unauthorized') {
        setUser(null)
      } else {
        setError(res.error.message)
      }
    } else if (res.data) {
      setUser(res.data)
    }

    setLoading(false)
  }, [])

  useEffect(() => {
    apiClient.get<CurrentUser>('/me').then((res) => {
      if (res.error) {
        if (res.error.code === 'unauthorized') {
          setUser(null)
        } else {
          setError(res.error.message)
        }
      } else if (res.data) {
        setUser(res.data)
      }
      setLoading(false)
    })
  }, [])

  // Register 401 interceptor so any API call that gets 401 clears user state
  useEffect(() => {
    setUnauthorizedHandler(() => setUser(null))
    return () => {
      resetUnauthorizedHandler()
    }
  }, [])

  function login(user: CurrentUser) {
    setUser(user)
    setError(null)
  }

  async function logout() {
    await apiClient.post('/auth/logout')
    setUser(null)
  }

  async function switchWorkspace(workspaceId: string) {
    const res = await apiClient.post<{ active_workspace_id: string; role: string }>(
      '/workspaces/switch',
      { workspace_id: workspaceId },
    )

    if (res.error) {
      setError(res.error.message)
      return null
    }

    if (res.data) {
      // Update local user state with new active workspace
      setUser((prev) =>
        prev
          ? {
              ...prev,
              active_workspace_id: res.data!.active_workspace_id,
              role: res.data!.role,
            }
          : null,
      )
      return res.data
    }

    return null
  }

  async function reauthenticate(): Promise<boolean> {
    resetUnauthorizedDebounce()
    setLoading(true)
    setError(null)

    const res = await apiClient.get<CurrentUser>('/me')

    if (res.error) {
      if (res.error.code === 'unauthorized') {
        setUser(null)
      } else {
        setError(res.error.message)
      }
      setLoading(false)
      return false
    }

    if (res.data) {
      setUser(res.data)
    }

    setLoading(false)
    return res.data != null
  }

  return (
    <MeContext.Provider value={{ user, loading, error, refresh, reauthenticate, login, logout, switchWorkspace }}>
      {children}
    </MeContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export { useMe } from './useMe'
export default MeContext
