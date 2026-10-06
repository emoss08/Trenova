import { useT } from "@trenova/shared/i18n/use-t";
import { PlugIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useShipmentCapabilities } from "@/lib/shipment-board/capabilities";
import { useNavigate } from "react-router";
import { lazy, Suspense } from "react";
import { ShipmentMapPanelBoundary } from "../../map/map-boundary";

const ShipmentMapPanel = lazy(() => import("../../map/shipment-map-panel"));

const INTEGRATIONS_PATH = "/admin/integrations";

/** The live map when Google Maps is configured, otherwise a way to connect it. */
export default function ShipmentMapView() {
  const t = useT();
  const { maps } = useShipmentCapabilities();
  const navigate = useNavigate();

  if (!maps) {
    return (
      <div className="flex h-full min-h-80 flex-col items-center justify-center gap-2 p-8 text-center">
        <span className="text-muted-foreground font-mono text-xs">{t("Live map")}</span>
        <p className="text-sm font-semibold">{t("Google Maps isn't connected for this workspace")}</p>
        <p className="text-muted-foreground max-w-sm text-sm">
          {t("Connect it to see trucks, lanes and geofences here. Table and Timeline work without it.")}
        </p>
        <Button variant="outline" size="sm" onClick={() => void navigate(INTEGRATIONS_PATH)}>
          <PlugIcon className="size-3.5" />
          {t("Connect Google Maps")}
        </Button>
      </div>
    );
  }

  return (
    <ShipmentMapPanelBoundary>
      <Suspense fallback={null}>
        <ShipmentMapPanel backgroundEnabled />
      </Suspense>
    </ShipmentMapPanelBoundary>
  );
}
