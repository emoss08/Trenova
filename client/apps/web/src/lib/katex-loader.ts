import type Katex from "katex";
import { useEffect, useSyncExternalStore } from "react";

type Typesetter = typeof Katex;

let typesetter: Typesetter | null = null;
let loading: Promise<Typesetter> | null = null;
const listeners = new Set<() => void>();

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function current(): Typesetter | null {
  return typesetter;
}

/**
 * Reads the math typesetter and its stylesheet, once for the whole tab.
 * KaTeX is most of the weight of a reply's renderer and almost no reply in a
 * TMS has math in it, so it is fetched the first time a formula is drawn. A
 * failed read is forgotten, so the next formula asks again.
 */
export function loadKatex(): Promise<Typesetter> {
  loading ??= Promise.all([import("katex"), import("katex/dist/katex.min.css")])
    .then(([module]) => {
      typesetter = module.default;
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
 * The math typesetter, for a formula about to be drawn: null until it has
 * been read, and asked for the first time a formula mounts. Every formula on
 * screen re-renders once when it arrives; until then each shows the text it
 * was written in.
 */
export function useKatex(): Typesetter | null {
  const loaded = useSyncExternalStore(subscribe, current, current);

  useEffect(() => {
    if (loaded === null) {
      loadKatex().catch(() => undefined);
    }
  }, [loaded]);

  return loaded;
}
