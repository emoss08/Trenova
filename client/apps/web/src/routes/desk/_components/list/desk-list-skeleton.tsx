import { DeskListStill } from "./desk-list";

const ROWS = 6;

/**
 * A list while it is read: rows in the list's own place and gutter, so the
 * head above it stays as it is and the rows arrive where these were.
 */
export function DeskListSkeleton({ label }: { label: string }) {
  return (
    <DeskListStill aria-busy aria-label={label}>
      {Array.from({ length: ROWS }, (_, index) => (
        <span key={index} className="m-1 block h-12 ui-shimmer rounded-[8px]" />
      ))}
    </DeskListStill>
  );
}
