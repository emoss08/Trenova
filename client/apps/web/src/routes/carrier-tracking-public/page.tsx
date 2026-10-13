import { Metadata } from "@/components/metadata";
import { publicLinkErrorKind } from "@/components/public-page/error-kind";
import { PublicPageShell } from "@/components/public-page/public-page-shell";
import { StatusCard } from "@/components/public-page/status-card";
import {
  checkCallPostFailure,
  newClientEventId,
  nextStopAction,
  type CheckCallPostFailure,
} from "@/lib/carrier-tracking";
import { CarrierTrackingService } from "@/services/carrier-tracking";
import {
  AlertTriangleIcon,
  CheckCircleIcon,
  ClockIcon,
  MarkerPin01Icon,
  SlashCircle01Icon,
} from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import { Separator } from "@trenova/shared/components/ui/separator";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { stopTypeLabel } from "@trenova/shared/lib/stop-type";
import type {
  PublicCheckCallPayload,
  PublicTrackingStop,
  PublicTrackingView,
} from "@trenova/shared/types/carrier-tracking";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useId, useRef, useState } from "react";
import { useParams } from "react-router";

const carrierTrackingService = new CarrierTrackingService();

const LIMITS = { city: 100, stateCode: 10, delayReason: 255, note: 1000, contactName: 100 };

type UpdateDraft = {
  city: string;
  stateCode: string;
  delayed: boolean;
  delayReason: string;
  estimatedArrival: string;
  note: string;
  contactName: string;
  shareLocation: boolean;
};

const emptyDraft: UpdateDraft = {
  city: "",
  stateCode: "",
  delayed: false,
  delayReason: "",
  estimatedArrival: "",
  note: "",
  contactName: "",
  shareLocation: false,
};

