import type { ReactNode } from "react";

/** A choice row's name over one muted line of detail, each cut short to the row's width. */
export function OptionStack({ primary, secondary }: { primary: ReactNode; secondary?: ReactNode }) {
  return (
    <div className="flex size-full min-w-0 flex-col items-start">
      <span className="w-full truncate">{primary}</span>
      {secondary ? (
        <span className="text-2xs text-muted-foreground w-full truncate">{secondary}</span>
      ) : null}
    </div>
  );
}
