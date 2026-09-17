import { GhostBar, GhostLine } from "@trenova/shared/components/ui/empty-sheet";

export function IntelEmptySketch() {
  return (
    <div className="flex flex-col gap-3 rounded-lg border p-4">
      <div className="flex items-center gap-2">
        <GhostLine className="w-24" />
        <GhostLine className="w-16" />
      </div>
      <GhostBar share={40} className="w-full" />
      <GhostBar share={70} className="w-full" />
      <GhostBar share={20} className="w-full" />
      <GhostLine className="w-2/3" />
    </div>
  );
}
