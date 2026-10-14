import { createContext, useContext, useId, useLayoutEffect, useState, type ReactNode } from "react";

type SetHot = (key: string | null) => void;

const SetHotContext = createContext<SetHot>(() => {});

/**
 * The rule that lights one series inside one provider: every mark carrying its key in
 * `data-series` gets `--an-hot: 1`, which analytics.css turns into full opacity, a stronger
 * chip border and an outlined dot while the rest of the page dims.
 */
export function highlightRule(scope: string, key: string): string {
  return `[data-hl="${CSS.escape(scope)}"] [data-series="${CSS.escape(key)}"]{--an-hot:1}`;
}

function canAdoptSheets(): boolean {
  return typeof CSSStyleSheet !== "undefined" && "adoptedStyleSheets" in document;
}

/**
 * Keeps the highlight rule in a constructed stylesheet adopted by the document: changing the
 * highlighted series rewrites one rule instead of re-rendering every card, and a constructed
 * sheet is not an inline style a content security policy can refuse.
 */
function useHighlightSheet(scope: string, hot: string | null) {
  const [sheet] = useState(() => (canAdoptSheets() ? new CSSStyleSheet() : null));

  useLayoutEffect(() => {
    if (!sheet) {
      return;
    }
    document.adoptedStyleSheets = [...document.adoptedStyleSheets, sheet];
    return () => {
      document.adoptedStyleSheets = document.adoptedStyleSheets.filter((item) => item !== sheet);
    };
  }, [sheet]);

  useLayoutEffect(() => {
    sheet?.replaceSync(hot === null ? "" : highlightRule(scope, hot));
  }, [hot, scope, sheet]);
}

/**
 * One series highlighted at a time, shared by every chart on the page. Only this element
 * re-renders when it changes; the marks read it from CSS.
 */
export function HighlightProvider({ children }: { children: ReactNode }) {
  const scope = useId();
  const [hot, setHot] = useState<string | null>(null);
  useHighlightSheet(scope, hot);

  return (
    <SetHotContext value={setHot}>
      <div className="an-hl" data-hl={scope} data-hot={hot === null ? undefined : ""}>
        {children}
      </div>
    </SetHotContext>
  );
}

export function useSetHot(): SetHot {
  return useContext(SetHotContext);
}

/** Hover handlers that highlight a series while the pointer or focus is on it. */
export function hoverProps(setHot: SetHot, key: string) {
  return {
    onMouseEnter: () => setHot(key),
    onMouseLeave: () => setHot(null),
    onFocus: () => setHot(key),
    onBlur: () => setHot(null),
  };
}
