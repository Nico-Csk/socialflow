import { BrowserRouter, Routes, Route, Navigate, Link, useLocation } from 'react-router-dom'
import { MeProvider, useMe } from '@/context/MeContext'
import Login from '@/pages/Login'
import Register from '@/pages/Register'
import WorkspaceSwitcher from '@/components/WorkspaceSwitcher'
import ClientList from '@/pages/Clients/ClientList'
import ClientForm from '@/pages/Clients/ClientForm'
import ContentList from '@/pages/ContentItems/ContentList'
import ContentForm from '@/pages/ContentItems/ContentForm'
import ContentDetail from '@/pages/ContentItems/ContentDetail'
import Dashboard from '@/pages/Dashboard/Dashboard'
import Calendar from '@/pages/Calendar/Calendar'
import TaskList from '@/pages/Tasks/TaskList'
import TaskForm from '@/pages/Tasks/TaskForm'

function Home() {
  return (
    <div className="min-h-screen bg-white flex flex-col items-center justify-center gap-6 px-4">
      <h1 className="text-4xl font-bold text-gray-900">SocialFlow</h1>
      <p className="text-lg text-gray-500 max-w-md text-center">
        Content workflow management for community managers.
      </p>
      <div className="flex gap-4">
        <a
          href="/login"
          className="rounded-lg bg-socialflow-600 px-6 py-3 text-white font-medium hover:bg-socialflow-700 transition-colors"
        >
          Log in
        </a>
        <a
          href="/register"
          className="rounded-lg border border-gray-300 px-6 py-3 text-gray-700 font-medium hover:bg-gray-50 transition-colors"
        >
          Register
        </a>
      </div>
    </div>
  )
}

// NAV_ITEMS paths are relative to /dashboard
const NAV_ITEMS: { label: string; path: string; icon: string; disabled?: boolean }[] = [
  { label: 'Dashboard', path: '/dashboard', icon: '📊' },
  { label: 'Content', path: '/dashboard/content-items', icon: '📝' },
  { label: 'Clients', path: '/dashboard/clients', icon: '👥' },
  { label: 'Calendar', path: '/dashboard/calendar', icon: '📅' },
  { label: 'Tasks', path: '/dashboard/tasks', icon: '✅' },
]

function DashboardShell() {
  const { user, loading, logout } = useMe()
  const location = useLocation()

  if (loading) {
    return (
      <div className="min-h-screen bg-white flex items-center justify-center">
        <p className="text-gray-500">Loading...</p>
      </div>
    )
  }

  if (!user) {
    return <Navigate to="/login" replace />
  }

  return (
    <div className="min-h-screen bg-gray-50 flex">
      {/* Sidebar */}
      <aside className="w-56 bg-white border-r border-gray-200 flex flex-col">
        <div className="px-4 py-4 border-b border-gray-100">
          <h1 className="text-lg font-bold text-gray-900">SocialFlow</h1>
          <div className="mt-2">
            <WorkspaceSwitcher />
          </div>
        </div>

        <nav className="flex-1 px-3 py-4 space-y-1">
          {NAV_ITEMS.map((item) => (
            <Link
              key={item.path}
              to={item.disabled ? '#' : item.path}
              className={`flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                item.disabled
                  ? 'text-gray-300 cursor-not-allowed'
                  : (item.path === '/dashboard'
                      ? location.pathname === '/dashboard'
                      : location.pathname.startsWith(item.path))
                    ? 'bg-socialflow-50 text-socialflow-700'
                    : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
              }`}
              onClick={(e) => { if (item.disabled) e.preventDefault() }}
            >
              <span className="text-base">{item.icon}</span>
              {item.label}
              {item.disabled && <span className="text-[10px] text-gray-300 ml-auto">soon</span>}
            </Link>
          ))}
        </nav>

        <div className="px-4 py-3 border-t border-gray-100">
          <p className="text-xs text-gray-400 truncate">{user.email}</p>
          <button
            onClick={logout}
            className="text-xs text-gray-500 hover:text-gray-700 transition-colors mt-1"
          >
            Log out
          </button>
        </div>
      </aside>

      {/* Main content */}
      <div className="flex-1 flex flex-col min-w-0">
        <main className="flex-1 p-6">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/clients" element={<ClientList />} />
            <Route path="/clients/new" element={<ClientForm />} />
            <Route path="/clients/:id/edit" element={<ClientForm />} />
            <Route path="/content-items" element={<ContentList />} />
            <Route path="/content-items/new" element={<ContentForm />} />
            <Route path="/content-items/:id" element={<ContentDetail />} />
            <Route path="/content-items/:id/edit" element={<ContentForm />} />
            <Route path="/calendar" element={<Calendar />} />
            <Route path="/tasks" element={<TaskList />} />
            <Route path="/tasks/new" element={<TaskForm />} />
            <Route path="/tasks/:id/edit" element={<TaskForm />} />
          </Routes>
        </main>
      </div>
    </div>
  )
}

function App() {
  return (
    <BrowserRouter>
      <MeProvider>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/login" element={<Login />} />
          <Route path="/register" element={<Register />} />
          <Route path="/dashboard/*" element={<DashboardShell />} />
        </Routes>
      </MeProvider>
    </BrowserRouter>
  )
}

export default App
