import { DataTableSkeleton } from "@trenova/shared/components/data-table/data-table-skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQueryState } from "nuqs";
import { SAFETY_AGENTS_PARAM, safetyAgentsParser, type SafetyView } from "../../ai-control-tabs";
import {
  FigsSkeleton,
  HeroSkeleton,
  SecHeadSkeleton,
  SegSkeleton,
  SkBlock,
  SkLines,
  SkText,
  type FigSkeleton,
} from "../kit/tab-skeleton";
import { MAX_COMPARED_AGENTS } from "./safety-figures";

/** What the tool rules table shows at first: tool, audience, ceiling, holders, runs alone. */
export const TOOL_RULE_COLUMNS = 5;
/** What the by-agent table shows at first: agent, tool, audience, both answers, held by. */
const AGENT_TOOL_COLUMNS = 6;

/** Tools that run without a person, that send outside, and open agents holding them. */
const FIGURES: readonly FigSkeleton[] = [
  [172, 28, 212],
  [212, 28, 112],
  [176, 20, 104],
];

/** The audiences the tools reach, from the person's own work out to money, by share. */
const EGRESS_SEGMENTS = [
  { grow: 5, label: 52 },
  { grow: 14, label: 64 },
  { grow: 6, label: 104 },
  { grow: 4, label: 88 },
  { grow: 7, label: 112 },
  { grow: 3, label: 40 },
] as const;

const VIEW_WIDTHS = [74, 66] as const;
const CHIP_WIDTHS = [136, 120, 148, 112] as const;
const NAME_WIDTHS = [132, 112, 156, 120] as const;
const ROSTER_TILES = 8;
const EMPTY_NOTE = ["96%", "62%"] as const;

/**
 * The head of the Safety tab while what agents can do is read: Nova's sentence, the three
 * figures, and who sees the work of the tools that change things.
 */
export function SafetyHeadSkeleton() {
  return (
    <>
      <HeroSkeleton />
      <FigsSkeleton items={FIGURES} />
      <section className="sec">
        <SecHeadSkeleton title={112} note={148} />
        <div className="egm" aria-hidden>
          {EGRESS_SEGMENTS.map((segment, index) => (
            <div key={index} className="egm-s" style={{ flexGrow: segment.grow }}>
              <span className="egm-l">
                <b>
                  <SkText w={16} />
                </b>
                <span>
                  <SkText w={segment.label} />
                </span>
              </span>
              <span className="egm-b ui-shimmer" />
            </div>
          ))}
        </div>
      </section>
    </>
  );
}

/**
 * By agent while its code or the agents are read: the picker with a chip for each agent the
 * address names, then their headers over the table, or the roster to pick from when none is.
 */
export function ByAgentSkeleton() {
  const [stored] = useQueryState(SAFETY_AGENTS_PARAM, safetyAgentsParser);
  const picked = Math.min(new Set(stored).size, MAX_COMPARED_AGENTS);

  return (
    <div className="contents" aria-busy="true">
      <div className="tb" aria-hidden>
        <div className="pk">
          {CHIP_WIDTHS.slice(0, picked).map((width, index) => (
            <SkBlock key={index} w={width} h={28} r={4} />
          ))}
          {picked < MAX_COMPARED_AGENTS && <SkBlock w={picked > 0 ? 132 : 112} h={28} />}
        </div>
        <span className="sp" />
      </div>
      {picked === 0 ? (
        <div className="pk-e" aria-hidden>
          <b>
            <SkText w={344} />
          </b>
          <span>
            <SkLines widths={EMPTY_NOTE} />
          </span>
          <div className="ms">
            {Array.from({ length: ROSTER_TILES }, (_, index) => (
              <span key={index} className="ms-i">
                <SkBlock w={32} h={32} />
              </span>
            ))}
          </div>
        </div>
      ) : (
        <>
          <div className="ahs" aria-hidden>
            {NAME_WIDTHS.slice(0, picked).map((width, index) => (
              <div key={index} className="ah">
                <div className="ah-h">
                  <SkBlock w={28} h={28} />
                  <div className="sh-t">
                    <b>
                      <SkText w={width} />
                    </b>
                    <span>
                      <SkText w={184} />
                    </span>
                  </div>
                </div>
              </div>
            ))}
          </div>
          <DataTableSkeleton columnCount={AGENT_TOOL_COLUMNS} aria-hidden />
        </>
      )}
    </div>
  );
}

/** The table the view switch picks: every tool's rule, or the picked agents' tools. */
function SafetyBodySkeleton({ view }: { view: SafetyView }) {
  return view === "agents" ? (
    <ByAgentSkeleton />
  ) : (
    <DataTableSkeleton columnCount={TOOL_RULE_COLUMNS} aria-hidden />
  );
}

/**
 * The Safety tab while its code is read: the head, the view switch the tab draws itself,
 * and the view it opens on.
 */
export function SafetyTabSkeleton({ view }: { view: SafetyView }) {
  const t = useT();

  return (
    <div className="tabp" aria-busy="true" aria-label={t("Loading safety")}>
      <SafetyHeadSkeleton />
      <div className="vw">
        <SegSkeleton widths={VIEW_WIDTHS} />
      </div>
      <SafetyBodySkeleton view={view} />
    </div>
  );
}
