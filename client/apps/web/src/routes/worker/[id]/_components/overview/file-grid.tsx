import { HoverGlideGroup } from "@/components/hover-glide";
import { SectionPanel } from "@/components/section-panel";
import type { WorkerOverview } from "@/lib/graphql/worker-overview";
import type { WorkerTarget } from "@/routes/worker/use-worker-state";
import type { WorkerAreas } from "@/routes/worker/worker-areas";
import {
  ArrowRightIcon,
  Beaker02Icon,
  CalendarRangeIcon,
  ClipboardCheckIcon,
  FolderCheckIcon,
  GraduationHat01Icon,
  HeartHandIcon,
  IdCardIcon,
  Lock01Icon,
  Phone01Icon,
  ShieldTickIcon,
  Star01Icon,
  SunsetIcon,
  Wallet02Icon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { useElementWidth } from "@trenova/shared/hooks/use-element-width";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  fileCellRules,
  fileCells,
  fileColumns,
  type FileCellKey,
  type FileNoteTone,
} from "./file-model";

const CELL_ICONS: Record<FileCellKey, IconComponent> = {
  credentials: IdCardIcon,
  "dq-file": FolderCheckIcon,
  "drug-and-alcohol": Beaker02Icon,
  training: GraduationHat01Icon,
  safety: ShieldTickIcon,
  reviews: Star01Icon,
  checklists: ClipboardCheckIcon,
  "time-off": SunsetIcon,
  leave: CalendarRangeIcon,
  "pay-profile": Wallet02Icon,
  benefits: HeartHandIcon,
  "driver-portal": Phone01Icon,
};

const NOTE_TONES: Record<FileNoteTone, string> = {
  muted: "text-muted-foreground",
  warning: "text-warning",
  danger: "text-danger",
};

const GRID_COLUMNS = { 1: "grid-cols-1", 2: "grid-cols-2", 3: "grid-cols-3" } as const;

/**
 * Every part of the worker's record the reader can open and where it stands, as a ruled grid
 * of buttons that open that tab and view. It takes three columns, then two, then one as its
 * width narrows, and draws its hairlines for whatever number of cells the plan leaves.
 */
export function FileGrid({
  overview,
  areas,
  onOpen,
}: {
  overview: WorkerOverview;
  areas: Pick<WorkerAreas, "isDrawable">;
  onOpen: (target: WorkerTarget) => void;
}) {
  const t = useT();
  const [ref, width] = useElementWidth<HTMLDivElement>();
  const cells = fileCells(overview, areas, t);
  if (cells.length === 0) return null;
  const columns = fileColumns(width);

  return (
    <SectionPanel title={t("The file")} hint={t("Every part of the record and where it stands")}>
      <div ref={ref} className="min-w-0">
        <HoverGlideGroup className={cn("grid p-1.5", GRID_COLUMNS[columns])}>
          {cells.map((cell, index) => {
            const Icon = CELL_ICONS[cell.key];
            const rules = fileCellRules(index, cells.length, columns);
            return (
              <button
                key={cell.key}
                type="button"
                data-glide-target
                data-file-cell={cell.key}
                data-rule-right={rules.right ? "" : undefined}
                data-rule-bottom={rules.bottom ? "" : undefined}
                onClick={() =>
                  onOpen({
                    tab: cell.area.tab,
                    ...(cell.area.view ? { view: cell.area.view } : {}),
                  } as WorkerTarget)
                }
                className={cn(
                  "group/cell relative flex min-w-0 flex-col gap-0.5 rounded-md ui-focus-ring px-3 pt-2.5 pb-[11px] text-left",
                  rules.right &&
                    "before:absolute before:top-2 before:-right-px before:bottom-2 before:w-px before:bg-border-subtle",
                  rules.bottom &&
                    "after:absolute after:right-1.5 after:-bottom-px after:left-1.5 after:h-px after:bg-border-subtle",
                )}
              >
                <span className="flex items-center gap-1.5 text-xs font-medium text-foreground-subtle">
                  <Icon aria-hidden className="size-3.5" />
                  {cell.label}
                </span>
                {cell.state === null ? (
                  <span className="flex items-center gap-1 text-sm text-foreground-muted">
                    <Lock01Icon aria-hidden className="size-3" />
                    {t("Hidden from you")}
                  </span>
                ) : (
                  <span className="truncate text-sm">{cell.state}</span>
                )}
                {cell.note ? (
                  <span className={cn("truncate text-xs tabular-nums", NOTE_TONES[cell.note.tone])}>
                    {cell.note.text}
                  </span>
                ) : null}
                <ArrowRightIcon
                  aria-hidden
                  className="absolute top-2.5 right-2.5 size-3.5 -translate-x-[3px] text-foreground-subtle opacity-0 transition-[opacity,translate] group-hover/cell:translate-x-0 group-hover/cell:opacity-100 group-focus-visible/cell:translate-x-0 group-focus-visible/cell:opacity-100"
                />
              </button>
            );
          })}
        </HoverGlideGroup>
      </div>
    </SectionPanel>
  );
}
