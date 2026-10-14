import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { useT } from "@trenova/shared/i18n/use-t";
import type { BacktestResponse } from "@trenova/shared/types/formula-template";

/** Says a re-rating run stopped at its time limit, so its figures cover only the rows it reached. */
export function BacktestTruncatedNote({ data }: { data: BacktestResponse }) {
  const t = useT();

  if (!data.truncated) return null;

  return (
    <Alert size="sm" variant="warning">
      <AlertDescription>
        {t(
          "{0, plural, one {The run reached its time limit after # shipment; the figures cover only that one.} other {The run reached its time limit after # shipments; the figures cover only those.}}",
          data.results.length,
        )}
      </AlertDescription>
    </Alert>
  );
}
