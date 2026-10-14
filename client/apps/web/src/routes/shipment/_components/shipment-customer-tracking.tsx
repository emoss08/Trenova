import { recordPath } from "@/config/record-links";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { usePermission } from "@/hooks/use-permission";
import {
  linkState,
  liveLinkCount,
  skipReasonLabel,
  statusEmailSteps,
  statusEventLabel,
  TRACKING_FIELDS,
  trackingFieldLabel,
  type StatusEmailStep,
} from "@/lib/customer-tracking";
import {
  issueCustomerTrackingLink,
  revokeCustomerTrackingLink,
  type CustomerTrackingLinkRow,
  type CustomerTrackingProfile,
  type IssuedCustomerTrackingLink,
} from "@/lib/graphql/customer-tracking";
import { customer as customerQueries } from "@/lib/queries/customer";
import { shipment as shipmentQueries } from "@/lib/queries/shipment";
import { issueTrackingLinkFormSchema, type IssueTrackingLinkFormValues } from "@/types/check-call";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  AlertTriangleIcon,
  ArrowUpRightIcon,
  CheckIcon,
  ChevronDownIcon,
  ClockIcon,
  Copy01Icon,
  EyeIcon,
  Link01Icon,
  Send01Icon,
  SlashCircle01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { Dialog, DialogContent, DialogTitle } from "@trenova/shared/components/ui/dialog";
import { Input } from "@trenova/shared/components/ui/input";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo, formatUnixDateMedium, formatUnixDateTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { buttonVariants } from "@trenova/shared/lib/variants/button";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { lazy, useCallback, useState } from "react";
import { useForm } from "react-hook-form";
import { Link } from "react-router";
import { TrackingViewSkeleton } from "@/routes/shipment-tracking-public/_components/tracking-view-skeleton";

const CustomerTrackingPreview = lazy(() => import("./customer-tracking-preview"));

function Section({
  title,
  aside,
  children,
  className,
}: {
  title: React.ReactNode;
  aside?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <section className={cn("flex flex-col gap-3 border-b border-border-subtle px-4 py-5 last:border-b-0", className)}>
      <div className="flex min-h-7 items-center justify-between gap-2">
        <h3 className="flex items-center gap-2 text-sm font-semibold">{title}</h3>
        {aside}
      </div>
      {children}
    </section>
  );
}

/**
 * Sharing a shipment with its customer: what the customer sees, the links out
 * to its tracking page, and the status emails its rules send. What the page
 * shows and which emails go out are set on the customer.
 */
export default function ShipmentCustomerTracking({ shipment }: { shipment?: Shipment | null }) {
  const shipmentId = shipment?.id ?? "";
  const customerId = shipment?.customerId ?? "";
  const profileQuery = useQuery({
    ...customerQueries.trackingProfile(customerId),
    enabled: customerId !== "",
  });
  if (!shipmentId) {
    return null;
  }
  const profile = profileQuery.data ?? null;

  return (
    <div className="flex flex-col">
      <Audience
        shipmentId={shipmentId}
        customerId={customerId}
        customerName={shipment?.customer?.name ?? ""}
        profile={profile}
        loading={profileQuery.isLoading}
      />
      <Share
        shipmentId={shipmentId}
        canceled={shipment?.status === "Canceled"}
        linksOff={profile != null && !profile.linkEnabled}
        customerId={customerId}
      />
      <StatusEmails shipmentId={shipmentId} profile={profile} />
    </div>
  );
}

