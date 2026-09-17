import { classifyCarrierIntelError } from "@/lib/carrier-intelligence";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { cn } from "@trenova/shared/lib/utils";
import { RefreshCwIcon } from "lucide-react";

export type IntelInlineErrorProps = {
  error: unknown;
  title: string;
  onRetry?: () => void;
  className?: string;
};

export function IntelInlineError({ error, title, onRetry, className }: IntelInlineErrorProps) {
  const t = useT();
  const kind = classifyCarrierIntelError(error);
  const message =
    kind === "rate-limit"
      ? t("The provider is rate limiting requests. Wait a moment, then try again.")
      : kind === "forbidden"
        ? t("You don't have access to this. Ask an administrator for carrier intelligence access.")
        : graphQLErrorMessage(error, t("Something went wrong. Try again in a moment."));

  return (
    <div
      role="alert"
      className={cn(
        "flex flex-wrap items-center justify-between gap-3 rounded-lg border px-4 py-3",
        className,
      )}
      data-error-kind={kind}
    >
      <div className="flex min-w-0 flex-col gap-0.5">
        <span className="text-sm font-medium">{title}</span>
        <span className="text-muted-foreground text-xs">{message}</span>
      </div>
      {onRetry && kind !== "forbidden" ? (
        <Button type="button" variant="ghost" size="sm" onClick={onRetry}>
          <RefreshCwIcon className="size-3.5" aria-hidden />
          {t("Try again")}
        </Button>
      ) : null}
    </div>
  );
}
