import { useState, useEffect, useCallback, type FormEvent } from 'react'
import { useParams, Link } from 'react-router-dom'
import apiClient from '@/lib/apiClient'
import { NEXT_STATUS } from '@/lib/statusTransitions'

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
  comments: Comment[]
}

interface Comment {
  id: string
  content_item_id: string
  author_id: string
  body: string
  created_at: string
  author_name?: string
  author_email?: string
}

const STATUS_COLORS: Record<string, string> = {
  draft: 'bg-gray-100 text-gray-700',
  review: 'bg-yellow-100 text-yellow-800',
  approved: 'bg-blue-100 text-blue-800',
  published: 'bg-green-100 text-green-800',
  archived: 'bg-red-100 text-red-800',
}

export default function ContentDetail() {
  const { id } = useParams<{ id: string }>()
  const [item, setItem] = useState<ContentItem | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [commentBody, setCommentBody] = useState('')
  const [submittingComment, setSubmittingComment] = useState(false)
  const [transitioning, setTransitioning] = useState(false)

  const loadItem = useCallback(async () => {
    setLoading(true)
    setError('')
    const res = await apiClient.get<ContentItem>(`/content-items/${id}`)
    if (res.error) {
      setError(res.error.message)
    } else {
      setItem(res.data ?? null)
    }
    setLoading(false)
  }, [id])

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    loadItem()
  }, [loadItem])

  async function handleTransition(newStatus: string) {
    setTransitioning(true)
    const res = await apiClient.patch<ContentItem>(`/content-items/${id}/status`, { status: newStatus })
    if (res.error) {
      setError(res.error.message)
    } else if (res.data) {
      setItem((prev) => (prev ? { ...prev, status: res.data!.status, updated_at: res.data!.updated_at } : prev))
      setError('')
    }
    setTransitioning(false)
  }

  async function handleAddComment(e: FormEvent) {
    e.preventDefault()
    if (!commentBody.trim()) return
    setSubmittingComment(true)
    const res = await apiClient.post<Comment>(`/content-items/${id}/comments`, { body: commentBody })
    if (res.error) {
      setError(res.error.message)
    } else if (res.data) {
      setItem((prev) =>
        prev ? { ...prev, comments: [...(prev.comments ?? []), res.data!] } : prev,
      )
      setCommentBody('')
      setError('')
    }
    setSubmittingComment(false)
  }

  async function handleDeleteComment(commentId: string) {
    const res = await apiClient.delete(`/comments/${commentId}`)
    if (res.error) {
      setError(res.error.message)
      return
    }
    setItem((prev) =>
      prev ? { ...prev, comments: (prev.comments ?? []).filter((c) => c.id !== commentId) } : prev,
    )
  }

  if (loading) {
    return <p className="text-gray-500 p-4">Loading...</p>
  }

  if (!item) {
    return (
      <div className="p-4">
        <p className="text-red-600">{error || 'Content item not found'}</p>
        <Link to="/content-items" className="text-socialflow-600 hover:underline text-sm mt-2 inline-block">
          ← Back to list
        </Link>
      </div>
    )
  }

  const allowedTransitions = NEXT_STATUS[item.status] ?? []

  return (
    <div className="max-w-3xl">
      {/* Breadcrumb */}
      <div className="mb-4">
        <Link to="/dashboard/content-items" className="text-sm text-socialflow-600 hover:text-socialflow-700">
          ← Content Items
        </Link>
      </div>

      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 mb-4">
          {error}
        </div>
      )}

      {/* Item header */}
      <div className="bg-white rounded-lg border border-gray-200 p-6 mb-6">
        <div className="flex items-start justify-between mb-4">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">{item.title}</h1>
            <div className="flex gap-2 mt-2">
              <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${STATUS_COLORS[item.status] ?? ''}`}>
                {item.status}
              </span>
              <span className="text-xs text-gray-500 capitalize">{item.platform} · {item.content_type}</span>
              {item.scheduled_date && (
                <span className="text-xs text-gray-500">Scheduled: {item.scheduled_date}</span>
              )}
            </div>
          </div>
          <Link
            to={`/dashboard/content-items/${item.id}/edit`}
            className="text-sm text-socialflow-600 hover:text-socialflow-700 font-medium"
          >
            Edit
          </Link>
        </div>

        {item.description && (
          <div className="mt-4 text-sm text-gray-600 whitespace-pre-wrap">{item.description}</div>
        )}

        {/* Status transitions */}
        {allowedTransitions.length > 0 && (
          <div className="mt-6 pt-4 border-t border-gray-100">
            <p className="text-xs text-gray-500 mb-2">Move to:</p>
            <div className="flex gap-2">
              {allowedTransitions.map((next) => (
                <button
                  key={next}
                  onClick={() => handleTransition(next)}
                  disabled={transitioning}
                  className="rounded-md border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50"
                >
                  {next.charAt(0).toUpperCase() + next.slice(1)}
                </button>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Comments section */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">
          Comments ({item.comments?.length ?? 0})
        </h3>

        {/* Add comment form */}
        <form onSubmit={handleAddComment} className="mb-6">
          <textarea
            value={commentBody}
            onChange={(e) => setCommentBody(e.target.value)}
            placeholder="Write a comment..."
            rows={2}
            className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-socialflow-500 focus:outline-none focus:ring-1 focus:ring-socialflow-500"
          />
          <button
            type="submit"
            disabled={submittingComment || !commentBody.trim()}
            className="mt-2 rounded-lg bg-socialflow-600 px-4 py-1.5 text-xs font-medium text-white hover:bg-socialflow-700 transition-colors disabled:opacity-50"
          >
            {submittingComment ? 'Posting...' : 'Add Comment'}
          </button>
        </form>

        {/* Comments list */}
        {(!item.comments || item.comments.length === 0) ? (
          <p className="text-sm text-gray-400">No comments yet.</p>
        ) : (
          <div className="space-y-3">
            {item.comments.map((c) => (
              <div key={c.id} className="rounded-lg bg-gray-50 px-4 py-3">
                <div className="flex items-start justify-between">
                  <div>
                    <p className="text-xs font-medium text-gray-700">
                      {c.author_name || c.author_email || c.author_id}
                    </p>
                    <p className="text-xs text-gray-400">
                      {new Date(c.created_at).toLocaleString()}
                    </p>
                  </div>
                  <button
                    onClick={() => handleDeleteComment(c.id)}
                    className="text-xs text-red-500 hover:text-red-700 font-medium"
                  >
                    Delete
                  </button>
                </div>
                <p className="mt-1 text-sm text-gray-600">{c.body}</p>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
