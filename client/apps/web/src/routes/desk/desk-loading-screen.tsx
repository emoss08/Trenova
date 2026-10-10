import { useT } from "@trenova/shared/i18n/use-t";
import "@/components/desk-chat/desk-chat.css";
import "./_styles/desk-v2.css";

/**
 * The Desk while it is first opened: the room's shape — the rail, the strip
 * across the top and the box in the middle — in place of the app-wide
 * loading card, so nothing jumps when the Desk takes over.
 */
export function DeskLoadingScreen() {
  const t = useT();

  return (
    <main data-slot="desk-loading-screen" className="dsk" aria-busy>
      <span role="status" className="sr-only">
        {t("Opening the desk")}
      </span>
      <aside className="dk-sb" aria-hidden>
        <div className="dk-sb-top">
          <i className="dk-sk dk-sk-logo" />
        </div>
        <div className="dk-sb-list">
          <i className="dk-sk dk-sk-row" />
          <i className="dk-sk dk-sk-row" />
          <i className="dk-sk dk-sk-row" />
          <i className="dk-sk dk-sk-gh" />
          <i className="dk-sk dk-sk-row" />
          <i className="dk-sk dk-sk-row" />
        </div>
      </aside>
      <div className="dk-mainc">
        <header className="dk-top" aria-hidden>
          <i className="dk-sk dk-sk-title" />
        </header>
        <div className="dk-home-w">
          <div className="dk-home">
            <div className="dk-home-in">
              <i className="dk-sk dk-sk-cmp" />
            </div>
          </div>
        </div>
      </div>
    </main>
  );
}
