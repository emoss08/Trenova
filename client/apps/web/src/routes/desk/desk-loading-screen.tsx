import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import "./_styles/desk-v2.css";

/**
 * The Desk while it is first opened: the room's shape — the rail, the strip
 * across the top and the box in the middle — in place of the app-wide
 * loading card, so nothing jumps when the Desk takes over.
 */
export function DeskLoadingScreen() {
  return (
    <main data-slot="desk-loading-screen" className="dsk" aria-busy>
      <aside className="dk-sb" aria-hidden>
        <div className="dk-sb-top">
          <Skeleton className="dk-sk-logo" />
        </div>
        <div className="dk-sb-list">
          <Skeleton className="dk-sk-row" />
          <Skeleton className="dk-sk-row" />
          <Skeleton className="dk-sk-row" />
          <Skeleton className="dk-sk-gh" />
          <Skeleton className="dk-sk-row" />
          <Skeleton className="dk-sk-row" />
        </div>
      </aside>
      <div className="dk-mainc">
        <header className="dk-top" aria-hidden>
          <Skeleton className="dk-sk-title" />
        </header>
        <div className="dk-home-w">
          <div className="dk-home">
            <div className="dk-home-in">
              <Skeleton className="dk-sk-cmp" />
            </div>
          </div>
        </div>
      </div>
    </main>
  );
}
