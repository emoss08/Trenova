import { DeskConversationSkeleton } from "@/components/desk-chat/desk-skeletons";
import { conversationAtPath } from "@/lib/conversation-path";
import { useT } from "@trenova/shared/i18n/use-t";
import { useLocation } from "react-router";
import "@/components/desk-chat/desk-frame.css";

/** How wide each of the rail's places is drawn: Today, Watchtower, Decisions, Scheduled, Memory. */
const PLACE_WIDTHS = [38, 58, 52, 54, 44] as const;

/**
 * The rail's six shelves, from pinned to older, each a heading over
 * conversations of these widths: more than the tallest screen holds, so the
 * rail is full to its foot and fades out there as the real one does.
 */
const SHELF_ROWS = [
  [64, 82],
  [78, 62, 84, 70],
  [70, 88, 56, 74, 66],
  [82, 60, 76, 90, 58, 72],
  [68, 86, 74, 62, 80, 54, 88],
  [76, 58, 84, 66, 90, 72, 60, 80, 70, 86],
] as const;

/** What the Desk is opening onto, which decides the page drawn under the strip. */
type Opening = "home" | "conversation" | "place";

function openingAt(pathname: string): Opening {
  if (conversationAtPath(pathname) !== null) {
    return "conversation";
  }

  return pathname === "/desk" || pathname === "/desk/" ? "home" : "place";
}

function percent(width: number) {
  return { width: `${width}%` };
}

/**
 * The Desk while it is first opened: the room as it is about to be drawn —
 * the rail with its places, shelves and the person's row, the strip across
 * the top and the page under it — so nothing moves when the Desk takes over.
 * A conversation's address opens on a conversation, the Desk's own on Today,
 * and any other place on its strip, which its page fills in once it loads.
 */
export function DeskLoadingScreen() {
  const t = useT();
  const opening = openingAt(useLocation().pathname);

  return (
    <main data-slot="desk-loading-screen" className="dsk" aria-busy>
      <span role="status" className="sr-only">
        {t("Opening the desk")}
      </span>
      <aside className="dk-sb" aria-hidden>
        <div className="dk-sb-top">
          <i className="dk-sk dk-sk-logo" />
          <span className="dk-sk-sp" />
          <i className="dk-sk dk-sk-ic" />
          <i className="dk-sk dk-sk-ic" />
        </div>
        <div className="dk-sb-list dk-skr-list">
          {PLACE_WIDTHS.map((width) => (
            <div key={width} className="dk-skr">
              <i className="dk-sk dk-sk-glyph" />
              <i className="dk-sk dk-sk-label" style={percent(width)} />
            </div>
          ))}
          {SHELF_ROWS.map((rows, shelf) => (
            // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
            <div key={shelf}>
              <div className="dk-skr-gh">
                <i className="dk-sk" />
              </div>
              {rows.map((width, row) => (
                // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
                <div key={row} className="dk-skr">
                  <i className="dk-sk dk-sk-dot" />
                  <i className="dk-sk dk-sk-label" style={percent(width)} />
                </div>
              ))}
            </div>
          ))}
        </div>
        <div className="dk-skr-me">
          <i className="dk-sk dk-sk-me" />
          <i className="dk-sk dk-sk-name" />
          <span className="dk-sk-sp" />
          <i className="dk-sk dk-sk-ic" />
          <i className="dk-sk dk-sk-ic" />
        </div>
      </aside>
      <div className="dk-mainc">
        <header className="dk-top" aria-hidden>
          {opening === "conversation" ? (
            <>
              <i className="dk-sk dk-sk-tile" />
              <i className="dk-sk dk-sk-agent" />
              <i className="dk-sk dk-sk-title" />
              <span className="dk-sk-sp" />
              <i className="dk-sk dk-sk-ic" />
              <i className="dk-sk dk-sk-ic" />
              <i className="dk-sk dk-sk-ic" />
              <i className="dk-sk dk-sk-ws" />
            </>
          ) : (
            <i className="dk-sk dk-sk-place" />
          )}
        </header>
        {opening === "conversation" ? (
          <DeskConversationSkeleton announce={false} />
        ) : opening === "home" ? (
          <div className="dk-home-w" aria-hidden>
            <div className="dk-home">
              <div className="dk-home-in">
                <i className="dk-sk dk-sk-date" />
                <i className="dk-sk dk-sk-greet" />
                <i className="dk-sk dk-sk-headline" />
                <i className="dk-sk dk-sk-cmp" />
              </div>
            </div>
          </div>
        ) : (
          <div aria-hidden />
        )}
      </div>
    </main>
  );
}
