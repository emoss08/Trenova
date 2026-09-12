import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostBar, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { PlusIcon } from "lucide-react";

/**
 * The shape of the page once plans exist: a kind heading with two plans under
 * it, then the start of another kind. Each ghost plan carries the same parts
 * a real row does, its name, carrier line, who-is-on-it track and the two
 * costs, so what is missing reads at a glance.
 */
const GHOST_GROUPS: readonly {
  rows: readonly { name: string; detail: string; share: number }[];
}[] = [
  {
    rows: [
      { name: "w-28", detail: "w-3/5", share: 60 },
      { name: "w-20", detail: "w-2/5", share: 25 },
    ],
  },
  { rows: [{ name: "w-24", detail: "w-1/2", share: 40 }] },
];

type BenefitsEmptyProps = {
  title: string;
  description: string;
  onAddPlan?: () => void;
  className?: string;
};

export function BenefitsEmpty({ title, description, onAddPlan, className }: BenefitsEmptyProps) {
  const t = useT();

  return (
    <EmptySheet
      className={className}
      title={title}
      description={description}
      action={
        onAddPlan ? (
          <Button variant="outline" size="sm" onClick={onAddPlan}>
            <PlusIcon className="size-3.5" />
            {t("Add a plan")}
          </Button>
        ) : null
      }
      sketch={
        <div className="flex flex-col gap-3">
          {GHOST_GROUPS.map((group, groupIndex) => (
            <div key={groupIndex} className="flex flex-col gap-1.5">
              <div className="flex items-center justify-between px-1">
                <GhostLine className="w-16" />
                <GhostLine className="w-24" />
              </div>
              <div className="border-border/70 bg-card divide-border/60 divide-y divide-dashed rounded-md border">
                {group.rows.map((row, rowIndex) => (
                  <div
                    key={rowIndex}
                    className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2.5"
                  >
                    <div className="flex flex-col gap-2">
                      <span className="flex items-center gap-2">
                        <GhostLine className={`h-2 ${row.name}`} />
                        <GhostLine className="w-8" />
                      </span>
                      <GhostLine className={row.detail} />
                      <span className="flex items-center gap-2">
                        <GhostBar share={row.share} className="w-32" />
                        <GhostLine className="w-10" />
                      </span>
                    </div>
                    <div className="grid grid-cols-[auto_auto] items-center gap-x-2 gap-y-1.5">
                      <GhostLine className="w-10" />
                      <GhostLine className="w-12" />
                      <GhostLine className="w-10" />
                      <GhostLine className="w-12" />
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      }
    />
  );
}
