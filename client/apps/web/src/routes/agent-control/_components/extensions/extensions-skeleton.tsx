import { useT } from "@trenova/shared/i18n/use-t";
import { SkBlock, SkLines, SkSearch, SkText } from "../kit/tab-skeleton";

/** The featured extension's capabilities, two columns of two lines each. */
const CAPABILITIES = [
  ["100%", "62%"],
  ["100%", "48%"],
  ["100%", "70%"],
  ["100%", "36%"],
] as const;
/** The shelves: All extensions and On, the Categories heading, then each category. */
const SHELVES = [
  { label: 92, count: 8 },
  { label: 20, count: 8 },
] as const;
const CATEGORIES = [{ label: 84, count: 8 }] as const;
const SORT_WIDTHS = [48, 44, 34] as const;
const TILES = [
  { name: 92, vendor: 104, summary: ["100%", "58%"] },
  { name: 108, vendor: 88, summary: ["100%", "72%"] },
] as const;

function ShelfRow({ label, count }: { label: number; count: number }) {
  return (
    <span className="tsk-nav">
      <SkText w={label} />
      <SkText w={count} />
    </span>
  );
}

/**
 * The marketplace below its heading while the catalog is read: the featured extension
 * with its demo panel, the shelves beside the grid of tiles, and the note under it.
 */
export function ExtensionsBodySkeleton() {
  const t = useT();
  return (
    <div aria-busy="true" aria-label={t("Loading extensions")}>
      <section className="spot" aria-hidden>
        <div className="spot-m">
          <span className="spot-k">
            <SkText w={58} />
          </span>
          <div className="mkt-h">
            <SkBlock w={48} h={48} />
            <div className="mkt-n">
              <b>
                <SkText w={112} />
              </b>
              <span>
                <SkText w={120} />
              </span>
            </div>
            <SkBlock w={58} h={18} r={4} />
          </div>
          <p className="spot-p">
            <SkLines widths={["88%"]} />
          </p>
          <ul className="caps two">
            {CAPABILITIES.map((lines, index) => (
              <li key={index}>
                <SkBlock w={12} h={12} r={3} className="mt-0.5" />
                <span className="tsk-col">
                  <SkLines widths={lines} />
                </span>
              </li>
            ))}
          </ul>
          <div className="spot-a">
            <SkBlock w={168} h={36} />
            <span className="spot-s">
              <SkText w={196} />
            </span>
          </div>
        </div>
        <div className="demo ui-shimmer" />
      </section>
      <div className="mkt" aria-hidden>
        <div className="mkt-c">
          {SHELVES.map((shelf, index) => (
            <ShelfRow key={index} label={shelf.label} count={shelf.count} />
          ))}
          <span className="mkt-ch">
            <SkText w={64} />
          </span>
          {CATEGORIES.map((category, index) => (
            <ShelfRow key={index} label={category.label} count={category.count} />
          ))}
        </div>
        <div className="mkt-b">
          <div className="mkt-bh">
            <b>
              <SkText w={92} />
            </b>
            <em>
              <SkText w={8} />
            </em>
            <span className="sp" />
            <div className="seg sm">
              {SORT_WIDTHS.map((width, index) => (
                <span key={index} className="tsk-sg">
                  <SkText w={width} />
                </span>
              ))}
            </div>
          </div>
          <div className="mkt-g">
            {TILES.map((tile, index) => (
              <div key={index} className="mkt-t tsk-still">
                <div className="mkt-h">
                  <SkBlock w={40} h={40} />
                  <div className="mkt-n">
                    <b>
                      <SkText w={tile.name} />
                    </b>
                    <span>
                      <SkText w={tile.vendor} />
                    </span>
                  </div>
                  <SkBlock w={50} h={18} r={4} />
                </div>
                <p>
                  <SkLines widths={tile.summary} />
                </p>
                <div className="mkt-f">
                  <span className="mono">
                    <SkText w={40} />
                  </span>
                  <span>
                    <SkText w={38} />
                  </span>
                  <span className="sp" />
                  <span className="mkt-cta">
                    <SkText w={52} />
                  </span>
                </div>
              </div>
            ))}
          </div>
          <div className="mkt-more">
            <SkBlock w={13} h={13} r={3} />
            <span>
              <SkText w={420} />
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}

/**
 * The Extensions tab while its code and catalog arrive: the marketplace heading and its
 * large search, then the body.
 */
export function ExtensionsTabSkeleton() {
  const t = useT();
  return (
    <div className="tabp" aria-busy="true" aria-label={t("Loading extensions")}>
      <div className="mkt-top" aria-hidden>
        <div>
          <span className="tsk-h2">
            <SkText w={330} />
          </span>
          <p>
            <SkLines widths={[560, 236]} />
          </p>
        </div>
        <SkSearch large />
      </div>
      <ExtensionsBodySkeleton />
    </div>
  );
}
