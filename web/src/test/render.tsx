import { render, type RenderOptions } from '@testing-library/react'
import { MemoryRouter, type InitialEntry } from 'react-router-dom'
import { type ReactElement } from 'react'

interface RenderWithRouterOptions extends Omit<RenderOptions, 'wrapper'> {
  route?: string
  initialEntries?: InitialEntry[]
}

export function renderWithRouter(
  ui: ReactElement,
  { route, initialEntries, ...renderOptions }: RenderWithRouterOptions = {},
) {
  const entries = initialEntries ?? [route ?? '/']
  return render(ui, {
    wrapper: ({ children }) => (
      <MemoryRouter initialEntries={entries}>{children}</MemoryRouter>
    ),
    ...renderOptions,
  })
}
