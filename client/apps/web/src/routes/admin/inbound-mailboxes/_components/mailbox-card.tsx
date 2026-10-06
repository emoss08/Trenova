import type { InboundMailbox } from "@/lib/graphql/inbox";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDate } from "@trenova/shared/lib/date";
import {
  Edit02Icon,
  Key01Icon,
  LockKeyholeCircleIcon,
  Mail01Icon,
  RefreshCw02Icon,
} from "@trenova/shared/components/icons";
import { Link } from "react-router";
import { reviewPolicyLabel } from "./mailbox-form-dialog";

/**
 * One address and what it is trusted with. A mailbox that cannot verify its
 * deliveries says so first, because until it can, every message sent to it
 * is refused and nothing else about it matters. A Resend mailbox without an
 * API key says so next: its mail lands, but without a body or attachments.
 */
export function MailboxCard({
  mailbox,
  canEdit,
  onEdit,
  onRotate,
  onSecret,
  onApiKey,
}: {
  mailbox: InboundMailbox;
  canEdit: boolean;
  onEdit: () => void;
  onRotate: () => void;
  onSecret: () => void;
  onApiKey: () => void;
}) {
  const t = useT();
  const listening = mailbox.status === "Active";
  const readsContent = mailbox.provider === "Resend";

  return (
    <article className="bg-card border-border flex flex-col gap-3 rounded-lg border p-4">
      <header className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-start gap-3">
          <span className="bg-muted text-foreground-muted flex size-9 shrink-0 items-center justify-center rounded-md">
            <Mail01Icon className="size-4" aria-hidden />
          </span>
          <div className="flex min-w-0 flex-col gap-0.5">
            <h3 className="truncate text-sm font-semibold">{mailbox.name}</h3>
            <span className="text-foreground-subtle truncate text-xs">{mailbox.address}</span>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-1.5">
          <Badge variant="neutral" appearance="outline">
            {mailbox.provider}
          </Badge>
          <Badge variant={listening ? "success" : "neutral"}>
            {listening ? t("Listening") : t("Not listening")}
          </Badge>
        </div>
      </header>

      {!mailbox.hasSigningSecret && listening && (
        <Alert size="sm" variant="warning">
          <AlertDescription>
            {t("No signing secret is set, so every delivery to this mailbox is refused.")}
          </AlertDescription>
        </Alert>
      )}

      {readsContent && !mailbox.hasApiKey && listening && (
        <Alert size="sm" variant="warning">
          <AlertDescription>
            {t(
              "No API key is set, so mail arrives without its body or attachments and waits for a person.",
            )}
          </AlertDescription>
        </Alert>
      )}

      <DescriptionList layout="inline">
        <DescriptionItem label={t("Trusted to")}>
          {reviewPolicyLabel(t, mailbox.reviewPolicy)}
          {mailbox.reviewPolicy === "ReviewBelowConfidence" &&
            ` · ${t("bar {0}%", Math.round(mailbox.minConfidence * 100))}`}
        </DescriptionItem>
        {mailbox.purpose !== "" && (
          <DescriptionItem label={t("Purpose")}>{mailbox.purpose}</DescriptionItem>
        )}
        <DescriptionItem label={t("Created")} numeric>
          {formatUnixDate(mailbox.createdAt)}
        </DescriptionItem>
      </DescriptionList>

      <footer className="flex flex-wrap items-center gap-1.5 pt-1">
        <Button
          size="sm"
          variant="ghost"
          nativeButton={false}
          render={<Link to={`/inbox?mailbox=${encodeURIComponent(mailbox.id)}`} />}
        >
          {t("Open its mail")}
        </Button>
        <div className="flex-1" />
        {canEdit && (
          <>
            <Button size="sm" variant="outline" onClick={onSecret}>
              <Key01Icon className="size-3.5" />
              {mailbox.hasSigningSecret ? t("Replace secret") : t("Set signing secret")}
            </Button>
            {readsContent && (
              <Button size="sm" variant="outline" onClick={onApiKey}>
                <LockKeyholeCircleIcon className="size-3.5" />
                {mailbox.hasApiKey ? t("Replace API key") : t("Set API key")}
              </Button>
            )}
            <Button size="sm" variant="outline" onClick={onRotate}>
              <RefreshCw02Icon className="size-3.5" />
              {t("Rotate URL")}
            </Button>
            <Button size="sm" variant="outline" onClick={onEdit}>
              <Edit02Icon className="size-3.5" />
              {t("Edit")}
            </Button>
          </>
        )}
      </footer>
    </article>
  );
}
