import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";

/** Pay that could not be read, said plainly: hiding it would read as having none. */
export function PayLoadFailed({ message, onRetry }: { message: string; onRetry: () => void }) {
  const t = useT();

  return (
    <div
      role="alert"
      className="flex items-center justify-between gap-3 rounded-2xl border border-border bg-card p-4"
    >
      <p className="text-sm text-muted-foreground">{message}</p>
      <Button variant="outline" size="sm" onClick={onRetry}>
        {t("Try again")}
      </Button>
    </div>
  );
}
