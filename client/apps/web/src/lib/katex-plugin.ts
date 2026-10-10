import type { Options as MarkdownOptions } from "react-markdown";
import { useEffect, useSyncExternalStore } from "react";

type RehypePlugin = NonNullable<MarkdownOptions["rehypePlugins"]>[number];

let plugin: RehypePlugin | null = null;
let loading: Promise<RehypePlugin> | null = null;
const listeners = new Set<() => void>();

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function current(): RehypePlugin | null {
  return plugin;
}

/**
 * Reads the math typesetter and its stylesheet, once for the whole tab.
 * KaTeX is most of the weight of a reply's renderer and almost no reply in a
 * TMS has math in it, so it is fetched the first time one does. A failed read
 * is forgotten, so the next reply with math asks again.
 */
export function loadKatexPlugin(): Promise<RehypePlugin> {
  loading ??= Promise.all([import("rehype-katex"), import("katex/dist/katex.min.css")])
    .then(([module]) => {
      plugin = module.default;
      for (const listener of listeners) listener();
      return module.default;
    })
    .catch((error: unknown) => {
      loading = null;
      throw error;
    });

  return loading;
}

/**
 * The math typesetter for a reply that needs it: null until it has been read,
 * and asked for only when `needed`. Every reply on screen re-renders once when
 * it arrives; until then their math shows as the text it was written in.
 */
export function useKatexPlugin(needed: boolean): RehypePlugin | null {
  const loaded = useSyncExternalStore(subscribe, current, current);

  useEffect(() => {
    if (needed && loaded === null) {
      loadKatexPlugin().catch(() => undefined);
    }
  }, [loaded, needed]);

  return loaded;
}
