import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { cn } from "@trenova/shared/lib/utils";
import { PlusIcon } from "lucide-react";

const INDENT_REM = 1.25;

/**
 * The chart as it will look once positions exist: a top title with two
 * reporting to it, one of those with its own two beneath, in the same
 * indented rows the real tree draws. Each ghost row carries what a real one
 * does, the title and its code, a pill, the department line under it and the
 * headcount at the right, so what is missing reads at a glance.
 */
const GHOST_ROWS: readonly {
  depth: number;
  title: string;
  department: string;
  pill?: boolean;
  people: string;
}[] = [
  { depth: 0, title: "w-28", department: "w-16", pill: true, people: "w-3" },
  { depth: 1, title: "w-36", department: "w-24", people: "w-3" },
  { depth: 2, title: "w-32", department: "w-20", pill: true, people: "w-5" },
  { depth: 2, title: "w-24", department: "w-20", pill: true, people: "w-5" },
  { depth: 1, title: "w-28", department: "w-14", people: "w-3" },
];

type OrgStructureEmptyProps = {
  title: string;
  description: string;
  onAddPosition?: () => void;
  className?: string;
};

export function OrgStructureEmpty({
  title,
  description,
  onAddPosition,
  className,
}: OrgStructureEmptyProps) {
  const t = useT();

  return (
    <EmptySheet
      className={className}
      title={title}
      description={description}
      action={
        onAddPosition ? (
          <Button variant="outline" size="sm" onClick={onAddPosition}>
            <PlusIcon className="size-3.5" />
            {t("Add a position")}
          </Button>
        ) : null
      }
      sketch={
        <div className="border-border/70 bg-card divide-border/60 divide-y divide-dashed rounded-md border text-left">
          {GHOST_ROWS.map((row, index) => (
            <div
              key={index}
              className="relative grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 py-2 pr-3"
              style={{ paddingLeft: `${0.5 + row.depth * INDENT_REM}rem` }}
            >
              {row.depth > 0 ? (
                <span
                  className="bg-border absolute top-0 bottom-0 w-px"
                  style={{ left: `${0.5 + (row.depth - 1) * INDENT_REM + 0.6}rem` }}
                />
              ) : null}
              <div className="flex min-w-0 items-center gap-2">
                <span
                  className={cn(
                    "border-border/70 block size-3 shrink-0 rounded-sm border",
                    row.depth > 0 && "border-dashed",
                  )}
                />
                <div className="flex min-w-0 flex-col gap-1.5">
                  <span className="flex items-center gap-2">
                    <GhostLine className={`h-2 ${row.title}`} />
                    <GhostLine className="w-6" />
                    {row.pill ? (
                      <span className="border-border/70 block h-3.5 w-10 rounded-md border" />
                    ) : null}
                  </span>
                  <GhostLine className={row.department} />
                </div>
              </div>
              <GhostLine className={`h-2 ${row.people}`} />
            </div>
          ))}
        </div>
      }
    />
  );
}