function Audience({
  shipmentId,
  customerId,
  customerName,
  profile,
  loading,
}: {
  shipmentId: string;
  customerId: string;
  customerName: string;
  profile: CustomerTrackingProfile | null;
  loading: boolean;
}) {
  const t = useT();
  const shown = new Set(profile?.visibleFields ?? []);
  const hidden = TRACKING_FIELDS.filter((field) => !shown.has(field));
  const [previewing, setPreviewing] = useState(false);
  const audience = customerName || t("The customer");

  return (
    <section className="flex flex-col gap-3 border-b border-border-subtle bg-muted/40 px-4 py-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className="text-xs text-muted-foreground">{t("Shared with")}</span>
          <span className="truncate text-base font-semibold">{audience}</span>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          <Button type="button" variant="outline" size="sm" onClick={() => setPreviewing(true)}>
            <EyeIcon className="size-3.5" aria-hidden />
            {t("Preview")}
          </Button>
          {customerId && (
            <Link
              to={recordPath("customer", customerId, { tab: "tracking" })}
              className={cn(buttonVariants({ variant: "ghost", size: "sm" }))}
            >
              {t("Settings")}
              <ArrowUpRightIcon className="size-3.5" aria-hidden />
            </Link>
          )}
        </div>
      </div>
      {loading ? (
        <Skeleton className="h-5 w-64" />
      ) : profile ? (
        <div className="flex flex-wrap items-center gap-1.5">
          <Badge variant={profile.linkEnabled ? "success" : "neutral"}>
            {profile.linkEnabled
              ? t("Links on · {0, plural, one {# day} other {# days}}", profile.linkTtlDays)
              : t("Links off")}
          </Badge>
          <Tooltip>
            <TooltipTrigger render={<span />}>
              <Badge variant="neutral" appearance="outline" className="cursor-help">
                <EyeIcon className="size-3" aria-hidden />
                {t("Sees {0} of {1} details", shown.size, TRACKING_FIELDS.length)}
              </Badge>
            </TooltipTrigger>
            <TooltipContent className="max-w-64">
              <div className="flex flex-col gap-1.5 py-0.5">
                <span className="font-medium">{t("The page shows")}</span>
                <span>
                  {TRACKING_FIELDS.filter((field) => shown.has(field))
                    .map((field) => trackingFieldLabel(t, field))
                    .join(", ")}
                </span>
                {hidden.length > 0 && (
                  <>
                    <span className="mt-1 font-medium">{t("Hidden")}</span>
                    <span>{hidden.map((field) => trackingFieldLabel(t, field)).join(", ")}</span>
                  </>
                )}
              </div>
            </TooltipContent>
          </Tooltip>
        </div>
      ) : null}
      <Dialog open={previewing} onOpenChange={setPreviewing}>
        <DialogContent size="fullscreen" className="bg-canvas">
          <DialogTitle className="sr-only">
            {t("Preview of the tracking page {0} sees", audience)}
          </DialogTitle>
          <div className="h-dvh overflow-y-auto">
            <LazyComponent fallback={<TrackingViewSkeleton />}>
              {previewing && <CustomerTrackingPreview shipmentId={shipmentId} audience={audience} />}
            </LazyComponent>
          </div>
        </DialogContent>
      </Dialog>
    </section>
  );
}

