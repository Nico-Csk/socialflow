import { describe, it, expect } from 'vitest'
import { cn } from '@/lib/utils'

describe('cn (classname merge utility)', () => {
  it('merges multiple class strings into one', () => {
    const result = cn('px-4', 'py-2', 'bg-red-500')
    expect(result).toBe('px-4 py-2 bg-red-500')
  })

  it('filters out falsy values (false, undefined, null)', () => {
    const isHidden = false
    const result = cn('base', isHidden && 'hidden', undefined, null, 'active')
    expect(result).toBe('base active')
  })

  it('resolves conflicting Tailwind classes via tailwind-merge', () => {
    // px-4 followed by px-2 → tailwind-merge keeps the last one (px-2 wins)
    const result = cn('px-4', 'px-2')
    expect(result).toBe('px-2')
  })

  it('removes duplicate identical classes', () => {
    const result = cn('px-4', 'px-4')
    expect(result).toBe('px-4')
  })

  it('flattens nested arrays of classes', () => {
    const result = cn(['flex', 'gap-2'], 'mt-4', ['text-sm', 'font-medium'])
    expect(result).toBe('flex gap-2 mt-4 text-sm font-medium')
  })

  it('returns an empty string for no inputs', () => {
    const result = cn()
    expect(result).toBe('')
  })
})
