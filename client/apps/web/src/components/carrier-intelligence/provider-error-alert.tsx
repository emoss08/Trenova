import { useT } from "@trenova/shared/i18n/use-t";
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
} from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { classifyCarrierIntelError } from "@/lib/carrier-intelligence";
import { OctagonAlertIcon, RefreshCwIcon, ShieldOffIcon, TimerIcon } from "lucide-react";
import type { ReactNode } from "react";

export type ProviderErrorAlertProps = {
  error: unknown;
  title: string;
  onRetry?: () => void;
  action?: ReactNode;
  className?: string;
};

export function ProviderErrorAlert({
  error,
  title,
  onRetry,
  action,
  className,
}: ProviderErrorAlertProps) {
  const t = useT();
  const kind = classifyCarrierIntelError(error);
  const message = graphQLErrorMessage(error, t("Something went wrong. Try again in a moment."));

  const heading =
    kind === "rate-limit"
      ? t("The provider is rate limiting requests")
      : kind === "forbidden"
        ? t("You do not have access to this")
        : title;

  const description =
    kind === "rate-limit"
      ? t("Wait a moment, then try again.")
      : kind === "forbidden"
        ? t("Ask an administrator for the carrier sourcing or carrier intelligence permission.")
        : message;

  const Icon =
    kind === "rate-limit" ? TimerIcon : kind === "forbidden" ? ShieldOffIcon : OctagonAlertIcon;

  return (
    <Alert
      variant={kind === "business" || kind === "rate-limit" ? "warning" : "destructive"}
      className={className}
      data-error-kind={kind}
    >
      <Icon />
      <AlertTitle>{heading}</AlertTitle>
      <AlertDescription>{description}</AlertDescription>
      {onRetry || action ? (
        <AlertAction className="flex items-center gap-2">
          {action}
          {onRetry && kind !== "forbidden" ? (
            <Button type="button" size="sm" variant="outline" onClick={onRetry}>
              <RefreshCwIcon className="size-3.5" />
              {t("Try again")}
            </Button>
          ) : null}
        </AlertAction>
      ) : null}
    </Alert>
  );
}
