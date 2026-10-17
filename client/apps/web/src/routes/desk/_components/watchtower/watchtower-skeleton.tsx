import { useT } from "@trenova/shared/i18n/use-t";
import "@/components/desk-chat/desk-frame.css";

/** The lanes and rows the queue is usually read in, at the widths of real titles. */
const LANES = [[82, 70, 88, 64], [76], [68, 84, 72]] as const;

function box(width: number | string, height: number) {
  return { width: typeof width === "number" ? `${width}%` : width, height };
}

/**
 * The Watchtower while its queue is first read: the queue's header, lanes and
 * rows down the left, and the card's clock, title and record in the middle,
 * each where the real one lands, so nothing moves when the queue arrives.
 */
export function WatchtowerSkeleton() {
  const t = useT();

  return (
    <div className="dk-dc2 dk-w3" aria-busy="true" aria-label={t("Reading the Watchtower")}>
      <aside className="dk-dc2-q" aria-hidden>
        <div className="dk-dc2-qh">
          <i className="dk-sk" style={box(84, 12)} />
          <i className="dk-sk" style={box(64, 10)} />
        </div>
        <div className="dk-dc2-prog" />
        <div className="dk-dc2-ql">
          {LANES.map((rows, lane) => (
            <div key={lane} className="dk-dc2-g">
              <div className="dk-dc2-gh">
                <i className="dk-sk" style={box(56, 9)} />
              </div>
              {rows.map((width, row) => (
                <div key={row} className="dk-dc2-qi flex gap-2.25 px-2 py-1.75">
                  <i className="dk-sk flex-none" style={box("22px", 22)} />
                  <span className="flex min-w-0 flex-1 flex-col gap-1.5">
                    <i className="dk-sk" style={box(width, 11)} />
                    <i className="dk-sk" style={box(width / 2, 9)} />
                  </span>
                </div>
              ))}
            </div>
          ))}
        </div>
      </aside>
      <main className="dk-dc2-main" aria-hidden>
        <div className="dk-dc2-card">
          <div className="dk-w3-clock">
            <i className="dk-sk rounded-full" style={box("44px", 44)} />
            <span className="flex flex-col gap-1.5">
              <i className="dk-sk" style={box("88px", 20)} />
              <i className="dk-sk" style={box("140px", 10)} />
            </span>
          </div>
          <i className="dk-sk" style={box(30, 10)} />
          <i className="dk-sk mt-3" style={box(78, 26)} />
          <i className="dk-sk mt-3" style={box(92, 12)} />
          <i className="dk-sk mt-2" style={box(64, 12)} />
          <i className="dk-sk mt-5" style={box(100, 46)} />
          <i className="dk-sk mt-5" style={box(100, 74)} />
        </div>
      </main>
    </div>
  );
}
