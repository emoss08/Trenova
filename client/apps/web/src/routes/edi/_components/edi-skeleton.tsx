import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableSkeleton } from "@trenova/shared/components/data-table/data-table-skeleton";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { EDIPageKind } from "./edi-types";
import { EDIOverviewSkeleton } from "./overview/edi-overview-skeleton";

type EDITableKind = Exclude<EDIPageKind, "overview">;

/**
 * The width of each column the kind's table draws, in the order it draws
 * them, taken from the column definitions' sizes, so the skeleton's columns
 * fall where the loaded ones do.
 */
const COLUMN_WIDTHS: Record<EDITableKind, readonly number[]> = {
  partners: [140, 220, 240, 180, 120, 120, 180, 220, 180],
  "communication-profiles": [280, 120, 120, 220, 220, 180],
  "mapping-profiles": [240, 260, 280, 120, 180],
  inbound: [160, 240, 280, 180, 180, 220],
  outbound: [160, 240, 280, 180, 180, 220],
  messages: [170, 220, 120, 170, 150, 140, 180],
  "inbound-files": [150, 280, 220, 110, 120, 140, 180],
  flows: [130, 220, 200, 220, 280, 170, 170],
};

/** The kinds whose table takes a selection column when the reader may act on rows in bulk. */
const SELECTABLE_KINDS: ReadonlySet<EDITableKind> = new Set([
  "inbound",
  "messages",
  "inbound-files",
]);

const SELECTION_COLUMN_WIDTH = 40;

function EDITableSkeleton({ kind, title }: { kind: EDITableKind; title: string }) {
  const t = useT();
  const canUpdate = usePermissionStore((state) =>
    state.hasPermission(Resource.EDI, Operation.Update),
  );

  const widths =
    canUpdate && SELECTABLE_KINDS.has(kind)
      ? [SELECTION_COLUMN_WIDTH, ...COLUMN_WIDTHS[kind]]
      : COLUMN_WIDTHS[kind];
  const table = (
    <DataTableSkeleton
      columnCount={widths.length}
      cellWidths={widths.map((width) => `${width}px`)}
    />
  );

  return (
    <div className="flex flex-col gap-1" aria-busy aria-label={t("Loading {0}", title)}>
      <div className={kind === "partners" ? "flex flex-col gap-4" : "flex flex-col"} aria-hidden>
        {table}
      </div>
    </div>
  );
}

type EDIPageSkeletonProps = {
  kind: EDIPageKind;
  /** The page's title, which names what is loading. */
  title: string;
};

/**
 * The first paint of every EDI page, drawn as that page's own layout: the
 * operations overview or the kind's table
 * framed the way the page frames it.
 */
export function EDIPageSkeleton({ kind, title }: EDIPageSkeletonProps) {
  if (kind === "overview") {
    return <EDIOverviewSkeleton />;
  }
  return <EDITableSkeleton kind={kind} title={title} />;
}
