import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { useT } from "@trenova/shared/i18n/use-t";

/**
 * What a virtualised, paged list draws after its last row: a shimmer line while the next page is
 * on its way, or, when that page failed, the failure with its retry. Nothing otherwise.
 */
export function LoadMoreFooter({
  loading,
  failed,
  error,
  failedTitle,
  onRetry,
}: {
  loading: boolean;
  failed: boolean;
  error: unknown;
  failedTitle: string;
  onRetry: () => void;
}) {
  const t = useT();
  if (failed) {
    return <ErrorState error={error} layout="compact" title={failedTitle} onRetry={onRetry} />;
  }
  if (!loading) return null;
  return (
    <div aria-busy aria-label={t("Loading more")} className="px-3 py-2.5">
      <span aria-hidden className="block h-2 w-32 ui-shimmer rounded-sm" />
    </div>
  );
}
