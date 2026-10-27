import { CheckCircleIcon } from "@trenova/shared/components/icons";
import { GhostBar, GhostBox, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { cn } from "@trenova/shared/lib/utils";

// Each tab's empty state is drawn as the tab it is going to be: the rows, bars and code it will
// hold, faint and fading out under the words (EmptySheet). The sketches are small, because the
// panel under the editor is short, and decoration only: EmptySheet hides them from assistive
// technology.

const FRAME = "border-border/70 bg-card rounded-lg border text-left";

/** A severity dot, or a check where the row reads clean. */
function Mark({ clean }: { clean?: boolean }) {
  return clean ? (
    <CheckCircleIcon className="text-success size-3 shrink-0 opacity-60" />
  ) : (
    <span className="bg-muted size-2 shrink-0 rounded-full" />
  );
}

/**
 * The problems list: a file's heading with its findings under it. With `clean`, each row wears a
 * check, so the sketch says the project checked clean rather than that nothing ran.
 */
export function ProblemsSketch({ clean = false }: { clean?: boolean }) {
  const rows = ["w-40", "w-28", "w-36"];
  return (
    <div className={cn(FRAME, "flex flex-col gap-2.5 p-3")}>
      <span className="flex items-center gap-2">
        <GhostBox className="size-3" />
        <GhostLine className="w-24" />
      </span>
      {rows.map((width, index) => (
        <span key={index} className="flex items-center gap-2 pl-5">
          <Mark clean={clean} />
          <GhostLine className={width} />
          <GhostLine className="ml-auto w-8" />
        </span>
      ))}
    </div>
  );
}

/** The tests tab: the readiness strip, a pass bar, and test rows. */
export function TestsSketch() {
  const rows = ["w-36", "w-28", "w-40"];
  return (
    <div className={cn(FRAME, "flex flex-col gap-2.5 p-3")}>
      <span className="flex items-center gap-3">
        {["w-12", "w-16", "w-14"].map((width, index) => (
          <span key={index} className="flex items-center gap-1.5">
            <span className="bg-muted size-2 rounded-full" />
            <GhostLine className={width} />
          </span>
        ))}
      </span>
      <GhostBar share={70} className="w-full" />
      {rows.map((width, index) => (
        <span key={index} className="flex items-center gap-2">
          <Mark />
          <GhostLine className={width} />
          {index === 1 ? (
            <span className="border-border/70 ml-auto block h-3.5 w-14 rounded-badge border" />
          ) : null}
        </span>
      ))}
    </div>
  );
}

/** The run tab: map, arrow and sample above the document the run makes. */
export function RunSketch() {
  const json = ["w-6", "w-24 ml-3", "w-32 ml-3", "w-20 ml-6", "w-28 ml-3", "w-6"];
  return (
    <div className={cn(FRAME, "flex flex-col gap-3 p-3")}>
      <span className="flex items-center gap-2">
        <span className="border-border/70 block h-4 w-20 rounded border" />
        <GhostLine className="w-3" />
        <span className="border-border/70 block h-4 w-24 rounded border" />
        <span className="bg-muted ml-auto block h-4 w-12 rounded" />
      </span>
      <span className="border-border/70 flex flex-col gap-1.5 rounded-md border p-2">
        {json.map((width, index) => (
          <GhostLine key={index} className={cn("h-1", width)} />
        ))}
      </span>
    </div>
  );
}

/** The replays table: a row per replay with its map, period and how much read the same. */
export function ReplaysSketch() {
  const rows = [
    { map: "w-16", share: 92 },
    { map: "w-20", share: 74 },
    { map: "w-14", share: 85 },
  ];
  return (
    <div className={cn(FRAME, "flex flex-col gap-2.5 p-3")}>
      {rows.map((row, index) => (
        <span key={index} className="flex items-center gap-3">
          <Mark />
          <GhostLine className={row.map} />
          <GhostLine className="w-16" />
          <GhostBar share={row.share} className="ml-auto w-20" />
        </span>
      ))}
    </div>
  );
}

/** The shadow: how much read the same, and the messages that came in beside the active one. */
export function ShadowSketch() {
  return (
    <div className={cn(FRAME, "flex flex-col gap-2.5 p-3")}>
      <span className="flex items-center gap-2">
        <span className="bg-muted block h-3 w-8 rounded" />
        <GhostLine className="w-24" />
      </span>
      <GhostBar share={88} className="w-full" />
      {["w-28", "w-20"].map((width, index) => (
        <span key={index} className="flex items-center gap-2">
          <Mark />
          <span className="border-border/70 block h-3.5 w-8 rounded-badge border" />
          <GhostLine className={width} />
          <GhostLine className="ml-auto w-10" />
        </span>
      ))}
    </div>
  );
}

/** The settings table: a label and its name beside the value's field or switch. */
export function SettingsSketch() {
  const rows = [
    { label: "w-20", field: "switch" },
    { label: "w-28", field: "input" },
    { label: "w-16", field: "input" },
  ] as const;
  return (
    <div className={cn(FRAME, "flex flex-col divide-y")}>
      {rows.map((row, index) => (
        <span key={index} className="flex items-center gap-3 px-3 py-2">
          <span className="flex flex-col gap-1.5">
            <GhostLine className={row.label} />
            <GhostLine className="h-1 w-12" />
          </span>
          {row.field === "switch" ? (
            <span className="bg-muted ml-auto block h-3.5 w-6 rounded-full" />
          ) : (
            <span className="border-border/70 ml-auto block h-5 w-28 rounded border" />
          )}
        </span>
      ))}
    </div>
  );
}
