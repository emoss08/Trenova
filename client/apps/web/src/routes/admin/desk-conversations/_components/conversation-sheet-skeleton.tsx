import { useT } from "@trenova/shared/i18n/use-t";

const FACTS = ["person", "agent", "origin", "messages", "started", "last"] as const;
const TURNS = ["a", "b", "c"] as const;

/** The opened conversation's own layout while it loads: the notice, its facts, and turns. */
export function ConversationSheetSkeleton() {
  const t = useT();

  return (
    <div
      aria-busy="true"
      aria-label={t("Opening the conversation")}
      className="flex min-h-0 flex-1 flex-col gap-4 px-4 pb-4"
    >
      <div className="ui-shimmer h-8 w-full rounded-md" />
      <div className="grid grid-cols-2 gap-x-6 gap-y-3 sm:grid-cols-3">
        {FACTS.map((fact) => (
          <div key={fact} className="flex flex-col gap-1">
            <div className="ui-shimmer h-3 w-16 rounded" />
            <div className="ui-shimmer h-4 w-28 rounded" />
          </div>
        ))}
      </div>
      <div className="flex flex-col gap-4">
        {TURNS.map((turn) => (
          <div key={turn} className="flex flex-col gap-2">
            <div className="ui-shimmer h-12 w-full rounded-md" />
            <div className="ui-shimmer h-3 w-32 rounded" />
            <div className="ui-shimmer h-4 w-5/6 rounded" />
            <div className="ui-shimmer h-4 w-2/3 rounded" />
          </div>
        ))}
      </div>
    </div>
  );
}
