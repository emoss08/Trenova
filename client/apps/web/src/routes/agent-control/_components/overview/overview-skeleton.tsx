import { DataTableSkeleton } from "@trenova/shared/components/data-table/data-table-skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import type { CSSProperties } from "react";
import { HeroSkeleton, SecHeadSkeleton, SkBlock, SkIconButton, SkText } from "../kit/tab-skeleton";

/** The four figures: the label, the value and the line under it, each at its loaded width. */
const FIGURES = [
  { label: 112, value: 72, sub: 52 },
  { label: 92, value: 66, sub: 156 },
  { label: 40, value: 60, sub: 104 },
  { label: 36, value: 82, sub: 118 },
] as const;
/** A week of calls under the first figure, oldest first. */
const DAY_BAR_HEIGHTS = [9, 13, 7, 16, 12, 18, 10] as const;
const TUNE_UPS = [
  { title: "62%", evidence: "84%" },
  { title: "48%", evidence: "70%" },
] as const;
const ROSTER_TILES = 8;
const LEGEND_WIDTHS = [38, 52, 30] as const;
const POLICY_ROWS = [
  { label: 104, value: 96 },
  { label: 128, value: 22 },
  { label: 112, value: 76 },
  { label: 112, value: 22 },
  { label: 52, value: 92 },
  { label: 48, value: 86 },
] as const;
const USAGE_COLUMNS = 6;
const USAGE_ROWS = 6;
/** What the "Hold to pause all agents" control measures. */
const HOLD_CONTROL_WIDTH = 230;
/** The note under it. */
const HOLD_NOTE_WIDTH = 150;

/** The week in four figures while usage is read. */
export function OverviewFiguresSkeleton() {
  const t = useT();
  return (
    <section
      className="figs"
      aria-busy="true"
      aria-label={t("Loading figures")}
      style={{ "--n": FIGURES.length } as CSSProperties}
    >
      {FIGURES.map((figure, index) => (
        <div key={index} className="fg" aria-hidden>
          <span className="lbl">
            <SkText w={figure.label} />
          </span>
          <b className="big">
            <SkText w={figure.value} />
          </b>
          <span className="sub">
            {index === 0 ? (
              <span className="fsub">
                <span className="fbars">
                  {DAY_BAR_HEIGHTS.map((height, day) => (
                    <SkBlock key={day} w={5} h={height} r={1.5} />
                  ))}
                </span>
                <SkText w={figure.sub} />
              </span>
            ) : (
              <SkText w={figure.sub} />
            )}
          </span>
        </div>
      ))}
    </section>
  );
}

/** The tune-up rows while they are read: the mark, the change and its evidence, the gain. */
export function TuneUpsSkeleton() {
  const t = useT();
  return (
    <div className="nds" aria-busy="true" aria-label={t("Loading tune-ups")}>
      {TUNE_UPS.map((row, index) => (
        <div key={index} className="nd tu" aria-hidden>
          <SkBlock w={30} h={30} r={8} />
          <div className="nd-b">
            <b>
              <SkText w={row.title} />
            </b>
            <span>
              <SkText w={row.evidence} />
            </span>
          </div>
          <SkBlock w={88} h={22} r={4} />
          <div className="nd-a">
            <SkIconButton />
            <SkBlock w={72} h={28} />
          </div>
        </div>
      ))}
    </div>
  );
}

/** Under the Agents heading while the roster is read: what is running, the tiles, the key. */
export function AgentsAtWorkBodySkeleton() {
  const t = useT();
  return (
    <div aria-busy="true" aria-label={t("Loading agents")}>
      <p className="wn-e" aria-hidden>
        <SkText w={170} />
      </p>
      <div className="ms" aria-hidden>
        {Array.from({ length: ROSTER_TILES }, (_, index) => (
          <span key={index} className="ms-i tsk-still">
            <SkBlock w={28} h={28} r={8} />
          </span>
        ))}
      </div>
      <div className="lgd" aria-hidden>
        {LEGEND_WIDTHS.map((width, index) => (
          <span key={index}>
            <SkBlock w={8} h={8} r={3} />
            <span className="tsk-ln">
              <SkText w={width} />
            </span>
          </span>
        ))}
      </div>
    </div>
  );
}

/** The settings that apply to every agent, while they are read. */
export function OrganizationWideSkeleton() {
  const t = useT();
  return (
    <section className="sec" aria-busy="true" aria-label={t("Loading organization-wide settings")}>
      <SecHeadSkeleton title={116} end={{ button: 58 }} />
      <div className="os" aria-hidden>
        {POLICY_ROWS.map((row, index) => (
          <div key={index} className="os-r tsk-still">
            <SkBlock w={13} h={13} r={3} />
            <span>
              <SkText w={row.label} />
            </span>
            <em>
              <SkText w={row.value} />
            </em>
          </div>
        ))}
      </div>
    </section>
  );
}

/**
 * The overview while its code and first reads arrive: Nova's sentence and the pause
 * control, the four figures, then tune-ups and usage by feature beside the agents and the
 * organization-wide settings. It assumes a provider is connected, the usual case, since
 * the setup steps only replace the figures when none is.
 */
export function OverviewTabSkeleton() {
  const t = useT();
  return (
    <div className="tabp ov2" aria-busy="true" aria-label={t("Loading the overview")}>
      <HeroSkeleton control={{ button: HOLD_CONTROL_WIDTH, note: HOLD_NOTE_WIDTH }} />
      <OverviewFiguresSkeleton />
      <div className="ov">
        <div className="ov-m">
          <section className="sec">
            <SecHeadSkeleton title={64} end={{ note: 170 }} />
            <TuneUpsSkeleton />
          </section>
          <section className="sec" aria-hidden>
            <SecHeadSkeleton title={118} note={40} />
            <DataTableSkeleton columnCount={USAGE_COLUMNS} rowCount={USAGE_ROWS} />
          </section>
        </div>
        <aside className="ov-a">
          <section className="sec">
            <SecHeadSkeleton title={48} note={56} end={{ note: 52 }} />
            <AgentsAtWorkBodySkeleton />
          </section>
          <OrganizationWideSkeleton />
        </aside>
      </div>
    </div>
  );
}
