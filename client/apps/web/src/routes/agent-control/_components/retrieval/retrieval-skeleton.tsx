import { usePermission } from "@/hooks/use-permission";
import { DataTableSkeleton } from "@trenova/shared/components/data-table/data-table-skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  FigsSkeleton,
  HeroSkeleton,
  SecHeadSkeleton,
  SkBlock,
  SkLines,
  SkSwitch,
  SkText,
  type FigSkeleton,
} from "../kit/tab-skeleton";

/** What the failed items table shows at first: source, status, error, attempts, last attempt. */
export const FAILED_ENTRY_COLUMNS = 5;

/** Indexed, waiting, failed, this month's cost against the budget, and the last run. */
const FIGURES: readonly FigSkeleton[] = [
  [52, 84, 124],
  [52, 52, 84],
  [40, 36, 88],
  [96, 76, 104, true],
  [52, 92, 84],
];

/** The budget's note and the pause's, line by line. */
const BUDGET_NOTE = ["96%", "88%", "42%"] as const;
const PAUSE_NOTE = ["94%", "56%"] as const;

/** Memory, documents and inbound email: each one's name, what it holds and its counts. */
const SOURCES = [
  { name: 64, about: 248, counts: [76, 68] },
  { name: 76, about: 292, counts: [84, 64] },
  { name: 96, about: 268, counts: [72, 60] },
] as const;

type SourceRowSkeletonProps = (typeof SOURCES)[number] & { canUpdate: boolean };

function SourceRowSkeleton({ name, about, counts, canUpdate }: SourceRowSkeletonProps) {
  return (
    <div className="src">
      <span className="src-i ui-shimmer" />
      <div className="src-m">
        <div className="src-h">
          <SkText w={name} />
          <SkBlock w={64} h={18} />
        </div>
        <p>
          <SkText w={about} />
        </p>
        <div className="sbar" />
        <div className="src-n">
          {counts.map((width, index) => (
            <SkText key={index} w={width} />
          ))}
          <SkText w={40} />
        </div>
      </div>
      {canUpdate && (
        <div className="src-c">
          <SkBlock w={92} h={28} />
          <SkSwitch />
        </div>
      )}
    </div>
  );
}

/** The budget and the pause, each a heading, its control and the note under it. */
function SettingsSkeleton() {
  return (
    <section className="sec">
      <SecHeadSkeleton title={56} end={{ note: 148 }} />
      <div className="pol" aria-hidden>
        <div className="po">
          <div className="po-h">
            <SkBlock w={14} h={14} />
            <SkText w={164} />
          </div>
          <SkBlock w="100%" h={34} />
          <p>
            <SkLines widths={BUDGET_NOTE} />
          </p>
        </div>
        <div className="po">
          <div className="po-h">
            <SkBlock w={14} h={14} />
            <SkText w={104} />
            <span className="sp" />
            <SkSwitch />
          </div>
          <p>
            <SkLines widths={PAUSE_NOTE} />
          </p>
        </div>
      </div>
    </section>
  );
}

/**
 * The Retrieval tab while its code or where the index stands is read: Nova's sentence and
 * its control, the five figures, the three sources and the failed items beside the
 * settings, each in the column the loaded tab puts it in.
 */
export function RetrievalTabSkeleton() {
  const t = useT();
  const { allowed: canUpdate } = usePermission(Resource.AIProvider, Operation.Update);

  return (
    <div className="tabp" aria-busy="true" aria-label={t("Loading retrieval")}>
      <HeroSkeleton control={canUpdate ? { button: 148, note: 112 } : undefined} />
      <FigsSkeleton items={FIGURES} />
      <div className="ov">
        <div className="ov-m">
          <section className="sec">
            <SecHeadSkeleton title={56} end={{ note: 176 }} />
            <div className="srcs" aria-hidden>
              {SOURCES.map((source, index) => (
                <SourceRowSkeleton key={index} {...source} canUpdate={canUpdate} />
              ))}
            </div>
          </section>
          <section className="sec">
            <DataTableSkeleton columnCount={FAILED_ENTRY_COLUMNS} aria-hidden />
          </section>
        </div>
        <aside className="ov-a">
          <SettingsSkeleton />
        </aside>
      </div>
    </div>
  );
}
