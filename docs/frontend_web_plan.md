# React + Vite Frontend Plan

This document tracks the implementation plan for the React + Vite web frontend and supporting backend work. It will be referenced and updated as tasks are completed.

## 1. Project scaffolding & tooling
- [x] Initialize React + Vite app under `internal/web/ui` with TypeScript.
- [x] Configure linting/formatting (ESLint, Prettier) and styling approach (e.g., Tailwind or CSS modules).
- [x] Set up build pipeline to emit assets consumed by the Go server (copy artifacts into `internal/web/static/build` or equivalent) and update the server to serve the bundle.

## 2. API design & backend updates
- [x] Define REST endpoints (metadata + configuration CRUD + scenario run implemented):
  - `/api/meta` for reference data (states, strategies, enums).
  - `/api/configurations` for CRUD of saved configurations.
  - `/api/scenarios/run` returning the full `ScenarioComparison` payload.
- [x] Implement persistence layer (JSON files or SQLite) plus validation/error handling.
- [x] Ensure responses expose projections, impact analysis, assumptions for frontend consumption.

## 3. Frontend architecture
- [x] Establish state management (React Context/Zustand) mirroring backend data structures.
- [x] Create API service layer with loading/error states.
- [ ] Implement routing or tabbed layout separating Inputs and Results views.

## 4. Input workflow
- [ ] Build multi-step form covering:
  1. Personal Details (Person A/B)
  2. Earnings & Benefits
  3. TSP & Savings
  4. Social Security
  5. Global Assumptions
  6. Scenario Builder (multiple scenarios)
- [ ] Provide validation, helper text, and ability to add/remove scenarios.
- [ ] Enable save/load of configurations from the UI.

## 5. Results & reporting
- [ ] Create dashboard with summary cards, comparison table, and charts (income timeline, TSP balance) using a charting library.
- [ ] Render detailed projection tables with filters/export options plus narrative sections using `ImpactAnalysis`/`LongTermAnalysis`.
- [ ] Handle loading, error, and empty states gracefully.

## 6. Persistence & UX polish
- [ ] Add configuration drawer for saved profiles, import/export JSON, and autosave of the last session.
- [ ] Ensure responsive design, accessibility compliance, and cohesive theming.
- [ ] Update README/docs with setup instructions for the new UI + backend endpoints.
