import { Metadata } from "@/components/metadata";
import { publicLinkErrorKind } from "@/components/public-page/error-kind";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { ShipmentTrackingService } from "@/services/shipment-tracking";
import {
  AlertTriangleIcon,
  ClockIcon,
  SlashCircle01Icon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQuery } from "@tanstack/react-query";
import { useParams } from "react-router";
import { TrackingTopBar, TrackingView } from "./_components/tracking-view";
import { TrackingViewSkeleton } from "./_components/tracking-view-skeleton";

const shipmentTrackingService = new ShipmentTrackingService();
const REFRESH_MS = 2 * 60 * 1000;

function Message({
  icon,
  title,
  body,
  action,
}: {
  icon: React.ReactNode;
  title: string;
  body: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="st-page relative flex min-h-full flex-col" data-tone="queued">
      <TrackingTopBar now={0} />
      <main className="st-in mx-auto flex w-full max-w-xl flex-1 flex-col items-start justify-center gap-4 px-5 pb-32">
        <span className="flex size-11 items-center justify-center rounded-full bg-background text-muted-foreground ring-1 ring-foreground/10">
          {icon}
        </span>
        <h1 className="font-newsreader text-4xl leading-tight tracking-[-0.02em]">{title}</h1>
        <p className="text-sm text-muted-foreground">{body}</p>
        {action}
      </main>
    </div>
  );
}

/**
 * The page a customer opens from a tracking link. It is read-only and shows
 * only what the customer's settings allow; the server decides that, so the
 * page draws whatever parts it was given.
 */
export function ShipmentTrackingPublicPage() {
  const t = useT();
  const { token = "" } = useParams();
  const now = useNowSeconds(30_000);

  const viewQuery = useQuery({
    queryKey: ["public-shipment-tracking", token] as const,
    queryFn: () => shipmentTrackingService.getPublicView(token),
    retry: false,
    refetchInterval: REFRESH_MS,
  });

  let content: React.ReactNode;
  if (viewQuery.isLoading) {
    content = <TrackingViewSkeleton />;
  } else if (viewQuery.isError) {
    const kind = publicLinkErrorKind(viewQuery.error);
    content =
      kind === "throttled" ? (
        <Message
          icon={<ClockIcon className="size-5" aria-hidden />}
          title={t("Too many attempts")}
          body={t("Please wait a minute and open the link again.")}
        />
      ) : kind === "unavailable" ? (
        <Message
          icon={<AlertTriangleIcon className="size-5" aria-hidden />}
          title={t("Temporarily unavailable")}
          body={t(
            "The shipment could not be loaded because of a temporary problem. Please try again in a moment.",
          )}
          action={
            <Button
              type="button"
              variant="outline"
              size="sm"
              isLoading={viewQuery.isRefetching}
              loadingText={t("Retrying...")}
              onClick={() => void viewQuery.refetch()}
            >
              {t("Try again")}
            </Button>
          }
        />
      ) : (
        <InvalidLink />
      );
  } else if (viewQuery.data) {
    content = (
      <TrackingView
        view={viewQuery.data}
        now={now}
        mode={{
          kind: "link",
          token,
          updatedAt: Math.floor(viewQuery.dataUpdatedAt / 1000),
          refreshing: viewQuery.isFetching,
          onRefresh: () => void viewQuery.refetch(),
        }}
      />
    );
  } else {
    content = <InvalidLink />;
  }

  return (
    <>
      <Metadata title={t("Shipment tracking")} description={t("Follow a shipment")} />
      <div className="fixed inset-0 h-svh w-full overflow-y-auto bg-canvas text-foreground">
        {content}
      </div>
    </>
  );
}

function InvalidLink() {
  const t = useT();
  return (
    <Message
      icon={<SlashCircle01Icon className="size-5" aria-hidden />}
      title={t("This tracking link is no longer valid")}
      body={t(
        "The link may have expired or been turned off. Ask the company that sent it for a new one.",
      )}
    />
  );
}
