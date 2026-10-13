import type { Rolldown } from "vite";

// A group names the packages whose code shares one chunk. Everything else is left to
// Rolldown, which puts each module in the chunk of exactly the routes that use it — so a
// library used a little on many pages arrives a little at a time, not all at once.
//
// Group a package only when one of these holds:
//   - every page needs it (the framework core), so it should be one stable, cached file;
//   - it is a large library that only ever loads behind a lazy boundary and must stay out
//     of the chunks the entry needs (editors, charts, PDF, maps).
//
// A group matches a package by its own `node_modules/<name>/` segment and nothing else.
// pnpm names a package's folder after its peers (`@base-ui+react@1.8.0_date-fns@4.4.0`),
// so a bare /date-fns/ also matched every Base UI module and moved all of Base UI into the
// date chunk the entry loads; an app file named route-prefetch.ts once matched /fetch/.

type Group = Rolldown.CodeSplittingGroup;

const escapeRegExp = (value: string) => value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

const SEPARATOR = "[\\\\/]";
const WITHIN_SEGMENT = "[^\\\\/]*";

/**
 * packageTest matches a module inside one of the named packages, by the path that follows a
 * node_modules folder. `*` stands for any run of characters within one path segment
 * (`@tiptap/*`, `prosemirror-*`), and a name may go on into a subpath
 * (`react-phone-number-input/flags`).
 */
export function packageTest(...names: string[]): RegExp {
  const alternatives = names.map((name) =>
    name
      .split("/")
      .map((part) => part.split("*").map(escapeRegExp).join(WITHIN_SEGMENT))
      .join(SEPARATOR),
  );
  return new RegExp(
    `${SEPARATOR}node_modules${SEPARATOR}(?:${alternatives.join("|")})(?:${SEPARATOR}|$)`,
  );
}

function group(name: string, ...packages: string[]): Group {
  return { name, test: packageTest(...packages) };
}

// `$initial` follows the module graph, where a barrel's re-export is an import: one icon in
// the shell made all of the icon set "initial", so every icon the app uses anywhere
// arrived with the first page, and the same went for every GraphQL document graphql.ts
// re-exports. These are left to automatic chunking, which places each module with the
// routes that use it.
const REACHED_THROUGH_BARRELS = [
  packageTest("@untitledui/icons"),
  /[\\/]packages[\\/]graphql[\\/]src[\\/]generated[\\/]/,
];

function initialGroup(name: string, matches: (id: string) => boolean): Group {
  return {
    name,
    test: (id) => matches(id) && !REACHED_THROUGH_BARRELS.some((pattern) => pattern.test(id)),
    tags: ["$initial"],
  };
}

const matching = (pattern: RegExp) => (id: string) => pattern.test(id);

export const chunkGroups: Group[] = [
  // Tiny styling helpers that virtually every module reaches through `cn()`. Without a
  // group of their own they get folded into whichever vendor chunk claims them first.
  group("class-utils", "clsx", "tailwind-merge", "class-variance-authority", "tailwind-variants"),
  // The framework core is on every page, so it must own a chunk; left ungrouped it is
  // absorbed by whatever group claims it first, and the entry then pulls that library in.
  group("react-vendor", "react", "react-dom", "react-is", "scheduler", "use-sync-external-store"),
  group("react-router", "react-router"),
  group("floating-ui", "@floating-ui/*"),
  group("store-utils", "zustand", "nuqs", "immer"),
  // TanStack Store is shared by Query's hotkeys and by Table. Claimed here first, it stays
  // out of the table chunk, which would otherwise be imported by every page for it.
  group(
    "tanstack-query",
    "@tanstack/query-core",
    "@tanstack/react-query",
    "@tanstack/hotkeys",
    "@tanstack/react-hotkeys",
    "@tanstack/store",
    "@tanstack/react-store",
    "@lukemorales/query-key-factory",
  ),
  // Table and virtualiser code only ever runs behind a lazy data table.
  group(
    "tanstack-table",
    "@tanstack/react-table",
    "@tanstack/table-core",
    "@tanstack/react-virtual",
    "@tanstack/virtual-core",
  ),
  group("motion", "motion", "motion-dom", "motion-utils", "framer-motion"),
  group("zod", "zod"),
  group("toast", "sonner"),
  // Shared by @codemirror/view and prosemirror-view. Ungrouped they land in the CodeMirror
  // chunk, which then makes opening the comment editor download all of CodeMirror as well.
  group("editor-dom-utils", "w3c-keyname", "style-mod", "crelt"),
  group("codemirror-view", "@codemirror/view"),
  group("codemirror-language", "@codemirror/language"),
  group("codemirror-autocomplete", "@codemirror/autocomplete"),
  group("tiptap", "@tiptap/*", "prosemirror-*"),
  group("dnd-kit", "@dnd-kit/*"),
  group("google-maps", "@vis.gl/react-google-maps"),
  group("nivo", "@nivo/*"),
  group("recharts", "recharts"),
  group("pdfjs", "pdfjs-dist"),
  // Must precede phone-utils: `react-phone-number-input/flags` is the barrel over the flag
  // components and is imported lazily, so it belongs with the flags rather than with the
  // phone metadata the input needs up front. Scoped to the React components on purpose —
  // react-phone-number-input reaches for country-flag-icons' tiny emoji helper
  // synchronously, and grouping that with the SVGs would pull them all into the entry.
  group("country-flags", "country-flag-icons/react", "react-phone-number-input/flags"),
  group("phone-utils", "react-phone-number-input", "libphonenumber-js"),
  // What the entry imports statically, it needs before first paint. Left to automatic
  // chunking that is a couple of hundred small shared chunks, each its own preload and each
  // compressed alone; here it is one chunk per source, split by how often each changes:
  // dependencies rarely, the shared package now and then, the app with every release.
  // (The generated GraphQL client has no group: its documents travel with their routes.) Do not give these a maxSize: cut by
  // size, the startup chunks evaluated out of order and the app failed to start with
  // "s is not a function".
  // Stylesheets block the first paint, each its own request: the startup ones are one file.
  initialGroup("initial-styles", matching(/\.css(?:$|\?)/)),
  initialGroup("initial-vendor", matching(/[\\/]node_modules[\\/]/)),
  initialGroup("initial-shared", matching(/[\\/]packages[\\/](?:shared|edition)[\\/]/)),
  // The HTML entry and Vite's virtual modules stay with the entry chunk.
  initialGroup("initial", (id) => !id.includes("\0") && !id.endsWith(".html")),
];
