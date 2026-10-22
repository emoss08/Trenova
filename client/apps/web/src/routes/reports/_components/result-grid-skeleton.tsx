import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const ROWS = 12;

/** A result grid while the rules module that formats its cells loads: a header and rows. */
export function ResultGridSkeleton({ density = "default" }: { density?: "compact" | "default" }) {
  const t = useT();
  const height = density === "compact" ? "h-5" : "h-6";
  return (
    <div
      aria-busy="true"
      aria-label={t("Loading the results…")}
      className="flex min-h-0 flex-1 flex-col gap-1.5 p-3"
    >
      <Skeleton aria-hidden="true" className="h-5 w-full" />
      {Array.from({ length: ROWS }, (_, index) => (
        <Skeleton
          key={index}
          aria-hidden="true"
          className={height}
          style={{ opacity: 1 - index * 0.06 }}
        />
      ))}
    </div>
  );
}
