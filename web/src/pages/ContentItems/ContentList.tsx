import { useState, useEffect, useCallback } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import apiClient from '@/lib/apiClient'

interface ContentItem {
  id: string
  workspace_id: string
  client_id: string | null
  title: string
  description: string
  platform: string
  content_type: string
  status: string
  scheduled_date: string | null
  created_by: string
  created_at: string
  updated_at: string
}

const STATUS_COLORS: Record<string, string> = {
  draft: 'bg-gray-100 text-gray-700',
  review: 'bg-yellow-100 text-yellow-800',
  approved: 'bg-blue-100 text-blue-800',
  published: 'bg-green-100 text-green-800',
  archived: 'bg-red-100 text-red-800',
}

export default function ContentList() {
  const [items, setItems] = useState<ContentItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [searchParams] = useSearchParams()
  const statusFilter = searchParams.get('status') ?? ''

  const loadItems = useCallback(async () => {
    setLoading(true)
    setError('')
    const query = statusFilter ? `?status=${statusFilter}` : ''
    const res = await apiClient.get<ContentItem[]>(`/content-items${query}`)
    if (res.error) {
      setError(res.error.message)
    } else {
      setItems(res.data ?? [])
    }
    setLoading(false)
  }, [statusFilter])

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    loadItems()
  }, [loadItems])

  const statusTabs = ['all', 'draft', 'review', 'approved', 'published', 'archived']

  if (loading) {
    return <p className="text-gray-500 p-4">Loading content...</p>
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold text-gray-900">Content Items</h2>
        <Link
          to="/dashboard/content-items/new"
          className="rounded-lg bg-socialflow-600 px-4 py-2 text-sm font-medium text-white hover:bg-socialflow-700 transition-colors"
        >
          + New Item
        </Link>
      </div>

      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 mb-4">
          {error}
        </div>
      )}

      {/* Status filter tabs */}
      <div className="flex gap-1 mb-4 flex-wrap">
        {statusTabs.map((s) => (
          <Link
            key={s}
            to={s === 'all' ? '/dashboard/content-items' : `/dashboard/content-items?status=${s}`}
            className={`rounded-md px-3 py-1.5 text-xs font-medium transition-colors ${
              (s === 'all' && !statusFilter) || statusFilter === s
                ? 'bg-socialflow-100 text-socialflow-700'
                : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
            }`}
          >
            {s.charAt(0).toUpperCase() + s.slice(1)}
          </Link>
        ))}
      </div>

      {items.length === 0 ? (
        <div className="rounded-lg border border-dashed border-gray-300 bg-white p-12 text-center">
          <p className="text-gray-500">No content items found. Create your first one to get started.</p>
        </div>
      ) : (
        <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="text-left px-4 py-3 font-medium text-gray-600">Title</th>
                <th className="text-left px-4 py-3 font-medium text-gray-600">Status</th>
                <th className="text-left px-4 py-3 font-medium text-gray-600">Platform</th>
                <th className="text-left px-4 py-3 font-medium text-gray-600">Scheduled</th>
                <th className="text-right px-4 py-3 font-medium text-gray-600">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {items.map((item) => (
                <tr key={item.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3">
                    <Link
                      to={`/dashboard/content-items/${item.id}`}
                      className="text-socialflow-600 hover:text-socialflow-700 font-medium"
                    >
                      {item.title}
                    </Link>
                    <p className="text-xs text-gray-400 mt-0.5">
                      {item.content_type} · {item.platform}
                    </p>
                  </td>
                  <td className="px-4 py-3">
                    <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_COLORS[item.status] ?? 'bg-gray-100 text-gray-700'}`}>
                      {item.status}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-gray-600 capitalize">{item.platform}</td>
                  <td className="px-4 py-3 text-gray-500 text-xs">
                    {item.scheduled_date ?? '—'}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Link
                      to={`/dashboard/content-items/${item.id}/edit`}
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
