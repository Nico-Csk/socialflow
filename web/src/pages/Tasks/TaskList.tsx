import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import apiClient from '@/lib/apiClient'

interface Task {
  id: string
  workspace_id: string
  title: string
  description: string
  assignee_id: string | null
  due_date: string | null
  done: boolean
  content_item_id: string | null
  client_id: string | null
  created_at: string
  updated_at: string
}

export default function TaskList() {
  const [tasks, setTasks] = useState<Task[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    apiClient.get<Task[]>('/tasks').then((res) => {
      if (res.error) {
        setError(res.error.message)
      } else {
        setTasks(res.data ?? [])
      }
      setLoading(false)
    })
  }, [])

  async function handleToggle(task: Task) {
    const res = await apiClient.put<Task>(`/tasks/${task.id}`, {
      title: task.title,
      description: task.description,
      assignee_id: task.assignee_id,
      due_date: task.due_date,
      done: !task.done,
      content_item_id: task.content_item_id,
      client_id: task.client_id,
    })
    if (res.data) {
      setTasks((prev) => prev.map((t) => (t.id === task.id ? res.data! : t)))
    }
  }

  async function handleDelete(taskId: string) {
    const res = await apiClient.delete(`/tasks/${taskId}`)
    if (res.error) {
      setError(res.error.message)
      return
    }
    setTasks((prev) => prev.filter((t) => t.id !== taskId))
  }

  const isOverdue = (task: Task) =>
    task.due_date && !task.done && task.due_date < new Date().toISOString().slice(0, 10)

  if (loading) {
    return <p className="text-gray-500 p-4">Loading tasks...</p>
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold text-gray-900">Tasks</h2>
        <Link
          to="/dashboard/tasks/new"
          className="rounded-lg bg-socialflow-600 px-4 py-2 text-sm font-medium text-white hover:bg-socialflow-700 transition-colors"
        >
          + New Task
        </Link>
      </div>

      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 mb-4">
          {error}
        </div>
      )}

      {tasks.length === 0 ? (
        <div className="rounded-lg border border-dashed border-gray-300 bg-white p-12 text-center">
          <p className="text-gray-500">No tasks yet. Create your first task to get started.</p>
        </div>
      ) : (
        <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="text-left px-4 py-3 font-medium text-gray-600">Done</th>
                <th className="text-left px-4 py-3 font-medium text-gray-600">Title</th>
                <th className="text-left px-4 py-3 font-medium text-gray-600">Due</th>
                <th className="text-left px-4 py-3 font-medium text-gray-600">Linked</th>
                <th className="text-right px-4 py-3 font-medium text-gray-600">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {tasks.map((task) => (
                <tr key={task.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3">
                    <input
                      type="checkbox"
                      checked={task.done}
                      onChange={() => handleToggle(task)}
                      className="h-4 w-4 rounded border-gray-300 text-socialflow-600 focus:ring-socialflow-500"
                    />
                  </td>
                  <td className="px-4 py-3">
                    <Link
                      to={`/dashboard/tasks/${task.id}/edit`}
                      className={`font-medium hover:text-socialflow-700 ${
                        task.done ? 'line-through text-gray-400' : 'text-gray-900'
                      }`}
                    >
                      {task.title}
                    </Link>
                    {task.description && (
                      <p className="text-xs text-gray-400 mt-0.5 truncate max-w-xs">{task.description}</p>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    {task.due_date ? (
                      <span
                        className={`text-xs ${isOverdue(task) ? 'text-red-600 font-medium' : 'text-gray-500'}`}
                      >
                        {task.due_date}
                      </span>
                    ) : (
                      <span className="text-xs text-gray-300">—</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-xs text-gray-500">
                    {task.content_item_id && (
                      <span className="mr-2">📝 Content</span>
                    )}
                    {task.client_id && (
                      <span>👤 Client</span>
                    )}
                    {!task.content_item_id && !task.client_id && '—'}
                  </td>
                  <td className="px-4 py-3 text-right space-x-2">
                    <Link
                      to={`/dashboard/tasks/${task.id}/edit`}
                      className="text-xs text-socialflow-600 hover:text-socialflow-700 font-medium"
                    >
                      Edit
                    </Link>
                    <button
                      onClick={() => handleDelete(task.id)}
                      className="text-xs text-red-500 hover:text-red-700 font-medium"
                    >
                      Delete
                    </button>
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
