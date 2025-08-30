# Repository Guidelines

## Project Structure & Module Organization
- App entry: `src/main.tsx`; root layout/pages in `src/app`.
- Routing: `src/routing`; UI components: `src/components` (PascalCase files).
- State: `src/stores`; data/services: `src/services`; utilities: `src/lib`.
- Styles: `src/styles`; types: `src/types`; constants/config: `src/{constants,config}`.
- Assets: `src/assets`; public static files: `public/`. Build output: `dist/`.
- Import alias: use `@/` for `src/` (e.g., `import { Button } from "@/components/Button"`).

## Build, Test, and Development Commands
- Install: `pnpm i` — installs dependencies (pnpm is required).
- Develop: `pnpm dev` — starts Vite dev server.
- Lint: `pnpm lint` — runs ESLint with Prettier integration.
- Build: `pnpm build` — type-checks (`tsc -b`) and builds with Vite.
- Preview: `pnpm preview` — serves the production build locally.

## Coding Style & Naming Conventions
- Language: React + TypeScript (TS 5). Two-space indentation.
- Components: PascalCase files in `src/components` (e.g., `UserCard.tsx`).
- Hooks: `useX` naming in `src/hooks` (e.g., `useDebounce.ts`).
- Utilities/Stores/Services: kebab- or camel-case (e.g., `date-utils.ts`, `authStore.ts`).
- Linting/Formatting: ESLint + Prettier (`eslint.config.mjs`); React Hooks rules enabled. Tailwind CSS v4 is used; a basic Stylelint config exists.

## Testing Guidelines
- This package does not include a test runner yet. When adding one (e.g., Vitest), place tests in `src/__tests__` or alongside files as `*.test.ts(x)`.
- Aim to cover core hooks, utilities, and store logic first. Prefer component tests that focus on behavior.

## Commit & Pull Request Guidelines
- Commits follow emoji + scope style: `🪛(fix): concise imperative message` or `🔥(feat): add shipment holds (#420)`.
- Keep messages short; reference issues/PRs when applicable.
- PRs: include a clear description, linked issues, screenshots/gifs for UI changes, and a brief testing summary. Ensure `pnpm lint` and `pnpm build` pass.

## Security & Configuration Tips
- Environment: create a local `.env` from team-provided values; never commit secrets.
- Private registries are configured in `.npmrc`. Use `pnpm` to respect workspace settings.
