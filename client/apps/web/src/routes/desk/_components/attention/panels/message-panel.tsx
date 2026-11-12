import type { WatchtowerMessagePanel } from "@/lib/graphql/watchtower";
import { Badge } from "@trenova/shared/components/ui/badge";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { getNameInitials } from "@trenova/shared/lib/utils";
import { formatFactTime } from "../../watchtower/lanes";
import { PanelBody, PanelCard } from "./panel-parts";

/**
 * A message that waits on a person, as mail reads: who sent it, what it is
 * about, what Nova took it for, and why it could not finish with it. The
 * message itself is read and answered in Triage.
 */
export function MessagePanel({
  message,
  now,
  timezone,
}: {
  message: WatchtowerMessagePanel;
  now: number;
  timezone: string;
}) {
  const t = useT();
  const sender = message.fromName || message.fromAddress || t("Unknown sender");

  return (
    <PanelCard
      eyebrow={
        message.receivedAt > 0
          ? t("Message · {0}", formatFactTime(t, message.receivedAt, now, timezone))
          : t("Message")
      }
      title={message.subject || t("(no subject)")}
      badge={
        message.held ? (
          <Badge variant="danger" dot>
            {t("Held back")}
          </Badge>
        ) : message.readAs !== "" ? (
          <Badge variant="info">{message.readAs}</Badge>
        ) : undefined
      }
    >
      <PanelBody className="flex flex-col gap-2.5">
        <div className="flex items-center gap-2.5">
          <span
            aria-hidden
            className="grid size-6 flex-none place-items-center rounded-full border border-dsk-b-sub bg-dsk-sunken text-3xs font-medium text-dsk-muted"
          >
            {getNameInitials(sender, "?")}
          </span>
          <span className="flex min-w-0 flex-col">
            <span className="truncate text-dsk-base text-dsk-fg">{sender}</span>
            {message.fromName !== "" && message.fromAddress !== "" && (
              <span className="truncate text-xs text-dsk-subtle">{message.fromAddress}</span>
            )}
          </span>
        </div>
        {message.why !== "" && (
          <p className="flex items-start gap-2 text-dsk-sm text-dsk-fg2">
            <NovaBadge size="xs" className="mt-0.5 flex-none" />
            <span>{message.why}</span>
          </p>
        )}
      </PanelBody>
    </PanelCard>
  );
}
