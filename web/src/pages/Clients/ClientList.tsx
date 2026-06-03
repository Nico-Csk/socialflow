import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import apiClient from '@/lib/apiClient'

interface Client {
  id: string
  workspace_id: string
  name: string
  social_handles: Record<string, string>
  notes: string
  active: boolean
  created_at: string
}

export default function ClientList() {
  const [clients, setClients] = useState<Client[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    apiClient.get<Client[]>('/clients').then((res) => {
      if (res.error) {
        setError(res.error.message)
      } else {
        setClients(res.data ?? [])
      }
      setLoading(false)
    })
  }, [])

  if (loading) {
    return <p className="text-gray-500 p-4">Loading clients...</p>
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold text-gray-900">Clients</h2>
        <Link
          to="/dashboard/clients/new"
          className="rounded-lg bg-socialflow-600 px-4 py-2 text-sm font-medium text-white hover:bg-socialflow-700 transition-colors"
        >
          + New Client
        </Link>
      </div>

      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 mb-4">
          {error}
        </div>
      )}

      {clients.length === 0 ? (
        <div className="rounded-lg border border-dashed border-gray-300 bg-white p-12 text-center">
          <p className="text-gray-500">No clients yet. Create your first client to get started.</p>
        </div>
      ) : (
        <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="text-left px-4 py-3 font-medium text-gray-600">Name</th>
                <th className="text-left px-4 py-3 font-medium text-gray-600">Active</th>
                <th className="text-left px-4 py-3 font-medium text-gray-600">Handles</th>
                <th className="text-right px-4 py-3 font-medium text-gray-600">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {clients.map((c) => (
                <tr key={c.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3">
                    <span className="text-gray-900 font-medium">
                      {c.name}
                    </span>
                    {c.notes && <p className="text-xs text-gray-400 mt-0.5 truncate max-w-xs">{c.notes}</p>}
                  </td>
                  <td className="px-4 py-3">
                    {c.active ? (
                      <span className="inline-flex items-center rounded-full bg-green-50 px-2 py-0.5 text-xs font-medium text-green-700">Active</span>
                    ) : (
                      <span className="inline-flex items-center rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600">Inactive</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-gray-500 text-xs">
                    {c.social_handles && Object.keys(c.social_handles).length > 0
                      ? Object.entries(c.social_handles).map(([k, v]) => (
                          <span key={k} className="mr-2">
                            <span className="font-medium">{k}:</span> {String(v)}
                          </span>
                        ))
                      : '—'}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Link
                      to={`/dashboard/clients/${c.id}/edit`}
                      className="text-xs text-socialflow-600 hover:text-socialflow-700 font-medium"
                    >
                      Edit
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
