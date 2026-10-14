import { InputField } from "@/components/fields/input-field";
import { SectionPanel } from "@/components/section-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { usePermission } from "@/hooks/use-permission";
import {
  linkState,
  linkStateAttrs,
  liveLinkCount,
  notificationStatusAttrs,
  skipReasonLabel,
  statusEventLabel,
} from "@/lib/customer-tracking";
import {
  issueCustomerTrackingLink,
  revokeCustomerTrackingLink,
  type CustomerTrackingLinkRow,
  type IssuedCustomerTrackingLink,
} from "@/lib/graphql/customer-tracking";
import { shipment as shipmentQueries } from "@/lib/queries/shipment";
import { issueTrackingLinkFormSchema, type IssueTrackingLinkFormValues } from "@/types/check-call";
import { zodResolver } from "@hookform/resolvers/zod";
import { Copy01Icon, Link01Icon, Mail01Icon, XCloseIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Input } from "@trenova/shared/components/ui/input";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";

const nowInSeconds = () => Math.floor(Date.now() / 1000);

/**
 * Sharing a shipment with its customer: links to its tracking page, and the
 * status emails the customer's rules have sent. What the page shows and which
 * emails go out are set on the customer.
 */
export default function ShipmentCustomerTracking({ shipment }: { shipment?: Shipment | null }) {
  const shipmentId = shipment?.id ?? "";
  if (!shipmentId) {
    return null;
  }
  return (
    <div className="flex flex-col gap-4 p-4">
      <TrackingLinks shipmentId={shipmentId} canceled={shipment?.status === "Canceled"} />
      <StatusEmails shipmentId={shipmentId} />
    </div>
  );
}

function TrackingLinks({ shipmentId, canceled }: { shipmentId: string; canceled: boolean }) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canShare } = usePermission(Resource.Shipment, Operation.Update);
  const [now] = useState(nowInSeconds);
  const [issued, setIssued] = useState<IssuedCustomerTrackingLink | null>(null);
  const linksQuery = useQuery(shipmentQueries.customerTrackingLinks(shipmentId));
  const links = linksQuery.data ?? [];
  const live = liveLinkCount(links, now);

  const refresh = useCallback(
    () =>
      queryClient.invalidateQueries({
        queryKey: shipmentQueries.customerTrackingLinks(shipmentId).queryKey,
      }),
    [queryClient, shipmentId],
  );

  const revoke = useApiMutation({
    mutationFn: (linkId?: string) => revokeCustomerTrackingLink(shipmentId, linkId),
    resourceName: "Tracking link",
    onSuccess: refresh,
  });

  return (
    <SectionPanel
      title={t("Customer tracking links")}
      icon={<Link01Icon />}
      count={live}
      action={
        canShare && live > 1 ? (
          <Button
            type="button"
            size="sm"
            variant="ghost"
            disabled={revoke.isPending}
            onClick={() => revoke.mutate(undefined)}
          >
            {t("Revoke all")}
          </Button>
        ) : undefined
      }
    >
      <div className="flex flex-col gap-3 p-3">
        {canShare && !canceled && !issued && (
          <ShareLinkForm
            shipmentId={shipmentId}
            onIssued={(result) => {
              setIssued(result);
              void refresh();
            }}
          />
        )}
        {issued && <IssuedLink issued={issued} onDone={() => setIssued(null)} />}
        {linksQuery.isLoading ? (
          <Skeleton className="h-12" />
        ) : links.length === 0 ? (
          <p className="text-muted-foreground text-xs">
            {t("No link has been shared for this shipment.")}
          </p>
        ) : (
          <ul className="divide-border-subtle flex flex-col divide-y">
            {links.map((link) => (
              <LinkRow
                key={link.id}
                link={link}
                now={now}
                canRevoke={canShare}
                busy={revoke.isPending}
                onRevoke={() => revoke.mutate(link.id)}
              />
            ))}
          </ul>
        )}
      </div>
    </SectionPanel>
  );
}

function LinkRow({
  link,
  now,
  canRevoke,
  busy,
  onRevoke,
}: {
  link: CustomerTrackingLinkRow;
  now: number;
  canRevoke: boolean;
  busy: boolean;
  onRevoke: () => void;
}) {
  const t = useT();
  const state = linkState(link, now);
  const attrs = linkStateAttrs(t)[state];

  return (
    <li className="flex flex-wrap items-center gap-x-3 gap-y-1 py-2 text-xs">
      <Badge variant={phaseTone(attrs.phase)}>{attrs.text}</Badge>
      <span>
        {link.purpose === "Notification"
          ? t("In a status email to {0}", link.recipientEmail ?? "")
          : link.recipientEmail
            ? t("Emailed to {0}", link.recipientEmail)
            : t("Shared by hand")}
      </span>
      <span className="text-muted-foreground tabular-nums">
        {t("Created {0}", formatUnixDateTime(link.createdAt))}
      </span>
      <span className="text-muted-foreground tabular-nums">
        {link.viewCount > 0 && link.lastViewedAt != null
          ? t(
              "{0, plural, one {Opened # time} other {Opened # times}}, last {1}",
              link.viewCount,
              formatUnixDateTime(link.lastViewedAt),
            )
          : t("Not opened yet")}
      </span>
      {state === "live" && (
        <span className="text-muted-foreground tabular-nums">
          {t("Works until {0}", formatUnixDateTime(link.expiresAt))}
        </span>
      )}
      {canRevoke && state === "live" && (
        <Button
          type="button"
          size="sm"
          variant="ghost"
          className="ml-auto"
          disabled={busy}
          onClick={onRevoke}
        >
          <XCloseIcon className="size-3.5" aria-hidden />
          {t("Revoke")}
        </Button>
      )}
    </li>
  );
}

