#!/usr/bin/env node
/**
 * Fails the build on a loading state that does not look like what loads.
 *
 * A page's first paint is its Suspense fallback: the chunk is still on the
 * wire, or a suspense query has not answered. `DataTableLazyComponent` draws a
 * table skeleton by default, and it was wrapped around dashboards, consoles,
 * calendars, rosters and every tab of AI control, so each of them opened as a
 * grid of grey rows and then jumped into a layout that shared nothing with it.
 *
 * Four rules hold the line:
 *
 *   table-skeleton-on-non-table  Everything inside `<DataTableLazyComponent>`
 *       must render a `DataTable`. Anything else goes in
 *       `<LazyComponent fallback={<ItsOwnSkeleton />}>`, whose fallback is
 *       required.
 *   unresolved-lazy-child  A child of `<DataTableLazyComponent>` the check
 *       could not follow to its module. Import it, or lazy-load it with
 *       `lazy(() => import(...))` or `lazyComponent(...)`, so it can.
 *   suspense-without-fallback  A bare `<Suspense>` draws nothing and says
 *       nothing about it. Name the skeleton, or write `fallback={null}` where
 *       nothing on screen is the right answer (a dialog that is not open yet).
 *   spinner-fallback  A spinner as a boundary's fallback holds the place of
 *       content it looks nothing like.
 *
 * A genuine exception takes `loading-skeleton-ignore: <reason>` in a comment on
 * the line or the line above.
 *
 * Usage: node scripts/check-loading-skeletons.mjs [--format=github]
 */

import { auditLoadingSkeletons } from "./loading-skeletons.mjs";

const ROOT = new URL("..", import.meta.url).pathname;
const GITHUB = process.argv.includes("--format=github");

const { violations, files } = await auditLoadingSkeletons(ROOT);

if (violations.length === 0) {
  console.log(`loading skeletons: clean (${files} files)`);
  process.exit(0);
}

const byRule = new Map();
for (const violation of violations) {
  byRule.set(violation.rule, (byRule.get(violation.rule) ?? 0) + 1);
  if (GITHUB) {
    console.log(
      `::error file=client/${violation.file},line=${violation.line},col=${violation.column}::${violation.rule}: ${violation.message}`,
    );
  } else {
    console.log(
      `${violation.file}:${violation.line}:${violation.column}\n  ${violation.rule}: ${violation.message}\n`,
    );
  }
}

console.log(`\nloading skeletons: ${violations.length} violation(s) in ${files} files`);
for (const [id, count] of [...byRule].sort((a, b) => b[1] - a[1])) {
  console.log(`  ${id.padEnd(28)} ${count}`);
}
console.log(
  "\nA page's first paint is its fallback; it should be the page's own layout. See\n" +
    '"Loading states" in docs/engineering/design-system.md. For a genuine exception, put\n' +
    "`loading-skeleton-ignore: <reason>` in a comment on or above the line.",
);
process.exit(1);
