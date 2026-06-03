# SocialFlow Frontend

React 19 + TypeScript + Vite + Tailwind CSS 4 frontend for the SocialFlow monorepo. Uses shadcn/ui conventions (`cn()` utility configured, component library available on demand).

## Scripts

| Script | Command | Purpose |
|--------|---------|---------|
| `dev` | `vite` | Start Vite dev server with HMR |
| `build` | `tsc -b && vite build` | Type-check then production build |
| `lint` | `eslint .` | Run ESLint across all files |
| `preview` | `vite preview` | Preview production build locally |
| `test` | `vitest run` | Run tests once (no watch) |
| `test:watch` | `vitest` | Run tests in watch mode |
| `test:coverage` | `vitest run --coverage` | Run tests with coverage report |

Run scripts from the `web/` directory with `npm run <script>`.

## Testing

**Stack**: [Vitest](https://vitest.dev) + [Testing Library](https://testing-library.com) on `jsdom` environment.

**Setup** (`src/test/setup.ts`):
- `@testing-library/jest-dom/vitest` — DOM-specific matchers (`toBeInTheDocument`, `toHaveTextContent`, etc.)
- `globals: true` — test globals (`describe`, `it`, `expect`, `vi`) available without imports
- Automatic cleanup after each test (`cleanup()` from Testing Library + `vi.clearAllMocks()`)

**Rules**:
- Tests SHALL NOT change existing behavior without explicit intent. If refactoring, preserve what the tests verify unless the spec has changed.

## Coverage

```bash
npm run test:coverage
```

Generates reports in `./coverage/`:
- `text` — terminal summary
- `json` — machine-readable
- `html` — open `./coverage/index.html` in a browser for an interactive report

Coverage is collected from `src/**/*.{ts,tsx}` (v8 provider), excluding test files, type declarations, and `src/main.tsx`.

## TypeScript

Key compiler options in `tsconfig.app.json`:

| Option | Value | Effect |
|--------|-------|--------|
| `verbatimModuleSyntax` | `true` | Imports must use `import type` for type-only imports |
| `paths` | `{ "@/*": ["./src/*"] }` | `@/` resolves to `src/` |
| `noUnusedLocals` | `true` | Error on unused local variables |
| `noUnusedParameters` | `true` | Error on unused function parameters |

The `@/` alias is also configured in `vite.config.ts` and `vitest.config.ts` for build-time and test-time resolution.

## ESLint

Flat config (`eslint.config.js`) with:

| Plugin | Purpose |
|--------|---------|
| `@eslint/js` | Core ESLint recommended rules |
| `typescript-eslint` | TypeScript-aware lint rules |
| `eslint-plugin-react-hooks` | Rules of Hooks enforcement |
| `eslint-plugin-react-refresh` | Fast Refresh compatibility checks |

Ignores: `dist/`, `coverage/` directories.

Run: `npm run lint`

## Related Docs

- [`../README.md`](../README.md) — project architecture, API reference, database schema, and backend conventions
