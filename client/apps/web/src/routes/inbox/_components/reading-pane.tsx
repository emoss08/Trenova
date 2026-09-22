import { DocumentIntelligenceDialog } from "@/components/documents/document-intelligence-dialog";
import { SectionPanel } from "@/components/section-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import type { InboundMessageDetail } from "@/lib/graphql/inbox";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo, formatUnixDateTime } from "@trenova/shared/lib/date";
import { getNameInitials } from "@trenova/shared/lib/utils";
import type { Document } from "@trenova/shared/types/document";
import {
  ArchiveIcon,
  ArrowLeftIcon,
  CheckIcon,
  Link2Icon,
  MessageSquareIcon,
  PaperclipIcon,
  XIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { AttachmentList } from "./attachment-list";
import { STATUS_VARIANT, statusLabel } from "./classification";
import { DeskReading } from "./desk-reading";
import { LinkMessageDialog } from "./link-dialog";
import { MessageTrail } from "./message-trail";
import { senderName } from "./message-row";
import { suggestNextStep } from "./next-step";
import { splitQuotedBody } from "./quoted-body";
import type { ReviewOutcome } from "./use-inbox-actions";

export type ReadingPaneActions = {
  review: (args: { id: string; status: ReviewOutcome; note: string }) => void;
  ask: (message: InboundMessageDetail) => void;
  canAsk: boolean;
  busy: boolean;
  refresh: () => Promise<void>;
};

function ToolbarAction({
  label,
  shortcut,
  icon: Icon,
  onClick,
  disabled,
  variant = "ghost",
}: {
  label: string;
  shortcut: string;
  icon: typeof CheckIcon;
  onClick: () => void;
  disabled?: boolean;
  variant?: "default" | "ghost" | "outline";
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button size="sm" variant={variant} onClick={onClick} disabled={disabled}>
            <Icon className="size-3.5" />
            <span className="hidden @3xl:inline">{label}</span>
          </Button>
        }
      />
      <TooltipContent className="flex items-center gap-2">
        {label}
        <Kbd>{shortcut}</Kbd>
      </TooltipContent>
    </Tooltip>
  );
}

/**
 * One message, read the way mail is read: the sender and subject at the top,
 * what the desk made of it beside the words, the words themselves with the
 * thread they replied to folded away, the files, and how it got here.
 *
 * The toolbar is the triage: handle, ignore, link, ask — each with the key
 * that does the same from anywhere in the inbox. A note written at the foot
 * rides with whichever of the first two is chosen.
 */
