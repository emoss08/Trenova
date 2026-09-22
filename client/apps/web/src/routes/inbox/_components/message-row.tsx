import type { InboundMessageRow } from "@/lib/graphql/inbox";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatDurationFromSeconds, formatUnixTime } from "@trenova/shared/lib/date";
import { cn, getNameInitials } from "@trenova/shared/lib/utils";
import { PaperclipIcon, TruckIcon } from "lucide-react";
import { forwardRef } from "react";
import { CLASSIFICATION_ICON, CLASSIFICATION_VARIANT, classificationLabel } from "./classification";

/** How long a message may wait before its age is worth raising. */
export const WAITING_TOO_LONG_SECONDS = 4 * 3600;

export function senderName(message: Pick<InboundMessageRow, "fromName" | "fromAddress">): string {
  return message.fromName.trim() === "" ? message.fromAddress : message.fromName;
}

/**
 * One piece of mail, read at a glance: who, what about, the first words, and
 * what the desk made of it.
 *
 * A message waiting on a person carries a dot and a heavier sender, which is
 * what "unread" means here — not whether somebody opened it, but whether
 * somebody still has to do something. How long it has waited is shown once
 * it is long enough to matter.
 */
export const InboxMessageRow = forwardRef<
  HTMLButtonElement,
  {
    message: InboundMessageRow;
    active: boolean;
    now: number;
    onOpen: (id: string) => void;
  }
>(function InboxMessageRow({ message, active, now, onOpen }, ref) {
  const t = useT();
  const classification = message.classification ?? null;
  const KindIcon = classification === null ? null : CLASSIFICATION_ICON[classification];
  const waitingFor = now - message.receivedAt;
  const quarantined = message.status === "Quarantined";

  return (
    <li className="relative">
      <button
        ref={ref}
        type="button"
        onClick={() => onOpen(message.id)}
        aria-current={active ? "true" : undefined}
        className={cn(
          "ui-focus-ring border-border-subtle flex w-full items-start gap-3 border-b py-3 pr-4 pl-5 text-left transition-colors",
          active ? "bg-surface-selected" : "hover:bg-surface-hover",
        )}
      >
        {active && <span aria-hidden className="bg-brand absolute inset-y-0 left-0 w-0.5" />}
        {message.needsReview && (
          <span
            aria-hidden
            className={cn(
              "absolute top-[1.35rem] left-2 size-1.5 rounded-full",
              quarantined ? "bg-danger" : "bg-warning",
            )}
          />
        )}

        <span
          aria-hidden
          className="bg-muted text-foreground-muted flex size-8 shrink-0 items-center justify-center rounded-md text-xs font-medium"
        >
          {getNameInitials(senderName(message), "?")}
        </span>

        <span className="flex min-w-0 flex-1 flex-col gap-0.5">
          <span className="flex min-w-0 items-baseline gap-2">
            <span
              className={cn(
                "min-w-0 flex-1 truncate text-sm",
                message.needsReview ? "text-foreground font-medium" : "text-foreground-muted",
              )}
            >
              {senderName(message)}
            </span>
            <span className="text-foreground-subtle shrink-0 text-xs tabular-nums">
              {formatUnixTime(message.receivedAt)}
            </span>
          </span>

          <span
            className={cn(
              "truncate text-sm",
              message.needsReview ? "text-foreground" : "text-foreground-muted",
            )}
          >
            {message.subject === "" ? t("(no subject)") : message.subject}
          </span>

          {message.preview !== "" && (
            <span className="text-foreground-subtle line-clamp-1 text-xs">{message.preview}</span>
          )}

          <span className="flex min-w-0 flex-wrap items-center gap-1.5 pt-1">
            {classification !== null && KindIcon !== null && (
              <Badge variant={CLASSIFICATION_VARIANT[classification]} className="gap-1">
                <KindIcon className="size-3" aria-hidden />
                {classificationLabel(t, classification)}
              </Badge>
            )}
            {quarantined && (
              <Badge variant="danger" appearance="outline">
                {t("Held back")}
              </Badge>
            )}
            {message.matchedShipment && (
              <Badge variant="neutral" appearance="outline" className="gap-1">
                <TruckIcon className="size-3" aria-hidden />
                {message.matchedShipment.proNumber}
              </Badge>
            )}
            {message.attachmentCount > 0 && (
              <span className="text-foreground-subtle flex items-center gap-0.5 text-xs">
                <PaperclipIcon className="size-3" aria-hidden />
                <span className="tabular-nums">{message.attachmentCount}</span>
                <span className="sr-only">{t("attachments")}</span>
              </span>
            )}
            {message.needsReview && waitingFor >= WAITING_TOO_LONG_SECONDS && (
              <span className="text-warning ml-auto text-xs tabular-nums">
                {t("Waiting {0}", formatDurationFromSeconds(waitingFor))}
              </span>
            )}
          </span>
        </span>
      </button>
    </li>
  );
});
