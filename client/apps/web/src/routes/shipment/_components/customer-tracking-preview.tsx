import { useNowSeconds } from "@/hooks/use-now-seconds";
import { shipment as shipmentQueries } from "@/lib/queries/shipment";
import { TrackingView } from "@/routes/shipment-tracking-public/_components/tracking-view";
import { TrackingViewSkeleton } from "@/routes/shipment-tracking-public/_components/tracking-view-skeleton";
import { AlertTriangleIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQuery } from "@tanstack/react-query";

/**
 * The customer's tracking page as it would read through a link sent now,
 * drawn from the shipment's current state and the customer's settings.
 */
export default function CustomerTrackingPreview({
  shipmentId,
  audience,
}: {
  shipmentId: string;
  audience: string;
}) {
  const t = useT();
  const now = useNowSeconds(30_000);
  const previewQuery = useQuery(shipmentQueries.customerTrackingPreview(shipmentId));

  if (previewQuery.isLoading) {
    return <TrackingViewSkeleton />;
  }
  if (previewQuery.isError || !previewQuery.data) {
    return (
      <div className="mx-auto w-full max-w-xl px-5 pt-24">
        <Alert variant="destructive">
          <AlertTriangleIcon aria-hidden />
          <AlertTitle>{t("The preview could not be drawn")}</AlertTitle>
          <AlertDescription>
            {previewQuery.error instanceof Error
              ? previewQuery.error.message
              : t("Try again in a moment.")}
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  return (
    <TrackingView view={previewQuery.data} now={now} mode={{ kind: "preview", audience }} />
  );
}
