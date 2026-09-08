import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";

/**
 * The roster as it will look once people report in: a path heading with its
 * count, then a row per person carrying the same parts a real one does. The
 * avatar, name and title, the three health dots, the tenure at the end. Two
 * direct reports first, then the start of a terminal's group, so both ways of
 * joining the team are drawn.
 */
const GHOST_GROUPS: readonly {
  heading: string;
  rows: readonly { name: string; detail: string }[];
}[] = [
  {
    heading: "w-12",
    rows: [
      { name: "w-28", detail: "w-40" },
      { name: "w-20", detail: "w-32" },
    ],
  },
  { heading: "w-16", rows: [{ name: "w-24", detail: "w-36" }] },
];

const HEALTH_DOTS = ["w-14", "w-10", "w-12"] as const;

type MyTeamEmptyProps = {
  title: string;
  description: string;
  className?: string;
};

export function MyTeamEmpty({ title, description, className }: MyTeamEmptyProps) {
  return (
    <EmptySheet
      className={className}
      title={title}
      description={description}
      sketch={
        <div className="flex flex-col gap-3">
          {GHOST_GROUPS.map((group, groupIndex) => (
            <div key={groupIndex} className="flex flex-col gap-1.5">
              <div className="flex items-center gap-1.5 px-1">
                <GhostLine className={group.heading} />
                <GhostLine className="w-3" />
              </div>
              <div className="border-border/70 bg-card divide-border/60 divide-y divide-dashed rounded-md border">
                {group.rows.map((row, rowIndex) => (
                  <div
                    key={rowIndex}
                    className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2.5"
                  >
                    <div className="flex min-w-0 items-center gap-2.5">
                      <span className="bg-muted size-8 shrink-0 rounded-full" />
                      <div className="flex min-w-0 flex-col gap-2">
                        <GhostLine className={`h-2 ${row.name}`} />
                        <GhostLine className={row.detail} />
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <span className="hidden items-center gap-3 md:flex">
                        {HEALTH_DOTS.map((width) => (
                          <span key={width} className="flex items-center gap-1.5">
                            <span className="bg-muted size-1.5 shrink-0 rounded-full" />
                            <GhostLine className={width} />
                          </span>
                        ))}
                      </span>
                      <GhostLine className="hidden w-10 sm:block" />
                      <span className="bg-muted size-3 rounded-sm" />
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