function optional(value: string): string | undefined {
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

function localInputToUnix(value: string): number | undefined {
  if (!value) {
    return undefined;
  }
  const parsed = new Date(value).getTime();
  return Number.isNaN(parsed) ? undefined : Math.floor(parsed / 1000);
}

function currentPosition(): Promise<{ latitude: number; longitude: number } | null> {
  if (typeof navigator === "undefined" || !navigator.geolocation) {
    return Promise.resolve(null);
  }
  return new Promise((resolve) => {
    navigator.geolocation.getCurrentPosition(
      (position) =>
        resolve({ latitude: position.coords.latitude, longitude: position.coords.longitude }),
      () => resolve(null),
      { enableHighAccuracy: false, timeout: 10_000, maximumAge: 300_000 },
    );
  });
}

function stopPlace(stop: PublicTrackingStop): string {
  const town = [stop.city, stop.stateCode].filter(Boolean).join(", ");
  return [stop.locationName, town].filter(Boolean).join(" · ");
}

function StopLine({ stop }: { stop: PublicTrackingStop }) {
  const t = useT();
  let status = t("Due {0}", formatUnixDateTime(stop.windowStart));
  if (stop.departedAt != null) {
    status = t("Left {0}", formatUnixDateTime(stop.departedAt));
  } else if (stop.arrivedAt != null) {
    status = t("Arrived {0}", formatUnixDateTime(stop.arrivedAt));
  }

  return (
    <li className="flex flex-col gap-0.5 py-2">
      <span className="text-xs font-medium">
        {t("Stop {0} · {1}", stop.sequence + 1, stopTypeLabel(stop.type))}
      </span>
      <span className="text-xs text-muted-foreground">{stopPlace(stop)}</span>
      <span className="text-xs tabular-nums">{status}</span>
    </li>
  );
}

function UpdateForm({
  isSubmitting,
  onSubmit,
}: {
  isSubmitting: boolean;
  onSubmit: (draft: UpdateDraft) => void;
}) {
  const t = useT();
  const id = useId();
  const [draft, setDraft] = useState<UpdateDraft>(emptyDraft);
  const set = <K extends keyof UpdateDraft>(key: K, value: UpdateDraft[K]) =>
    setDraft((current) => ({ ...current, [key]: value }));
  const missingReason = draft.delayed && draft.delayReason.trim() === "";

  return (
    <form
      className="flex flex-col gap-3"
      noValidate
      onSubmit={(event) => {
        event.preventDefault();
        if (missingReason) {
          return;
        }
        onSubmit(draft);
      }}
    >
      <div className="grid grid-cols-[1fr_6rem] gap-2">
        <div className="flex flex-col gap-1">
          <Label htmlFor={`${id}-city`}>{t("City")}</Label>
          <Input
            id={`${id}-city`}
            value={draft.city}
            maxLength={LIMITS.city}
            autoComplete="address-level2"
            onChange={(event) => set("city", event.target.value)}
          />
        </div>
        <div className="flex flex-col gap-1">
          <Label htmlFor={`${id}-state`}>{t("State")}</Label>
          <Input
            id={`${id}-state`}
            value={draft.stateCode}
            maxLength={LIMITS.stateCode}
            autoComplete="address-level1"
            onChange={(event) => set("stateCode", event.target.value.toUpperCase())}
          />
        </div>
      </div>
      <label className="flex items-center gap-2 text-xs">
        <Checkbox
          checked={draft.shareLocation}
          onCheckedChange={(checked) => set("shareLocation", checked === true)}
        />
        {t("Share this device's location")}
      </label>
      <label className="flex items-center gap-2 text-xs">
        <Checkbox
          checked={draft.delayed}
          onCheckedChange={(checked) => set("delayed", checked === true)}
        />
        {t("Running late")}
      </label>
      {draft.delayed ? (
        <div className="flex flex-col gap-1">
          <Label htmlFor={`${id}-reason`}>{t("Why")}</Label>
          <Input
            id={`${id}-reason`}
            value={draft.delayReason}
            maxLength={LIMITS.delayReason}
            aria-invalid={missingReason || undefined}
            onChange={(event) => set("delayReason", event.target.value)}
          />
          {missingReason ? (
            <p className="text-xs text-danger-foreground">{t("Say why the load is delayed")}</p>
          ) : null}
        </div>
      ) : null}
      <div className="flex flex-col gap-1">
        <Label htmlFor={`${id}-eta`}>{t("Expected at the next stop")}</Label>
        <Input
          id={`${id}-eta`}
          type="datetime-local"
          value={draft.estimatedArrival}
          onChange={(event) => set("estimatedArrival", event.target.value)}
        />
      </div>
      <div className="flex flex-col gap-1">
        <Label htmlFor={`${id}-contact`}>{t("Your name")}</Label>
        <Input
          id={`${id}-contact`}
          value={draft.contactName}
          maxLength={LIMITS.contactName}
          autoComplete="name"
          onChange={(event) => set("contactName", event.target.value)}
        />
      </div>
      <div className="flex flex-col gap-1">
        <Label htmlFor={`${id}-note`}>{t("Note")}</Label>
        <Textarea
          id={`${id}-note`}
          value={draft.note}
          maxLength={LIMITS.note}
          onChange={(event) => set("note", event.target.value)}
        />
      </div>
      <Button type="submit" isLoading={isSubmitting} loadingText={t("Sending...")}>
        {t("Send update")}
      </Button>
    </form>
  );
}

function TrackingCard({
  view,
  isSubmitting,
  failure,
  sent,
  onPost,
}: {
  view: PublicTrackingView;
  isSubmitting: boolean;
  failure: CheckCallPostFailure | null;
  sent: string | null;
  onPost: (payload: Omit<PublicCheckCallPayload, "clientEventId">, shareLocation: boolean) => void;
}) {
  const t = useT();
  const next = nextStopAction(view.stops);

  return (
    <Card className="w-full max-w-md">
      <CardHeader>
        <CardTitle className="text-base">
          {view.shipmentProNumber
            ? t("Load {0} for {1}", view.shipmentProNumber, view.companyName)
            : t("Load for {0}", view.companyName)}
        </CardTitle>
        <p className="text-xs text-muted-foreground">
          {[
            view.carrierName,
            view.lastCheckCallAt
              ? t("Last update {0}", formatUnixDateTime(view.lastCheckCallAt))
              : t("No updates yet"),
          ]
            .filter(Boolean)
            .join(" · ")}
        </p>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <ul className="divide-y divide-border">
          {view.stops.map((stop) => (
            <StopLine key={stop.id} stop={stop} />
          ))}
        </ul>

        {sent ? (
          <Alert size="sm" variant="success">
            <AlertDescription>{sent}</AlertDescription>
          </Alert>
        ) : null}
        {failure?.kind === "rejected" ? (
          <Alert size="sm" variant="destructive">
            <AlertDescription>{failure.messages.join(" ")}</AlertDescription>
          </Alert>
        ) : null}
        {failure?.kind === "unavailable" ? (
          <Alert size="sm" variant="warning">
            <AlertDescription>
              {t("Your update could not be sent because of a temporary problem. Please try again.")}
            </AlertDescription>
          </Alert>
        ) : null}
        {failure?.kind === "throttled" ? (
          <Alert size="sm" variant="warning">
            <AlertDescription>
              {t("Too many updates in a short time. Please wait a minute and try again.")}
            </AlertDescription>
          </Alert>
        ) : null}

        {next ? (
          <Button
            type="button"
            size="lg"
            disabled={isSubmitting}
            onClick={() => onPost({ kind: next.kind, stopId: next.stop.id }, true)}
          >
            <MarkerPin01Icon className="size-4" aria-hidden />
            {next.kind === "Arrival"
              ? t("I've arrived at stop {0}", next.stop.sequence + 1)
              : t("I've left stop {0}", next.stop.sequence + 1)}
          </Button>
        ) : null}

        <Separator />
        <p className="text-sm font-medium">{t("Send a position update")}</p>
        <UpdateForm
          isSubmitting={isSubmitting}
          onSubmit={(draft) =>
            onPost(
              {
                kind: "Update",
                city: optional(draft.city),
                stateCode: optional(draft.stateCode),
                delayed: draft.delayed,
                delayReason: draft.delayed ? optional(draft.delayReason) : undefined,
                estimatedArrival: localInputToUnix(draft.estimatedArrival),
                note: optional(draft.note),
                contactName: optional(draft.contactName),
              },
              draft.shareLocation,
            )
          }
        />

        <div className="flex items-center gap-1.5 rounded-md border bg-muted px-2 py-1.5">
          <ClockIcon className="size-3.5 shrink-0 text-muted-foreground" aria-hidden />
          <span className="text-xs text-muted-foreground">
            {t(
              "This link works until {0} or until the load is done.",
              formatUnixDateTime(view.expiresAt),
            )}
          </span>
        </div>
      </CardContent>
    </Card>
  );
}

function InvalidLinkCard() {
  const t = useT();

  return (
    <StatusCard
      icon={<SlashCircle01Icon className="size-8 text-muted-foreground" aria-hidden />}
      title={t("This tracking link is no longer valid")}
      body={t(
        "The link may have expired or been replaced by a newer one. Contact the broker if you believe this is an error.",
      )}
    />
  );
}

function ThrottledCard() {
  const t = useT();

  return (
    <StatusCard
      icon={<ClockIcon className="size-8 text-muted-foreground" aria-hidden />}
      title={t("Too many attempts")}
      body={t("Please wait a minute and open the link again.")}
    />
  );
}

/**
 * The page a carrier opens from a tracking link to report where the load is.
 * Every post carries an event id kept until it succeeds, so a retry after a
 * dropped connection records the update once.
 */
export function CarrierTrackingPublicPage() {
  const t = useT();
  const { token = "" } = useParams();
  const [failure, setFailure] = useState<CheckCallPostFailure | null>(null);
  const [sent, setSent] = useState<string | null>(null);
  const pendingEventId = useRef<string | null>(null);

  const previewQuery = useQuery({
    queryKey: ["public-carrier-tracking", token] as const,
    queryFn: () => carrierTrackingService.getPublicTracking(token),
    retry: false,
  });

  const postMutation = useMutation({
    mutationFn: async ({
      payload,
      shareLocation,
    }: {
      payload: Omit<PublicCheckCallPayload, "clientEventId">;
      shareLocation: boolean;
    }) => {
      pendingEventId.current ??= newClientEventId();
      const position = shareLocation ? await currentPosition() : null;
      return carrierTrackingService.postCheckCall(token, {
        ...payload,
        ...position,
        clientEventId: pendingEventId.current,
      });
    },
    onSuccess: (result, { payload }) => {
      pendingEventId.current = null;
      setFailure(null);
      if (result.outcome === "Refused" && result.reason) {
        setSent(t("Your update was saved, but the stop time was not changed: {0}", result.reason));
      } else if (payload.kind === "Arrival") {
        setSent(t("Arrival sent. Thank you."));
      } else if (payload.kind === "Departure") {
        setSent(t("Departure sent. Thank you."));
      } else {
        setSent(t("Update sent. Thank you."));
      }
      void previewQuery.refetch();
    },
    onError: (error: unknown) => {
      const next = checkCallPostFailure(error);
      if (next.kind === "rejected" || next.kind === "closed" || next.kind === "invalid") {
        pendingEventId.current = null;
      }
      setSent(null);
      setFailure(next);
    },
  });

  let content: React.ReactNode;
  if (failure?.kind === "invalid") {
    content = <InvalidLinkCard />;
  } else if (failure?.kind === "closed" || previewQuery.data?.closed) {
    content = (
      <StatusCard
        icon={<CheckCircleIcon className="size-8 text-muted-foreground" aria-hidden />}
        title={t("This load is no longer being tracked")}
        body={t("The load is finished or was canceled. Contact the broker if anything changed.")}
      />
    );
  } else if (previewQuery.isLoading) {
    content = (
      <Card className="w-full max-w-md">
        <CardContent className="flex flex-col gap-3 py-6">
          <Skeleton className="h-5 w-48 rounded-md" />
          <Skeleton className="h-24 rounded-md" />
          <Skeleton className="h-9 rounded-md" />
        </CardContent>
      </Card>
    );
  } else if (previewQuery.isError) {
    const kind = publicLinkErrorKind(previewQuery.error);
    content =
      kind === "throttled" ? (
        <ThrottledCard />
      ) : kind === "unavailable" ? (
        <StatusCard
          icon={<AlertTriangleIcon className="size-8 text-muted-foreground" aria-hidden />}
          title={t("Temporarily unavailable")}
          body={t(
            "The load could not be loaded because of a temporary problem. Please try again in a moment.",
          )}
          action={
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="mt-2"
              isLoading={previewQuery.isRefetching}
              loadingText={t("Retrying...")}
              onClick={() => void previewQuery.refetch()}
            >
              {t("Try again")}
            </Button>
          }
        />
      ) : (
        <InvalidLinkCard />
      );
  } else if (previewQuery.data) {
    content = (
      <TrackingCard
        view={previewQuery.data}
        isSubmitting={postMutation.isPending}
        failure={failure}
        sent={sent}
        onPost={(payload, shareLocation) => postMutation.mutate({ payload, shareLocation })}
      />
    );
  } else {
    content = <InvalidLinkCard />;
  }

  return (
    <>
      <Metadata title={t("Load tracking")} description={t("Report where a load is")} />
      <PublicPageShell
        footer={t(
          "Powered by Trenova. Questions about this load? Reply to the email the link arrived in.",
        )}
      >
        {content}
      </PublicPageShell>
    </>
  );
}
