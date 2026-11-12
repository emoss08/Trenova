import { Activity, useState, type ReactNode } from "react";

/**
 * A view that keeps what a person did in it while another view is in front.
 *
 * It mounts the first time it is shown and is hidden, not unmounted, after:
 * its state, its scroll and what it has read are there when the person comes
 * back, while its effects rest and it costs nothing to lay out. A view never
 * shown is never mounted, so nothing is read for a place nobody opened.
 */
export function Kept({ shown, children }: { shown: boolean; children: ReactNode }) {
  const [seen, setSeen] = useState(shown);
  if (shown && !seen) {
    setSeen(true);
  }
  if (!seen) {
    return null;
  }

  return <Activity mode={shown ? "visible" : "hidden"}>{children}</Activity>;
}
