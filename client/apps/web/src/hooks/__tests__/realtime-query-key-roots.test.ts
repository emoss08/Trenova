import { queries } from "@/lib/queries";
import {
  RESOURCE_QUERY_KEY_MAP,
  queryKeyPrefix,
  type QueryKeyRoot,
} from "@trenova/shared/hooks/realtime-patching";
import { describe, expect, it } from "vitest";

/*
A realtime invalidation that names the wrong key fails in silence. The event
arrives, invalidateQueries runs, it matches nothing, and the screen does not
move — so the only symptom is a stale list, which reads as the server not
having sent anything.

Five resources were addressing rows that way. `createQueryKeys("assistant", {
proposals })` does not cache under the root it is handed: it prepends its scope
and the method name, so a component reading `queries.assistant.proposals(id)`
reads ["assistant", "proposals", "assistant-proposals", id]. The map named
"assistant-proposals", TanStack matched from the start of the key, and an
approval made anywhere else never reached an open thread.

The map lives in @trenova/shared, which cannot see the app's query factories,
so its factory-backed entries are literals. These tests are what keeps the
literals true: one reads the factories and rejects a root that can only miss,
the other pins every array root to a prefix a factory actually produces.
*/

/** Every scope the app's factories own, and every inner key declared under one. */
function factoryKeys() {
  const scopes = new Set<string>();
  const declared = new Map<string, string>();

  for (const [scope, group] of Object.entries(queries as Record<string, unknown>)) {
    scopes.add(scope);
    if (typeof group !== "object" || group === null) {
      continue;
    }

    for (const [method, entry] of Object.entries(group as Record<string, unknown>)) {
      if (method === "_def") {
        continue;
      }
      const key = builtKey(entry);
      // [scope, method, ...declared]: the third element is the root the
      // factory was handed, and the one a reader is tempted to write out.
      if (key && typeof key[2] === "string") {
        declared.set(key[2], `${scope}.${method}`);
      }
    }
  }

  return { scopes, declared };
}

/** A factory entry is either an object or a function of its dynamic arguments. */
function builtKey(entry: unknown): readonly unknown[] | null {
  const candidates: unknown[][] = [[], ["id"], ["id", "second"], [true], [true, false]];
  const built =
    typeof entry === "function"
      ? candidates.reduce<unknown>((found, args) => {
          if (found) return found;
          try {
            return (entry as (...a: unknown[]) => unknown)(...args);
          } catch {
            return null;
          }
        }, null)
      : entry;

  const key = (built as { queryKey?: unknown } | null)?.queryKey;

  return Array.isArray(key) ? key : null;
}

function describeRoot(resource: string, root: QueryKeyRoot): string {
  return `${resource}: ${JSON.stringify(root)}`;
}

describe("RESOURCE_QUERY_KEY_MAP addresses keys the app actually caches under", () => {
  it("never names a factory's inner key, which matches nothing on its own", () => {
    const { scopes, declared } = factoryKeys();
    const unreachable: string[] = [];

    for (const [resource, roots] of Object.entries(RESOURCE_QUERY_KEY_MAP)) {
      for (const root of roots) {
        if (typeof root !== "string" || scopes.has(root)) {
          continue;
        }
        const owner = declared.get(root);
        if (owner) {
          unreachable.push(`${describeRoot(resource, root)} is ${owner}'s inner key`);
        }
      }
    }

    expect(unreachable).toEqual([]);
  });

  it("spells every prefix the way the factory that owns it does", () => {
    const prefixes = new Set<string>();
    for (const group of Object.values(queries as Record<string, unknown>)) {
      for (const entry of Object.values((group ?? {}) as Record<string, unknown>)) {
        const def = (entry as { _def?: unknown })?._def;
        if (Array.isArray(def)) {
          prefixes.add(def.join("\u0000"));
        }
      }
    }

    const drifted: string[] = [];
    for (const [resource, roots] of Object.entries(RESOURCE_QUERY_KEY_MAP)) {
      for (const root of roots) {
        if (typeof root === "string") {
          continue;
        }
        if (!prefixes.has(queryKeyPrefix(root).join("\u0000"))) {
          drifted.push(`${describeRoot(resource, root)} is not any factory's prefix`);
        }
      }
    }

    expect(drifted).toEqual([]);
  });
});
