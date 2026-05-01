# ferscalc3 Project Memory

## Project Overview
FERS (Federal Employees Retirement System) retirement planning calculator.
- **Go reference engine**: `<code-root>/ferscalc3` — production-quality CLI + web server, DO NOT delete
- **Web app (active)**: `<code-root>/ferscalc-web` — SvelteKit 2 + Svelte 5 + Tailwind CSS v4, hosted at www.ferscalc.com on Netlify

## Migration Strategy
Rewriting the Go calculation engine in TypeScript for a fully client-side SvelteKit web app.
- Go code serves as **calculation oracle** — run both during development to verify parity
- All calculations run **client-side** (no API server, no hosting cost)
- Financial precision via `decimal.js` (mirrors Go's `shopspring/decimal`)
- Monte Carlo will use **Web Workers** for non-blocking UI

## ferscalc-web Stack
- SvelteKit 2.53.4, Svelte 5.53.9, TypeScript 5
- Tailwind CSS v4 (via `@tailwindcss/vite` plugin — no postcss.config.js needed)
- `@sveltejs/adapter-netlify` v4 — build output in `build/`, netlify functions in `.netlify/`
- Vitest for testing
- `decimal.js` for financial precision
- **Vite import**: `sveltekit` comes from `@sveltejs/kit/vite`, NOT `@sveltejs/vite-plugin-svelte`

## ferscalc-web Routes
- `/` — Landing/marketing page (src/routes/+page.svelte) with Netlify notify form
- `/calculator` — Main calculator app (Phase 2, stub in place)

## Build & Deploy
- `npm run build` → generates `build/` (static) + `.netlify/` (functions)
- `netlify.toml` sets `publish = "build"`, `NODE_VERSION = "22"`
- `.svelte-kit/` and `.netlify/` are gitignored

## Calculation Engine Port (Phase 1 — next)
Port Go modules to `src/lib/calc/`:
- `types.ts` — domain types (Employee, CashFlow, etc.)
- `fers.ts` — pension math (1.0%/1.1% multipliers, COLA, MRA+10, survivor)
- `tsp.ts` — TSP withdrawal strategies (4%, need-based, floor/ceiling, annuity, RMD)
- `social-security.ts` — SS benefit calculations
- `taxes.ts` — federal brackets, PA state/local, IRMAA, IRS Simplified Method
- `medicare.ts` — Medicare Part B + IRMAA
- `projection.ts` — annual cash flow loop
- `monte-carlo.ts` — Web Worker-based MC simulation

## Key Domain Rules (from Go reference)
- FERS multiplier: 1.1% if age 62+ AND 20+ years service; else 1.0%
- Sick leave: 2,087 hours = 1 year service credit (5 CFR 630.301)
- FERS COLA: CPI≤2%→full; 2-3%→capped 2%; >3%→CPI minus 1%
- WEP/GPO: repealed 2025 — no SS reduction for federal employees
- SECURE 2.0 RMD ages: 73 (born 1951-1959), 75 (born 1960+)
- IRS Simplified Method: monthly exclusion = employee_contributions / expected_payments
- Medicare IRMAA: 2-year lag on MAGI; base premium 2025 = $185/month
- MRA: typically 55-57 depending on birth year

## User Preferences
- Concise communication, no emojis
- Go code is the source of truth for all calculations during port
