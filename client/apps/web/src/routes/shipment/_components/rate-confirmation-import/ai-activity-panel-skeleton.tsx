import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { NovaPaneSkeleton } from "@/components/tariff-studio/studio-skeletons";

/**
 * The import assistant while its code is read: the page assistant's header (the agent's name
 * over the required-fields progress) and the conversation's outline over the composer, the same
 * outline the formula studio's assistant pane draws, since both are the page assistant.
 */
export function AIActivityPanelSkeleton() {
  const t = useT();

  return (
    <div
      className="flex h-full min-h-0 flex-col border-l"
      aria-busy
      aria-label={t("Loading the assistant")}
    >
      <div className="contents" aria-hidden>
        <div className="flex shrink-0 flex-col gap-2 border-b px-3 py-2.5">
          <div className="text-sm">
            <Skeleton className="inline-block h-3 w-28 align-middle" />
          </div>
          <div className="flex flex-col gap-1">
            <div className="text-2xs">
              <Skeleton className="inline-block h-2 w-24 align-middle" />
            </div>
            <Skeleton className="h-0.5 w-full rounded-full" />
          </div>
        </div>
        <div className="flex min-h-0 flex-1 flex-col">
          <NovaPaneSkeleton />
        </div>
      </div>
    </div>
  );
}
