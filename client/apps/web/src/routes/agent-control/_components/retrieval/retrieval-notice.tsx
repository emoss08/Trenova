import type { AIRetrievalAvailability } from "@/lib/graphql/ai-retrieval";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Ic } from "../kit/ic";
import { ENABLE_VECTOR_COMMAND, availabilityNotice } from "./retrieval-model";
import { Button } from "@trenova/shared/components/ui/button";

/** Reasons Nova's sentence and its control already speak to. */
const SAID_ABOVE = new Set(["NoProvider", "Disabled", "BudgetPaused"]);

type RetrievalNoticeProps = {
  availability: AIRetrievalAvailability;
  onOpenProviders: () => void;
};

/**
 * Why agents are searching by keyword when the reason is outside this tab: a command
 * on the server, or a provider that is failing. Nothing is shown while search by meaning
 * works, or when Nova's sentence already says why it does not.
 */
export function RetrievalNotice({ availability, onOpenProviders }: RetrievalNoticeProps) {
  const t = useT();
  const notice = availabilityNotice(availability);

  if (!notice || (availability.reason && SAID_ABOVE.has(availability.reason))) {
    return null;
  }

  return (
    <div
      className={cn(
        "bnr",
        notice.variant === "destructive" ? "d" : notice.variant === "warning" && "w",
      )}
      role={notice.variant === "info" ? "status" : "alert"}
      data-testid="retrieval-notice"
    >
      <Ic n={notice.variant === "info" ? "sparkle" : "alert"} s={14} />
      <span>
        <b>{t(notice.title)}</b> {t(notice.message)}
        {notice.fix === "command" && (
          <>
            {" "}
            <code className="mono">{ENABLE_VECTOR_COMMAND}</code>
            {availability.extensionVersion &&
              ` · ${t("Installed pgvector: {0}", availability.extensionVersion)}`}
          </>
        )}
      </span>
      {notice.fix === "providers" && (
        <Button type="button" variant="outline" size="sm" onClick={onOpenProviders}>
          {t("Open Providers")}
        </Button>
      )}
    </div>
  );
}
