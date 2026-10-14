import { useT } from "@trenova/shared/i18n/use-t";
import {
  HeroSkeleton,
  SkBlock,
  SkIconButton,
  SkSearch,
  SkSwitch,
  SkText,
} from "../kit/tab-skeleton";

/** The filter choice: All, Waiting, Shadow and Off, each with its count. */
const FILTER_WIDTHS = [30, 58, 56, 32] as const;
/** The column heads over each shelf; the first and last columns have none. */
const HEAD_WIDTHS = [36, 70, 54, 44, 30] as const;
/** Two shelves, as a roster of chat and scheduled agents reads. */
const SHELVES = [
  {
    title: 44,
    note: 232,
    rows: [
      { name: 128, description: "78%", runs: 44, rate: 28, tools: 14 },
      { name: 104, description: "64%", runs: 38, rate: 28, tools: 10 },
      { name: 140, description: "72%", runs: 44, rate: 22, tools: 14 },
    ],
  },
  {
    title: 66,
    note: 196,
    rows: [
      { name: 116, description: "70%", runs: 38, rate: 28, tools: 10 },
      { name: 96, description: "58%", runs: 44, rate: 22, tools: 14 },
    ],
  },
] as const;
/** Fourteen days of runs, oldest first. */
const RUN_HEIGHTS = [6, 10, 4, 12, 8, 14, 18, 9, 5, 11, 16, 7, 13, 10] as const;
const NEW_AGENT_WIDTH = 120;

/** The agents under their trigger shelves while the roster is read. */
export function AgentShelvesSkeleton() {
  const t = useT();
  return (
    <div aria-busy="true" aria-label={t("Loading the roster")}>
      {SHELVES.map((shelf, shelfIndex) => (
        <section key={shelfIndex} className="shelf" aria-hidden>
          <header className="shelf-h">
            <SkBlock w={13} h={13} r={3} className="self-center" />
            <b>
              <SkText w={shelf.title} />
            </b>
            <span>
              <SkText w={shelf.note} />
            </span>
            <em>
              <SkText w={8} />
            </em>
          </header>
          <div className="al-hd">
            <span />
            {HEAD_WIDTHS.map((width, index) => (
              <span key={index}>
                <SkText w={width} />
              </span>
            ))}
            <span />
          </div>
          <div className="rows">
            {shelf.rows.map((row, index) => (
              <div key={index} className="al tsk-still">
                <span className="pc-mk">
                  <SkBlock w={32} h={32} r={9} />
                </span>
                <span className="al-t">
                  <span className="al-n">
                    <b>
                      <SkText w={row.name} />
                    </b>
                  </span>
                  <span className="al-d">
                    <SkText w={row.description} />
                  </span>
                </span>
                <span className="al-c al-act">
                  <span className="runs">
                    {RUN_HEIGHTS.map((height, day) => (
                      <SkBlock key={day} w={3} h={height} r={1} />
                    ))}
                  </span>
                  <span className="al-s">
                    <SkText w={row.runs} />
                  </span>
                </span>
                <span className="al-c al-ap">
                  <SkBlock w={18} h={18} r="full" />
                  <span className="mono">
                    <SkText w={row.rate} />
                  </span>
                </span>
                <span className="al-c al-w">
                  <span className="dim">
                    <SkText w={10} />
                  </span>
                </span>
                <span className="al-c al-tl">
                  <SkBlock w={70} h={5} r="full" className="tsk-tier" />
                  <span className="mono">
                    <SkText w={row.tools} />
                  </span>
                </span>
                <span className="al-x">
                  <SkIconButton />
                  <SkSwitch />
                </span>
              </div>
            ))}
          </div>
        </section>
      ))}
    </div>
  );
}

/**
 * The Agents tab while its code and roster arrive: Nova's sentence, the search, filter and
 * New agent toolbar, then two shelves of agent rows in the roster's own columns, which the
 * page's container queries fold the same way as the loaded rows.
 */
export function AgentsTabSkeleton() {
  const t = useT();
  return (
    <div className="tabp" aria-busy="true" aria-label={t("Loading agents")}>
      <HeroSkeleton />
      <div className="tb" aria-hidden>
        <SkSearch />
        <div className="seg">
          {FILTER_WIDTHS.map((width, index) => (
            <span key={index} className="tsk-sg">
              <SkText w={width} />
            </span>
          ))}
        </div>
        <span className="sp" />
        <span className="tb-ct mono">
          <SkText w={54} />
        </span>
        <SkBlock w={NEW_AGENT_WIDTH} h={32} />
      </div>
      <AgentShelvesSkeleton />
    </div>
  );
}
