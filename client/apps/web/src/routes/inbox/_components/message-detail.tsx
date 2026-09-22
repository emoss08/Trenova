import { SectionPanel } from "@/components/section-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  reviewInboundMessage,
} from "@/lib/graphql/inbox";
import { queries } from "@/lib/queries";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { useCallback, useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";
import {
  CLASSIFICATION_VARIANT,
  STATUS_VARIANT,
  classificationLabel,
  statusLabel,
} from "./classification";

/**
 * One message, in full.
 *
 * The panel reads the message directly rather than taking the row the list
 * already holds: a list row needs a headline and a detail view needs the
 * record, and the two do not have to come from one query.
 */
export function InboxMessageDetail({
  messageId,
  onClose,
}: {
  messageId: string;
  onClose: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [note, setNote] = useState("");

  const messageQuery = useQuery(queries.inbox.message(messageId));
  const message = messageQuery.data;

  const refresh = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queries.inbox._def }),
      queryClient.invalidateQueries({ queryKey: queries.watchtower._def }),
    ]);
  }, [queryClient]);

  const reviewMutation = useApiMutation({
    mutationFn: (status: "Actioned" | "Ignored") =>
      reviewInboundMessage(messageId, {
        status,
        note: note.trim() === "" ? null : note.trim(),
      }),
    onSuccess: async (updated) => {
      toast.success(
        updated.status === "Actioned" ? t("Marked as handled") : t("Marked as ignored"),
      );
      setNote("");
      await refresh();
    },
    resourceName: "Message",
  });

  if (messageQuery.isLoading) {
    return (
      <div className="flex flex-col gap-3 p-4">
        <Skeleton className="h-6 w-2/3" />
        <Skeleton className="h-24" />
        <Skeleton className="h-32" />
      </div>
    );
  }

  if (message === undefined) {
    return (
      <div className="text-muted-foreground p-4 text-sm">
        {t("This message could not be opened.")}
      </div>
    );
  }

  const busy = reviewMutation.isPending;

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <header className="border-border flex items-start justify-between gap-3 border-b px-4 py-3">
        <div className="flex min-w-0 flex-col gap-1">
          <h2 className="truncate text-sm font-semibold">
            {message.subject === "" ? t("(no subject)") : message.subject}
          </h2>
          <p className="text-muted-foreground truncate text-xs">
            {message.fromName === ""
              ? message.fromAddress
              : `${message.fromName} <${message.fromAddress}>`}
          </p>
        </div>
        <Button size="sm" variant="ghost" onClick={onClose}>
          {t("Close")}
        </Button>
      </header>

      <ScrollArea className="min-h-0 flex-1">
        <div className="flex flex-col gap-4 p-4">
          {message.failureText !== "" && (
            <Alert size="sm" variant="destructive">
              <AlertDescription>{message.failureText}</AlertDescription>
            </Alert>
          )}

          {message.reviewNote !== "" && (
            <Alert size="sm">
              <AlertDescription>{message.reviewNote}</AlertDescription>
            </Alert>
          )}

          <SectionPanel title={t("What this is")}>
            <div className="p-3">
              <DescriptionList layout="inline">
                <DescriptionItem label={t("Status")}>
                  <Badge variant={STATUS_VARIANT[message.status]} appearance="outline">
                    {statusLabel(t, message.status)}
                  </Badge>
                </DescriptionItem>
                <DescriptionItem label={t("Read as")}>
                  {message.classification === null || message.classification === undefined ? (
                    <span className="text-muted-foreground">{t("Not read yet")}</span>
                  ) : (
                    <span className="flex items-center gap-2">
                      <Badge variant={CLASSIFICATION_VARIANT[message.classification]}>
                        {classificationLabel(t, message.classification)}
                      </Badge>
                      <span className="text-muted-foreground text-xs tabular-nums">
                        {t("{0} confident", `${Math.round(message.confidence * 100)}%`)}
                      </span>
                    </span>
                  )}
                </DescriptionItem>
                <DescriptionItem label={t("Arrived")} numeric>
                  {formatUnixDateTime(message.receivedAt)}
                </DescriptionItem>
                {message.mailbox !== null && message.mailbox !== undefined && (
                  <DescriptionItem label={t("Mailbox")}>
                    {message.mailbox.address}
                  </DescriptionItem>
                )}
              </DescriptionList>
            </div>
          </SectionPanel>

          {/* The reason is shown beside the match, never instead of it. A match
              nobody can check is a match nobody will trust. */}
          <SectionPanel title={t("What it is about")}>
            <div className="p-3">
              {message.matchReason === "" ? (
                <p className="text-muted-foreground text-sm">
                  {t("Nothing was matched to this message.")}
                </p>
              ) : (
                <DescriptionList layout="inline">
                  <DescriptionItem label={t("Why")}>{message.matchReason}</DescriptionItem>
                  {message.matchedShipmentId !== null &&
                    message.matchedShipmentId !== undefined && (
                      <DescriptionItem label={t("Shipment")}>
                        <Link
                          className="text-brand ui-focus-ring underline-offset-4 hover:underline"
                          to={`/shipment-management/shipments?entityId=${message.matchedShipmentId}&modType=edit`}
                        >
                          {t("Open the shipment")}
                        </Link>
                      </DescriptionItem>
                    )}
                  {message.matchedCustomerId !== null &&
                    message.matchedCustomerId !== undefined && (
                      <DescriptionItem label={t("Customer")}>
                        <Link
                          className="text-brand ui-focus-ring underline-offset-4 hover:underline"
                          to={`/billing/configurations/customers?entityId=${message.matchedCustomerId}&modType=edit`}
                        >
                          {t("Open the customer")}
                        </Link>
                      </DescriptionItem>
                    )}
                </DescriptionList>
              )}
            </div>
          </SectionPanel>

          {message.attachments.length > 0 && (
            <SectionPanel title={t("Attachments")} count={message.attachments.length}>
              <ul className="divide-border-subtle divide-y">
                {message.attachments.map((attachment) => (
                  <li key={attachment.id} className="flex items-start gap-3 px-3 py-2">
                    <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                      <span className="truncate text-sm">{attachment.fileName}</span>
                      {attachment.failureText !== "" && (
                        <span className="text-danger text-xs">{attachment.failureText}</span>
                      )}
                    </div>
                    <Badge variant="neutral" appearance="outline">
                      {attachment.kind}
                    </Badge>
                  </li>
                ))}
              </ul>
            </SectionPanel>
          )}

          <SectionPanel title={t("The message")}>
            <div className="p-3">
              {/* The body is whatever a sender chose to write, so it is shown
                  as text and never as markup. */}
              <p className="text-muted-foreground text-sm whitespace-pre-wrap">
                {message.textBody === "" ? t("This message had no text.") : message.textBody}
              </p>
            </div>
          </SectionPanel>
        </div>
      </ScrollArea>

      {message.needsReview && (
        <footer className="border-border flex flex-col gap-2 border-t p-3">
          <Textarea
            value={note}
            onChange={(event) => setNote(event.target.value)}
            placeholder={t("Say what you did, so the next person does not repeat it")}
            rows={2}
          />
          <div className="flex justify-end gap-2">
            <Button
              size="sm"
              variant="outline"
              disabled={busy}
              onClick={() => reviewMutation.mutate("Ignored")}
            >
              {t("Ignore")}
            </Button>
            <Button size="sm" disabled={busy} onClick={() => reviewMutation.mutate("Actioned")}>
              {t("Mark handled")}
            </Button>
          </div>
        </footer>
      )}
    </div>
  );
}
