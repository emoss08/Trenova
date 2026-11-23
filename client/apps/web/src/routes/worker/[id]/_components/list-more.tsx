import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

/** What the end of a paged list needs from `useWorkerList`. */
export type ListMore = {
  hasMore: boolean;
  loadMore: () => void;
  loadingMore: boolean;
  loadMoreFailed: boolean;
  /** Why the next page failed. */
  error: unknown;
};

/**
 * The line under a paged list while more pages exist: a shimmer at a fixed height while the next
 * page is in flight (nothing moves when it lands), and the failure with a retry when it did not
 * come. It draws nothing once the list is whole.
 */
export function ListMoreLine({ more }: { more: ListMore }) {
  const t = useT();
  if (more.loadMoreFailed) {
    return (
      <div className="px-2 py-1.5">
        <ErrorState
          layout="compact"
          error={more.error}
          title={t("More could not be loaded")}
          onRetry={more.loadMore}
        />
      </div>
    );
  }
  if (!more.hasMore) return null;
  return (
    <div
      aria-busy={more.loadingMore}
      aria-label={more.loadingMore ? t("Loading more") : undefined}
      className="flex h-9 items-center px-4"
    >
      <span
        aria-hidden
        className={cn(
          "block h-2 w-32 rounded-sm",
          more.loadingMore ? "ui-shimmer" : "bg-transparent",
        )}
      />
    </div>
  );
}

/** How many rows the server returns at most for a list it reads whole (credentials, checklists). */
export const SERVER_LIST_CAP = 100;

/**
 * The line under a list the server reads whole and caps: drawn only when the cap was reached, so
 * the reader knows older rows exist. Leave it out where a summary count gives the real total.
 */
export function ListCapLine({ count, cap = SERVER_LIST_CAP }: { count: number; cap?: number }) {
  const t = useT();
  if (count < cap) return null;
  return (
    <p className="px-4 py-2 text-xs text-muted-foreground">{t("Showing the newest {0}", cap)}</p>
  );
}
