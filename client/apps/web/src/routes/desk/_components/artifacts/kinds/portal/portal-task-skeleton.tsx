import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

const ANSWER_LINES = ["w-11/12", "w-4/5", "w-2/3"] as const;
const PAGE_LINES = ["w-3/5", "w-1/2"] as const;

/** A portal task while it is read: its standing and site, the question, the answer, the pages. */
export function PortalTaskSkeleton() {
  const t = useT();

  return (
    <div
      className="dk-ax-pad flex flex-col gap-4"
      aria-busy
      aria-label={t("Loading the portal task")}
    >
      <div className="flex items-center gap-2" aria-hidden>
        <i className="dk-sk h-5 w-20 rounded-badge" />
        <i className="dk-sk h-4 w-40" />
        <i className="dk-sk ml-auto h-3 w-24" />
      </div>
      <i className="dk-sk h-4 w-3/4" aria-hidden />
      <div className="flex flex-col gap-1.5" aria-hidden>
        <i className="dk-sk h-3 w-28" />
        {ANSWER_LINES.map((line) => (
          <i key={line} className={cn("dk-sk h-4", line)} />
        ))}
      </div>
      <div className="flex flex-col gap-1" aria-hidden>
        <i className="dk-sk h-3 w-24" />
        {PAGE_LINES.map((line) => (
          <i key={line} className={cn("dk-sk h-3", line)} />
        ))}
      </div>
    </div>
  );
}
