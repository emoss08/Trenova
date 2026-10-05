/**
 * Packages that must exist exactly once in the client bundle.
 *
 * Each holds React context or module-level state. The apps alias
 * @trenova/shared to its source, so shared components import these from
 * packages/shared; a second copy there has its own context, and a component
 * from one copy rendered inside a provider from the other fails (a shared
 * DialogTitle inside a web Dialog.Root throws Base UI error #27, a shared
 * router hook reports it is outside a router).
 *
 * Three guards use this list:
 * - pnpm-workspace.yaml's catalog gives every workspace package the same
 *   version (reference them as "catalog:").
 * - Each app's vite.config.ts dedupes them, so a build bundles one copy.
 * - scripts/check-single-copies.ts fails CI when the lockfile resolves any of
 *   them to more than one copy.
 */
export const singletonPackages = [
  "react",
  "react-dom",
  "@base-ui/react",
  "@tanstack/react-query",
  "@tanstack/query-core",
  "@tanstack/react-table",
  "react-router",
  "react-hook-form",
  "react-error-boundary",
  "sonner",
  "zustand",
] as const;
