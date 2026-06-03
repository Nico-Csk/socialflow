import { useState, useEffect } from 'react'
import { useMe } from '@/context/MeContext'
import apiClient from '@/lib/apiClient'

interface Workspace {
  id: string
  name: string
}

export default function WorkspaceSwitcher() {
  const { user, switchWorkspace } = useMe()
  const [workspaces, setWorkspaces] = useState<Workspace[]>([])
  const [open, setOpen] = useState(false)
  const [switching, setSwitching] = useState(false)

  useEffect(() => {
    async function load() {
      const res = await apiClient.get<Workspace[]>('/workspaces')
      if (res.data) {
        setWorkspaces(res.data)
      }
    }
    load()
  }, [])

  const currentWs = workspaces.find((w) => w.id === user?.active_workspace_id)

  // Close dropdown on Escape key (document-level listener since overlay div is not focusable)
  useEffect(() => {
    if (!open) return
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [open])

  async function handleSwitch(wsId: string) {
    if (wsId === user?.active_workspace_id) {
      setOpen(false)
      return
    }
    setSwitching(true)
    await switchWorkspace(wsId)
    setSwitching(false)
    setOpen(false)
  }

  if (!user) return null

  return (
    <div className="relative">
      <button
        onClick={() => setOpen(!open)}
        disabled={switching}
        className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50"
      >
        <span className="max-w-[160px] truncate">
          {currentWs ? currentWs.name : 'No workspace'}
        </span>
        <svg className="h-4 w-4 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {open && (
        <div className="absolute right-0 mt-1 w-56 rounded-lg border border-gray-200 bg-white shadow-lg z-50">
          <div className="px-3 py-2 text-xs font-medium text-gray-500 uppercase">Workspaces</div>
          {workspaces.map((ws) => (
            <button
              key={ws.id}
              onClick={() => handleSwitch(ws.id)}
              className={`w-full text-left px-4 py-2 text-sm hover:bg-gray-50 transition-colors ${
                ws.id === user.active_workspace_id
                  ? 'font-medium text-socialflow-600 bg-socialflow-50'
                  : 'text-gray-700'
              }`}
            >
              {ws.name}
              {ws.id === user.active_workspace_id && (
                <span className="ml-2 text-xs text-socialflow-500">(active)</span>
              )}
            </button>
          ))}
          {workspaces.length === 0 && (
            <p className="px-4 py-3 text-sm text-gray-400">No workspaces yet.</p>
          )}
        </div>
      )}

      {/* Click-outside closer (Escape handled via useEffect document listener) */}
      {open && (
        <div
          className="fixed inset-0 z-40"
          onClick={() => setOpen(false)}
        />
      )}
    </div>
  )
}
