import type { AIRetrievalAvailability } from "@/lib/graphql/ai-retrieval";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { CircleAlertIcon, InfoIcon } from "lucide-react";
import { ENABLE_VECTOR_COMMAND, availabilityNotice } from "./retrieval-model";

type RetrievalNoticeProps = {
  availability: AIRetrievalAvailability;
  onOpenProviders: () => void;
  onOpenSettings: () => void;
};

/**
 * Says why agents are searching by keyword only, and the one thing that
 * fixes it: a command on the server, a route on the Providers tab, or a
 * setting below. Nothing is shown while search by meaning works.
 */
export function RetrievalNotice({
  availability,
  onOpenProviders,
  onOpenSettings,
}: RetrievalNoticeProps) {
  const t = useT();
  const notice = availabilityNotice(availability);

  if (!notice) {
    return null;
  }

  return (
    <Alert size="sm" variant={notice.variant} data-testid="retrieval-notice">
      {notice.variant === "info" ? <InfoIcon /> : <CircleAlertIcon />}
      <AlertTitle>{t(notice.title)}</AlertTitle>
      <AlertDescription className="flex flex-col items-start gap-1.5">
        <span>{t(notice.message)}</span>
        {notice.fix === "command" ? (
          <code className="bg-sunken rounded-sm px-1.5 py-0.5 font-mono text-xs select-all">
            {ENABLE_VECTOR_COMMAND}
          </code>
        ) : null}
        {notice.fix === "providers" ? (
          <Button variant="link" size="xxs" className="h-auto px-0" onClick={onOpenProviders}>
            {t("Open Providers")}
          </Button>
        ) : null}
        {notice.fix === "settings" ? (
          <Button variant="link" size="xxs" className="h-auto px-0" onClick={onOpenSettings}>
            {t("Go to the settings")}
          </Button>
        ) : null}
        {availability.extensionVersion && notice.fix === "command" ? (
          <span className="text-muted-foreground">
            {t("Installed pgvector: {0}", availability.extensionVersion)}
          </span>
        ) : null}
      </AlertDescription>
    </Alert>
  );
}