function ShareLinkForm({
  shipmentId,
  onIssued,
}: {
  shipmentId: string;
  onIssued: (issued: IssuedCustomerTrackingLink) => void;
}) {
  const t = useT();
  const form = useForm<IssueTrackingLinkFormValues>({
    resolver: zodResolver(issueTrackingLinkFormSchema),
    defaultValues: { recipientEmail: "" },
  });
  const {
    control,
    handleSubmit,
    reset,
    formState: { isSubmitting },
  } = form;

  const { mutateAsync } = useApiMutation({
    mutationFn: (values: IssueTrackingLinkFormValues) =>
      issueCustomerTrackingLink({
        shipmentId,
        recipientEmail: values.recipientEmail.trim() || undefined,
      }),
    resourceName: "Tracking link",
    form,
    onSuccess: (result) => {
      reset({ recipientEmail: "" });
      onIssued(result);
    },
  });

  return (
    <FormProvider {...form}>
      <Form
        className="flex flex-col gap-2"
        onSubmit={(event) => {
          event.stopPropagation();
          void handleSubmit((values) => mutateAsync(values))(event);
        }}
      >
        <FormGroup cols={1}>
          <FormControl>
            <InputField
              control={control}
              name="recipientEmail"
              type="email"
              label={t("Email to")}
              placeholder={t("shipping@customer.com")}
              description={t(
                "Leave empty to copy the link and share it yourself. The page shows what the customer's tracking settings allow.",
              )}
            />
          </FormControl>
        </FormGroup>
        <div className="flex justify-end">
          <Button type="submit" size="sm" isLoading={isSubmitting} loadingText={t("Creating...")}>
            <Mail01Icon className="size-3.5" aria-hidden />
            {t("Share link")}
          </Button>
        </div>
      </Form>
    </FormProvider>
  );
}

function IssuedLink({
  issued,
  onDone,
}: {
  issued: IssuedCustomerTrackingLink;
  onDone: () => void;
}) {
  const t = useT();
  const { copy, isCopied } = useCopyToClipboard();

  return (
    <div className="flex flex-col gap-2">
      {issued.emailed ? (
        <Alert size="sm" variant="success">
          <AlertDescription>
            {t("The link was emailed to {0}.", issued.link.recipientEmail ?? "")}
          </AlertDescription>
        </Alert>
      ) : issued.emailError ? (
        <Alert size="sm" variant="warning">
          <AlertDescription>
            {t("The link works, but the email could not be sent: {0}", issued.emailError)}
          </AlertDescription>
        </Alert>
      ) : null}
      <div className="flex items-center gap-2">
        <Input
          readOnly
          value={issued.url}
          aria-label={t("Tracking link")}
          className="font-mono text-xs"
        />
        <Button
          type="button"
          variant="outline"
          onClick={() => void copy(issued.url, { withToast: true })}
        >
          <Copy01Icon className="size-3.5" aria-hidden />
          {isCopied ? t("Copied") : t("Copy")}
        </Button>
      </div>
      <div className="flex items-center justify-between gap-2">
        <p className="text-muted-foreground text-xs">
          {t("This link is shown once. Share a new one if it is lost.")}
        </p>
        <Button type="button" size="sm" variant="ghost" onClick={onDone}>
          {t("Done")}
        </Button>
      </div>
    </div>
  );
}

function StatusEmails({ shipmentId }: { shipmentId: string }) {
  const t = useT();
  const notificationsQuery = useQuery(shipmentQueries.customerStatusNotifications(shipmentId));
  const rows = notificationsQuery.data ?? [];
  const statusAttrs = notificationStatusAttrs(t);

  return (
    <SectionPanel title={t("Customer status emails")} icon={<Mail01Icon />} count={rows.length}>
      <div className="p-3">
        {notificationsQuery.isLoading ? (
          <Skeleton className="h-12" />
        ) : rows.length === 0 ? (
          <p className="text-muted-foreground text-xs">
            {t(
              "No status email has been owed yet. The customer's tracking settings choose which events send one.",
            )}
          </p>
        ) : (
          <ul className="divide-border-subtle flex flex-col divide-y">
            {rows.map((row) => {
              const attrs = statusAttrs[row.status];
              const skipped = skipReasonLabel(t, row.skipReason);
              return (
                <li key={row.id} className="flex flex-col gap-0.5 py-2 text-xs">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-medium">{statusEventLabel(t, row.event)}</span>
                    <Badge variant={phaseTone(attrs.phase)} title={attrs.description}>
                      {attrs.text}
                    </Badge>
                    <span className="text-muted-foreground ml-auto tabular-nums">
                      {formatUnixDateTime(row.occurredAt)}
                    </span>
                  </div>
                  {row.recipients.length > 0 && (
                    <span className="text-muted-foreground">
                      {t("To {0}", row.recipients.join(", "))}
                    </span>
                  )}
                  {row.sentAt != null && (
                    <span className="text-muted-foreground tabular-nums">
                      {t("Sent {0}", formatUnixDateTime(row.sentAt))}
                    </span>
                  )}
                  {skipped && <span className="text-muted-foreground">{skipped}</span>}
                  {row.lastError && row.status !== "Sent" && (
                    <span className="text-danger-foreground">
                      {t(
                        "{0, plural, one {# attempt} other {# attempts}}: {1}",
                        row.attempts,
                        row.lastError,
                      )}
                    </span>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </SectionPanel>
  );
}
