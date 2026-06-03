import { useState, type FormEvent } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import apiClient from '@/lib/apiClient'
import { useMe } from '@/context/MeContext'

export default function Login() {
  const navigate = useNavigate()
  const { reauthenticate } = useMe()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)

    const res = await apiClient.post<{ id: string; email: string; name: string }>(
      '/auth/login',
      { email, password },
    )

    if (res.error) {
      setLoading(false)
      setError(res.error.message)
      return
    }

    // Reauthenticate to sync MeContext with the new session
    const ok = await reauthenticate()

    setLoading(false)

    if (!ok) {
      setError('Login succeeded but session could not be verified. Please try again.')
      return
    }

    navigate('/dashboard')
  }

  return (
    <div className="min-h-screen bg-white flex items-center justify-center px-4">
      <div className="w-full max-w-sm">
        <h1 className="text-2xl font-bold text-gray-900 mb-2">Log in to SocialFlow</h1>
        <p className="text-sm text-gray-500 mb-8">
          Enter your credentials to access your workspaces.
        </p>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          {error && (
            <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
              {error}
            </div>
          )}

          <label className="flex flex-col gap-1">
            <span className="text-sm font-medium text-gray-700">Email</span>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              autoFocus
              className="rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-socialflow-500 focus:outline-none focus:ring-1 focus:ring-socialflow-500"
              placeholder="you@example.com"
            />
          </label>

          <label className="flex flex-col gap-1">
            <span className="text-sm font-medium text-gray-700">Password</span>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              className="rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-socialflow-500 focus:outline-none focus:ring-1 focus:ring-socialflow-500"
              placeholder="Your password"
            />
          </label>

          <button
            type="submit"
            disabled={loading}
            className="rounded-lg bg-socialflow-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-socialflow-700 transition-colors disabled:opacity-50"
          >
            {loading ? 'Logging in...' : 'Log in'}
          </button>
        </form>

        <p className="mt-6 text-center text-sm text-gray-500">
          Don&apos;t have an account?{' '}
          <Link to="/register" className="text-socialflow-600 hover:text-socialflow-700 font-medium">
            Register
          </Link>
        </p>
      </div>
    </div>
  )
}
