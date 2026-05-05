import { CalendarClockIcon } from "lucide-react";

export function TimelinePlaceholder() {
  return (
    <div className="flex h-[300px] flex-col items-center justify-center gap-2 rounded-md border border-dashed border-border text-center">
      <CalendarClockIcon className="size-6 text-muted-foreground" />
      <p className="text-sm font-semibold">Timeline view coming soon</p>
      <p className="max-w-xs text-xs text-muted-foreground">
        A driver-by-driver gantt of today&apos;s shipments will land in a follow-up phase. Switch
        back to Table to keep working.
      </p>
    </div>
  );
}
