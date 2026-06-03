import { useContext } from 'react'
import MeContext from './MeContext'

export function useMe() {
  const ctx = useContext(MeContext)
  if (!ctx) {
    throw new Error('useMe must be used within a MeProvider')
  }
  return ctx
}
