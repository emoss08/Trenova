import { useEffect, useState } from "react";

/** Longer than a row's entrance (`dk-w-rise`), after which rows scrolled into view mount still. */
const ENTRANCE_MS = 480;

/** On the list once its rows have risen: rows that mount after it mount still. */
// i18n-ignore: class names, never shown to a person
export const LIST_SETTLED_CLASS = "dk-w-nq-settled";

/**
 * Whether a list's rows have finished rising in. Rows marked `dk-w-ar` rise
 * when a view first draws them, and again each time the view comes back in
 * front, so every arrival at a view reads the same; a row that scrolls into
 * view afterwards mounts still. `view` names what the list shows, and another
 * view is another arrival; `shown` is whether the list is in front, for a
 * view that is kept, hidden, while another is.
 */
export function useListEntrance(view: string, hasRows: boolean, shown = true): boolean {
  const [settledView, setSettledView] = useState<string | null>(null);

  // Coming back in front is an arrival: the rows rise again.
  const [wasShown, setWasShown] = useState(shown);
  if (shown !== wasShown) {
    setWasShown(shown);
    if (shown) {
      setSettledView(null);
    }
  }

  useEffect(() => {
    if (!hasRows || !shown) {
      return;
    }
    const timer = window.setTimeout(() => setSettledView(view), ENTRANCE_MS);
    return () => window.clearTimeout(timer);
  }, [view, hasRows, shown]);

  return settledView === view;
}
