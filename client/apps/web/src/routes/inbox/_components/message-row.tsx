import type { InboundMessageRow } from "@/lib/graphql/inbox";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { PaperclipIcon } from "lucide-react";
import {
  CLASSIFICATION_VARIANT,
  STATUS_VARIANT,
  classificationLabel,
  statusLabel,
} from "./classification";

/**
 * One piece of mail, as a row rather than a card.
 *
 * What a person scans for is who it is from, what it is, and how long it has
 * been sitting there. The match and the reason for it live in the detail,
 * because a reason worth reading is longer than a row.
 */
export function InboxMessageRow({
  message,
  active,
  now,
  onOpen,
}: {
  message: InboundMessageRow;
  active: boolean;
  now: number;
  onOpen: (message: InboundMessageRow) => void;
}) {
  const t = useT();

  return (
    <li>
      <button
        type="button"
        onClick={() => onOpen(message)}
        aria-current={active ? "true" : undefined}
        className={cn(
          "ui-focus-ring border-border flex w-full items-start gap-3 border-b px-4 py-3 text-left transition-colors",
          active ? "bg-surface-hover" : "hover:bg-surface-hover",
        )}
      >
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <div className="flex min-w-0 items-baseline gap-2">
            <span
              className={cn(
                "min-w-0 truncate text-sm",
                message.needsReview && "font-medium",
              )}
            >
              {message.subject === "" ? t("(no subject)") : message.subject}
            </span>
            {message.failureText !== "" && (
              <Badge variant="danger" appearance="outline">
                {t("Could not be read")}
              </Badge>
            )}
          </div>

          <div className="text-muted-foreground flex min-w-0 items-center gap-1.5 text-xs">
            <span className="min-w-0 truncate">
              {message.fromName === "" ? message.fromAddress : message.fromName}
            </span>
            <span aria-hidden>·</span>
            <span className="tabular-nums">{formatSecondsAgo(now - message.receivedAt)}</span>
          </div>

          {message.matchReason !== "" && (
            <p className="text-muted-foreground line-clamp-1 text-xs">{message.matchReason}</p>
          )}
        </div>

        <div className="flex shrink-0 items-center gap-1.5">
          {message.classification !== null && message.classification !== undefined && (
            <Badge variant={CLASSIFICATION_VARIANT[message.classification]}>
              {classificationLabel(t, message.classification)}
            </Badge>
          )}
          <Badge variant={STATUS_VARIANT[message.status]} appearance="outline">
            {statusLabel(t, message.status)}
          </Badge>
        </div>
      </button>
    </li>
  );
}

/** A file came with it. Shown only when there is one, so the row stays quiet. */
export function AttachmentMark({ count }: { count: number }) {
  const t = useT();

  if (count === 0) {
    return null;
  }

  return (
    <span className="text-muted-foreground flex items-center gap-1 text-xs">
      <PaperclipIcon className="size-3" aria-hidden />
      <span className="tabular-nums">{count}</span>
      <span className="sr-only">{t("attachments")}</span>
    </span>
  );
}
