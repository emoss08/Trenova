import { useT } from "@trenova/shared/i18n/use-t";

const TURNS = ["a", "b", "c"] as const;

/**
 * The opened conversation's body while it loads: the recorded-look notice,
 * then questions and the agent's turns in the transcript's own columns, the
 * time in the gutter, the folded working line and the answer's lines.
 */
export function ConversationSheetSkeleton() {
  const t = useT();

  return (
    <div
      aria-busy="true"
      aria-label={t("Opening the conversation")}
      className="flex min-h-0 flex-1 flex-col gap-6 overflow-hidden pt-4 pb-8"
    >
      <div className="px-4 sm:px-5">
        <div className="ui-shimmer h-8 w-full rounded-md" />
      </div>
      <div className="grid grid-cols-[minmax(0,1fr)] gap-y-5 px-4 sm:grid-cols-[4rem_minmax(0,1fr)] sm:gap-x-4 sm:pr-5 sm:pl-5">
        {TURNS.map((turn) => (
          <div key={turn} className="contents">
            <div className="max-sm:hidden" />
            <div className="ui-shimmer h-5 w-2/3 rounded" />
            <div className="flex justify-end max-sm:hidden">
              <div className="ui-shimmer h-3 w-10 rounded" />
            </div>
            <div className="flex min-w-0 flex-col gap-2">
              <div className="ui-shimmer h-3 w-36 rounded" />
              <div className="ui-shimmer h-4 w-full rounded" />
              <div className="ui-shimmer h-4 w-5/6 rounded" />
              <div className="ui-shimmer h-4 w-2/3 rounded" />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