function Share({
  shipmentId,
  canceled,
  linksOff,
  customerId,
}: {
  shipmentId: string;
  canceled: boolean;
  linksOff: boolean;
  customerId: string;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const now = useNowSeconds(30_000);
  const { allowed: canShare } = usePermission(Resource.Shipment, Operation.Update);
  const [issued, setIssued] = useState<IssuedCustomerTrackingLink | null>(null);
  const linksQuery = useQuery(shipmentQueries.customerTrackingLinks(shipmentId));
  const links = linksQuery.data ?? [];
  const live = links.filter((link) => linkState(link, now) === "live");
  const past = links.filter((link) => linkState(link, now) !== "live");

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

  const blocked = canceled
    ? t("The shipment is canceled, so it cannot be shared.")
    : linksOff
      ? t("Tracking links are turned off for this customer.")
      : null;

  return (
    <Section
      title={
        <>
          {t("Tracking links")}
          {liveLinkCount(links, now) > 0 && (
            <Badge variant="neutral" appearance="outline">
              {t("{0} live", liveLinkCount(links, now))}
            </Badge>
          )}
        </>
      }
      aside={
        canShare && live.length > 1 ? (
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
      {blocked ? (
        <Alert size="sm" variant={linksOff ? "info" : "default"}>
          <SlashCircle01Icon aria-hidden />
          <AlertDescription>
            {blocked}{" "}
            {linksOff && customerId && (
              <Link
                to={recordPath("customer", customerId, { tab: "tracking" })}
                className="font-medium underline underline-offset-2"
              >
                {t("Turn them on")}
              </Link>
            )}
          </AlertDescription>
        </Alert>
      ) : canShare && !issued ? (
        <ShareForm
          shipmentId={shipmentId}
          onIssued={(result) => {
            setIssued(result);
            void refresh();
          }}
        />
      ) : null}

      {issued && <IssuedLink issued={issued} onDone={() => setIssued(null)} />}

      {linksQuery.isLoading ? (
        <div className="flex flex-col gap-2">
          <Skeleton className="h-10" />
          <Skeleton className="h-10" />
        </div>
      ) : links.length === 0 ? (
        <p className="text-xs text-muted-foreground">
          {t("Nothing shared yet. Status emails add their own link when they go out.")}
        </p>
      ) : (
        <>
          {live.length > 0 && (
            <ul className="-mx-2 flex flex-col">
              {live.map((link) => (
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
          {past.length > 0 && (
            <Collapsible>
              <CollapsibleTrigger className="group flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground">
                <ChevronDownIcon
                  className="size-3.5 transition-transform group-data-[panel-open]:rotate-180"
                  aria-hidden
                />
                {t(
                  "{0, plural, one {# expired or revoked link} other {# expired or revoked links}}",
                  past.length,
                )}
              </CollapsibleTrigger>
              <CollapsibleContent>
                <ul className="-mx-2 mt-1 flex flex-col opacity-70">
                  {past.map((link) => (
                    <LinkRow key={link.id} link={link} now={now} canRevoke={false} busy={false} />
                  ))}
                </ul>
              </CollapsibleContent>
            </Collapsible>
          )}
        </>
      )}
    </Section>
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
  onRevoke?: () => void;
}) {
  const t = useT();
  const state = linkState(link, now);
  const who =
    link.recipientEmail ?? (link.purpose === "Notification" ? t("Status email") : t("Copied link"));
  const meta = [
    link.purpose === "Notification"
      ? t("In a status email {0}", formatSecondsAgo(now - link.createdAt))
      : link.recipientEmail
        ? t("Emailed {0}", formatSecondsAgo(now - link.createdAt))
        : t("Copied {0}", formatSecondsAgo(now - link.createdAt)),
    link.viewCount > 0 && link.lastViewedAt != null
      ? t(
          "{0, plural, one {Opened once} other {Opened # times}}, last {1}",
          link.viewCount,
          formatSecondsAgo(now - link.lastViewedAt),
        )
      : t("Not opened yet"),
    state === "live"
      ? t("Works until {0}", formatUnixDateMedium(link.expiresAt))
      : state === "revoked"
        ? t("Revoked")
        : t("Expired {0}", formatUnixDateMedium(link.expiresAt)),
  ];

  return (
    <li className="group flex items-center gap-3 rounded-md px-2 py-2 hover:bg-muted/60">
      <span
        className={cn(
          "size-2 shrink-0 rounded-full",
          state === "live"
            ? link.viewCount > 0
              ? "bg-success"
              : "bg-success/40"
            : "bg-border",
        )}
        aria-hidden
      />
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="truncate text-sm">{who}</span>
        <span className="truncate text-xs text-muted-foreground tabular-nums">
          {meta.join(" · ")}
        </span>
      </div>
      {canRevoke && state === "live" && onRevoke && (
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                type="button"
                size="icon-sm"
                variant="ghost"
                aria-label={t("Revoke")}
                disabled={busy}
                onClick={onRevoke}
                className="opacity-60 group-hover:opacity-100"
              />
            }
          >
            <XCloseIcon className="size-3.5" aria-hidden />
          </TooltipTrigger>
          <TooltipContent>{t("Revoke this link")}</TooltipContent>
        </Tooltip>
      )}
    </li>
  );
}

function ShareForm({
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
    register,
    handleSubmit,
    reset,
    formState: { isSubmitting, errors },
  } = form;
  const [copying, setCopying] = useState(false);

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
    <form
      className="flex flex-col gap-1.5"
      onSubmit={(event) => {
        event.stopPropagation();
        void handleSubmit((values) => mutateAsync(values))(event);
      }}
    >
      <div className="flex items-start gap-2">
        <div className="flex flex-1 flex-col gap-1">
          <Input
            type="email"
            placeholder={t("Email the link to…")}
            aria-label={t("Email the link to")}
            aria-invalid={errors.recipientEmail ? true : undefined}
            {...register("recipientEmail")}
          />
          {errors.recipientEmail?.message && (
            <span className="text-xs text-danger-foreground">{errors.recipientEmail.message}</span>
          )}
        </div>
        <Button type="submit" isLoading={isSubmitting && !copying} loadingText={t("Sending...")}>
          <Send01Icon className="size-3.5" aria-hidden />
          {t("Send")}
        </Button>
      </div>
      <Button
        type="button"
        variant="link"
        size="sm"
        className="self-start px-0 text-muted-foreground"
        disabled={isSubmitting}
        onClick={async () => {
          setCopying(true);
          try {
            await mutateAsync({ recipientEmail: "" });
          } finally {
            setCopying(false);
          }
        }}
      >
        <Link01Icon className="size-3.5" aria-hidden />
        {t("Copy a link to share yourself")}
      </Button>
    </form>
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

  const tone = issued.emailError
    ? "bg-warning-subtle text-warning-subtle-foreground ring-warning-border"
    : "bg-success-subtle text-success-subtle-foreground ring-success-border";

  return (
    <div className={cn("flex flex-col gap-2 rounded-lg p-3 text-xs ring-1", tone)}>
      <div className="flex items-center gap-2 font-medium">
        {issued.emailError ? (
          <AlertTriangleIcon className="size-3.5 shrink-0" aria-hidden />
        ) : (
          <CheckIcon className="size-3.5 shrink-0" aria-hidden />
        )}
        {issued.emailed
          ? t("Emailed to {0}", issued.link.recipientEmail ?? "")
          : issued.emailError
            ? t("The link works, but the email did not go out")
            : t("Link ready to share")}
      </div>
      {issued.emailError && <span className="opacity-80">{issued.emailError}</span>}
      <div className="flex items-center gap-2">
        <code
          className="min-w-0 flex-1 truncate rounded-md bg-background px-2 py-1.5 font-mono text-xs text-foreground ring-1 ring-foreground/10 select-all"
          title={issued.url}
        >
          {issued.url}
        </code>
        <Button
          type="button"
          size="sm"
          variant="outline"
          className="shrink-0"
          onClick={() => void copy(issued.url, { withToast: true })}
        >
          <Copy01Icon className="size-3.5" aria-hidden />
          {isCopied ? t("Copied") : t("Copy")}
        </Button>
      </div>
      <div className="flex items-center justify-between gap-2">
        <span className="opacity-80">{t("Shown once. Share a new link if this one is lost.")}</span>
        <Button type="button" size="sm" variant="ghost" onClick={onDone}>
          {t("Done")}
        </Button>
      </div>
    </div>
  );
}

function StepMark({ step }: { step: StatusEmailStep }) {
  const status = step.latest?.status;
  const base = "relative z-10 flex size-7 items-center justify-center rounded-full";
  if (status === "Sent") {
    return (
      <span className={cn(base, "bg-success text-foreground-on-solid")}>
        <CheckIcon className="size-3.5" aria-hidden />
      </span>
    );
  }
  if (status === "Pending") {
    return (
      <span className={cn(base, "bg-warning-subtle text-warning-subtle-foreground ring-1 ring-warning-border")}>
        <ClockIcon className="size-3.5" aria-hidden />
      </span>
    );
  }
  if (status === "Failed") {
    return (
      <span className={cn(base, "bg-danger-subtle text-danger-subtle-foreground ring-1 ring-danger-border")}>
        <XCloseIcon className="size-3.5" aria-hidden />
      </span>
    );
  }
  if (status === "Skipped") {
    return (
      <span className={cn(base, "bg-muted text-muted-foreground")}>
        <SlashCircle01Icon className="size-3.5" aria-hidden />
      </span>
    );
  }
  return (
    <span
      className={cn(
        base,
        "bg-background",
        step.enabled ? "ring-1 ring-border" : "border border-dashed border-border",
      )}
    />
  );
}

function stepCaption(t: ReturnType<typeof useT>, step: StatusEmailStep, now: number): string {
  const row = step.latest;
  if (!row) {
    return step.enabled ? t("Not yet") : t("Off");
  }
  switch (row.status) {
    case "Sent":
      return t("Sent {0}", formatSecondsAgo(now - (row.sentAt ?? row.occurredAt)));
    case "Pending":
      return row.attempts > 0 ? t("Retrying") : t("Sending");
    case "Failed":
      return t("Failed");
    default:
      return skipReasonLabel(t, row.skipReason) ?? t("Skipped");
  }
}

function StatusEmails({
  shipmentId,
  profile,
}: {
  shipmentId: string;
  profile: CustomerTrackingProfile | null;
}) {
  const t = useT();
  const now = useNowSeconds(30_000);
  const notificationsQuery = useQuery(shipmentQueries.customerStatusNotifications(shipmentId));
  const rows = notificationsQuery.data ?? [];
  const steps = statusEmailSteps(
    {
      PickedUp: profile?.notifyPickedUp ?? false,
      Delayed: profile?.notifyDelayed ?? false,
      Delivered: profile?.notifyDelivered ?? false,
      PodReady: profile?.notifyPodReady ?? false,
    },
    rows,
  );
  const troubled = steps
    .map((step) => step.latest)
    .filter(
      (row): row is NonNullable<typeof row> =>
        row != null && row.status !== "Sent" && row.lastError != null,
    );
  const recipients = (profile?.recipients ?? "")
    .split(/[\s,;]+/)
    .filter(Boolean);
  const anyOn = steps.some((step) => step.enabled);

  return (
    <Section
      title={t("Status emails")}
      aside={
        recipients.length > 0 ? (
          <Tooltip>
            <TooltipTrigger className="max-w-52 truncate text-xs text-muted-foreground">
              {recipients.length === 1
                ? t("To {0}", recipients[0])
                : t("To {0} +{1}", recipients[0], recipients.length - 1)}
            </TooltipTrigger>
            <TooltipContent>{recipients.join(", ")}</TooltipContent>
          </Tooltip>
        ) : undefined
      }
    >
      {notificationsQuery.isLoading ? (
        <Skeleton className="h-16" />
      ) : (
        <>
          <ol className="relative grid grid-cols-4">
            <span className="absolute top-3.5 right-[12.5%] left-[12.5%] h-px bg-border" aria-hidden />
            {steps.map((step) => (
              <li key={step.event} className="flex flex-col items-center gap-1.5 text-center">
                <StepMark step={step} />
                <span
                  className={cn(
                    "text-xs font-medium",
                    !step.enabled && !step.latest && "text-muted-foreground",
                  )}
                >
                  {step.event === "PodReady" ? t("POD ready") : statusEventLabel(t, step.event)}
                  {step.count > 1 && <span className="text-muted-foreground"> ×{step.count}</span>}
                </span>
                <span
                  className="text-xs text-muted-foreground tabular-nums"
                  title={step.latest ? formatUnixDateTime(step.latest.occurredAt) : undefined}
                >
                  {stepCaption(t, step, now)}
                </span>
              </li>
            ))}
          </ol>
          {troubled.map((row) => (
            <Alert key={row.id} size="sm" variant={row.status === "Failed" ? "destructive" : "warning"}>
              <AlertTriangleIcon aria-hidden />
              <AlertTitle>
                {row.status === "Failed"
                  ? t("{0}: the email gave up", statusEventLabel(t, row.event))
                  : t(
                      "{0}: retrying after {1, plural, one {# failure} other {# failures}}",
                      statusEventLabel(t, row.event),
                      row.attempts,
                    )}
              </AlertTitle>
              <AlertDescription className="font-mono">{row.lastError}</AlertDescription>
            </Alert>
          ))}
          {!anyOn && rows.length === 0 && (
            <p className="text-xs text-muted-foreground">
              {t("No status emails are turned on for this customer.")}
            </p>
          )}
        </>
      )}
    </Section>
  );
}
