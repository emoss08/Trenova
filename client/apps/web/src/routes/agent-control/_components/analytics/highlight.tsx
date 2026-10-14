import { createContext, useContext, useMemo, useState, type ReactNode } from "react";

type Highlight = {
  hot: string | null;
  setHot: (key: string | null) => void;
};

const HighlightContext = createContext<Highlight>({ hot: null, setHot: () => {} });

/** One series highlighted at a time, shared by every chart on the page. */
export function HighlightProvider({ children }: { children: ReactNode }) {
  const [hot, setHot] = useState<string | null>(null);
  const value = useMemo(() => ({ hot, setHot }), [hot]);

  return <HighlightContext.Provider value={value}>{children}</HighlightContext.Provider>;
}

export function useHighlight(): Highlight {
  return useContext(HighlightContext);
}

/** "is-hot" for the highlighted series, "is-dim" for the rest, nothing when none is. */
export function highlightClass(hot: string | null, key: string): string | undefined {
  if (hot === null) {
    return undefined;
  }

  return hot === key ? "is-hot" : "is-dim";
}

/** Hover handlers that highlight a series while the pointer or focus is on it. */
export function hoverProps(setHot: (key: string | null) => void, key: string) {
  return {
    onMouseEnter: () => setHot(key),
    onMouseLeave: () => setHot(null),
    onFocus: () => setHot(key),
    onBlur: () => setHot(null),
  };
}