export function ReadingPane({
  messageId,
  now,
  actions,
  linkOpen,
  onLinkOpenChange,
  onClose,
}: {
  messageId: string;
  now: number;
  actions: ReadingPaneActions;
  linkOpen: boolean;
  onLinkOpenChange: (open: boolean) => void;
  onClose: () => void;
}) {
  const t = useT();
  const [note, setNote] = useState("");
  const [showQuoted, setShowQuoted] = useState(false);
  const [readingDocument, setReadingDocument] = useState<Document | null>(null);

  const messageQuery = useQuery(queries.inbox.message(messageId));
  const message = messageQuery.data;

  const body = useMemo(() => splitQuotedBody(message?.textBody ?? ""), [message?.textBody]);
  const step = useMemo(
    () => (message === undefined ? ({ kind: "none" } as const) : suggestNextStep(message)),
    [message],
  );

  const attachMutation = useApiMutation({
    mutationFn: ({ documentId, shipmentId }: { documentId: string; shipmentId: string }) =>
      apiService.documentService.attachToShipment(documentId, shipmentId),
    onSuccess: async () => {
      toast.success(t("Filed on the shipment"));
      await actions.refresh();
    },
    resourceName: "Document",
  });

  const reviewDocumentMutation = useApiMutation({
    mutationFn: (documentId: string) => apiService.documentService.getById(documentId),
    onSuccess: (document) => setReadingDocument(document),
    resourceName: "Document",
  });

  if (messageQuery.isLoading) {
    return (
      <div className="flex flex-col gap-4 p-6" aria-busy="true">
        <Skeleton className="h-6 w-2/3" />
        <div className="flex gap-3">
          <Skeleton className="size-10 rounded-md" />
          <div className="flex flex-1 flex-col gap-1.5">
            <Skeleton className="h-4 w-1/3" />
            <Skeleton className="h-3 w-1/2" />
          </div>
        </div>
        <Skeleton className="h-40" />
        <Skeleton className="h-32" />
      </div>
    );
  }

  if (message === undefined) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-3 p-6 text-center">
        <p className="text-sm">{t("This message could not be opened.")}</p>
        <p className="text-foreground-subtle max-w-sm text-xs">
          {t(
            "It may have been removed by the retention sweep, or it belongs to another organization.",
          )}
        </p>
        <Button size="sm" variant="outline" onClick={onClose}>
          {t("Back to the list")}
        </Button>
      </div>
    );
  }

  const busy = actions.busy || attachMutation.isPending;
  const review = (status: ReviewOutcome) => {
    actions.review({ id: message.id, status, note });
    setNote("");
  };
  const handlers = {
    onReviewDraft: (documentId: string) => reviewDocumentMutation.mutate(documentId),
    onAttach: (documentId: string, shipmentId: string) =>
      attachMutation.mutate({ documentId, shipmentId }),
    onAsk: () => actions.ask(message),
    onLink: () => onLinkOpenChange(true),
    busy,
    canAsk: actions.canAsk,
  };
  const recipients = [...message.toAddresses, ...message.ccAddresses];

  return (
    <div className="@container flex min-h-0 flex-1 flex-col">
      <div className="border-border flex h-11 shrink-0 items-center gap-1 overflow-x-auto border-b px-3">
        <Button size="sm" variant="ghost" onClick={onClose} aria-label={t("Back to the list")}>
          <ArrowLeftIcon className="size-3.5" />
          <span className="lg:hidden">{t("Back")}</span>
        </Button>
        <div className="flex-1" />
        {message.needsReview && (
          <>
            <ToolbarAction
              label={t("Mark handled")}
              shortcut="e"
              icon={CheckIcon}
              variant="default"
              disabled={busy}
              onClick={() => review("Actioned")}
            />
            <ToolbarAction
              label={t("Ignore")}
              shortcut="#"
              icon={ArchiveIcon}
              disabled={busy}
              onClick={() => review("Ignored")}
            />
          </>
        )}
        <ToolbarAction
          label={t("Link by hand")}
          shortcut="l"
          icon={Link2Icon}
          disabled={busy}
          onClick={() => onLinkOpenChange(true)}
        />
        <ToolbarAction
          label={t("Ask the desk")}
          shortcut="a"
          icon={MessageSquareIcon}
          disabled={busy || !actions.canAsk}
          onClick={() => actions.ask(message)}
        />
        <Button
          size="icon-sm"
          variant="ghost"
          className="hidden lg:inline-flex"
          aria-label={t("Close")}
          onClick={onClose}
        >
          <XIcon className="size-4" />
        </Button>
      </div>

      <ScrollArea className="min-h-0 flex-1">
        <article className="mx-auto flex w-full max-w-3xl flex-col gap-5 px-4 pt-5 pb-24 @lg:px-6">
          <header className="flex flex-col gap-3">
            <div className="flex flex-col-reverse items-start gap-2 @lg:flex-row @lg:justify-between @lg:gap-3">
              <h1 className="min-w-0 text-lg font-semibold text-balance break-words">
                {message.subject === "" ? t("(no subject)") : message.subject}
              </h1>
              <Badge
                variant={STATUS_VARIANT[message.status]}
                appearance="outline"
                className="shrink-0 @lg:mt-1"
              >
                {statusLabel(t, message.status)}
              </Badge>
            </div>

            <div className="flex flex-wrap items-start gap-3">
              <span
                aria-hidden
                className="bg-muted text-foreground-muted flex size-10 shrink-0 items-center justify-center rounded-md text-sm font-medium"
              >
                {getNameInitials(senderName(message), "?")}
              </span>
              <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                <div className="flex min-w-0 flex-wrap items-baseline gap-x-2">
                  <span className="text-sm font-medium">{senderName(message)}</span>
                  {message.fromName.trim() !== "" && (
                    <span className="text-foreground-subtle truncate text-xs">
                      {message.fromAddress}
                    </span>
                  )}
                </div>
                {recipients.length > 0 && (
                  <span
                    className="text-foreground-subtle truncate text-xs"
                    title={recipients.join(", ")}
                  >
                    {t("To {0}", recipients.join(", "))}
                  </span>
                )}
              </div>
              <div className="flex basis-full flex-row gap-2 @lg:basis-auto @lg:flex-col @lg:items-end @lg:gap-0.5">
                <span className="text-foreground-muted text-xs tabular-nums">
                  {formatUnixDateTime(message.receivedAt)}
                </span>
                <span className="text-foreground-subtle text-xs tabular-nums">
                  {formatSecondsAgo(now - message.receivedAt)}
                </span>
              </div>
            </div>
          </header>

          {message.failureText !== "" && (
            <Alert size="sm" variant="destructive">
              <AlertDescription>{message.failureText}</AlertDescription>
            </Alert>
          )}
          {message.reviewNote !== "" && (
            <Alert size="sm" variant="info">
              <AlertDescription>{message.reviewNote}</AlertDescription>
            </Alert>
          )}

          <DeskReading message={message} step={step} handlers={handlers} />

          {/* The body is whatever a sender chose to write, so it is shown as
              text and never as markup. */}
          <section aria-label={t("The message")} className="flex flex-col gap-3">
            <p className="text-sm leading-relaxed whitespace-pre-wrap">
              {body.own === "" ? (
                <span className="text-foreground-subtle">{t("This message had no text.")}</span>
              ) : (
                body.own
              )}
            </p>
            {body.quoted !== "" && (
              <div className="flex flex-col gap-2">
                <Button
                  size="xs"
                  variant="secondary"
                  className="w-fit"
                  aria-expanded={showQuoted}
                  onClick={() => setShowQuoted((value) => !value)}
                >
                  {showQuoted ? t("Hide the earlier thread") : t("Show the earlier thread")}
                </Button>
                {showQuoted && (
                  <p className="text-foreground-subtle border-border border-l-2 pl-3 text-sm leading-relaxed whitespace-pre-wrap">
                    {body.quoted}
                  </p>
                )}
              </div>
            )}
          </section>

          {message.attachments.length > 0 && (
            <section aria-label={t("Attachments")} className="flex flex-col gap-2">
              <h2 className="text-foreground-subtle flex items-center gap-1.5 text-xs font-medium">
                <PaperclipIcon className="size-3.5" aria-hidden />
                {t(
                  "{0, plural, one {# attachment} other {# attachments}}",
                  message.attachments.length,
                )}
              </h2>
              <AttachmentList
                attachments={message.attachments}
                shipment={message.matchedShipment ?? null}
                busy={busy}
                onReview={handlers.onReviewDraft}
                onAttach={handlers.onAttach}
              />
            </section>
          )}

          <SectionPanel title={t("How it got here")}>
            <div className="p-3">
              <MessageTrail message={message} />
            </div>
          </SectionPanel>

          {message.needsReview && (
            <section aria-label={t("Close it out")} className="flex flex-col gap-2">
              <Textarea
                value={note}
                onChange={(event) => setNote(event.target.value)}
                placeholder={t("Say what you did, so the next person does not repeat it")}
                rows={2}
                maxLength={2000}
              />
              <div className="flex justify-end gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  disabled={busy}
                  onClick={() => review("Ignored")}
                >
                  {t("Ignore")}
                </Button>
                <Button size="sm" disabled={busy} onClick={() => review("Actioned")}>
                  {t("Mark handled")}
                </Button>
              </div>
            </section>
          )}
        </article>
      </ScrollArea>

      <LinkMessageDialog
        message={message}
        open={linkOpen}
        onOpenChange={onLinkOpenChange}
        onLinked={actions.refresh}
      />
      <DocumentIntelligenceDialog
        open={readingDocument !== null}
        onOpenChange={(open) => !open && setReadingDocument(null)}
        document={readingDocument}
        resourceType="inbound_message"
        resourceId={message.id}
      />
    </div>
  );
}
