import { notification } from "@trenova/shared/lib/queries/notification";
import { describe, expect, it } from "vitest";
import { queries } from "..";
import { ar } from "../ar";
import { assistant } from "../assistant";
import { attention } from "../attention";
import { homeLayout } from "../home-layout";
import { insight } from "../insight";
import { integration } from "../integration";
import { organization } from "../organization";
import { pageFavorite } from "../page-favorite";
import { reports } from "../reports";
import { shipment } from "../shipment";
import { shipmentBoard } from "../shipment-board";
import { sidebarPreferences } from "../sidebar-preferences";
import { telematics } from "../telematics";
import { user, userOrganization } from "../user";

// The app shell, the command palette (preloaded on every page) and the Home page import these
// factories from their own modules rather than through the `queries` object, which would
// bring every other factory, service and GraphQL document with it into their download.
// mergeQueryKeys hands back copies, not the same objects, so what must hold is that both build
// the very same keys: the cache matches a query by its key, and a query must keep meeting the
// page that invalidates it.
const DIRECT = {
  ar,
  assistant,
  attention,
  homeLayout,
  insight,
  integration,
  notification,
  organization,
  pageFavorite,
  reports,
  shipment,
  shipmentBoard,
  sidebarPreferences,
  telematics,
  user,
  userOrganization,
} as const;

// Key builders take ids (strings) or option objects; each is called with both kinds, and when
// a kind does not suit it, the error it throws is compared instead — the same builder fails
// the same way.
const SAMPLE_ARGS: unknown[][] = [
  ["arg-1", "arg-2", "arg-3"],
  [{}, {}, {}],
];

type KeyEntry = { _def?: unknown; queryKey?: unknown };
type KeyBuilder = (...args: unknown[]) => KeyEntry;

function keysOf(entry: unknown): { def: unknown; built: unknown[] } {
  const def = (entry as KeyEntry)._def;
  if (typeof entry !== "function") return { def, built: [(entry as KeyEntry).queryKey] };
  return {
    def,
    built: SAMPLE_ARGS.map((args) => {
      try {
        return (entry as KeyBuilder)(...args).queryKey;
      } catch (error) {
        return `throws: ${error instanceof Error ? error.message : String(error)}`;
      }
    }),
  };
}

describe("factories the shell imports directly", () => {
  it.each(Object.keys(DIRECT) as (keyof typeof DIRECT)[])(
    "%s builds exactly the keys the queries object builds",
    (name) => {
      const direct = DIRECT[name] as unknown as Record<string, unknown>;
      const merged = queries[name] as unknown as Record<string, unknown>;

      expect(Object.keys(direct).sort()).toEqual(Object.keys(merged).sort());
      for (const key of Object.keys(direct)) {
        const built = keysOf(direct[key]);
        expect(built, `${name}.${key}`).toEqual(keysOf(merged[key]));
        const madeAKey = built.built.some(
          (result) => !(typeof result === "string" && result.startsWith("throws:")),
        );
        expect(madeAKey, `${name}.${key} built no key from either kind of argument`).toBe(true);
      }
    },
  );

  it("roots each factory's keys at its own name", () => {
    expect(pageFavorite.all().queryKey[0]).toBe("pageFavorite");
    expect(sidebarPreferences.effective().queryKey[0]).toBe("sidebarPreferences");
    expect(attention.summary().queryKey[0]).toBe("attention");
  });
});
