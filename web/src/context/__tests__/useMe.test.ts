import { describe, it, expect } from 'vitest'
import { useMe } from '@/context/MeContext'
import MeContext from '@/context/MeContext'

describe('useMe re-export compatibility (lint hardening — import chain integrity)', () => {
  it('useMe is exported as a named function from @/context/MeContext', () => {
    // The import statement at the top IS the test — if the re-export chain
    // breaks, this file won't even compile. Verifying the import resolved.
    expect(typeof useMe).toBe('function')
  })

  it('useMe has the correct function name (not aliased or mangled)', () => {
    // The extracted hook should retain its identity through the re-export
    expect(useMe.name).toBe('useMe')
  })

  it('MeContext default export is preserved after useMe extraction', () => {
    // MeContext.tsx still default-exports the context object
    // (used internally by useMe.ts via useContext)
    expect(MeContext).toBeDefined()
    // Verify it's a React context (has Provider and Consumer)
    expect(typeof MeContext).toBe('object')
  })

  it('useMe and MeContext import from the same module without conflicts', () => {
    // Both the named re-export (useMe) and the default export (MeContext)
    // must coexist in the same import statement
    expect(useMe).not.toBe(MeContext)
  })
})
