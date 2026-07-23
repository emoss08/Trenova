import type { Shipment, ShipmentStatus } from "@trenova/shared/types/shipment";
import { useCommandCenterStore } from "../command-center/store";
import { useCommandCenterUrl } from "../command-center/url-state";
import { ShipmentCurrentPin, ShipmentEndpointPin, type PinTone } from "./shipment-pin";
import {
  getShipmentStopsWithCoords,
  pickCurrentStopIndex,
  type StopWithCoord,
} from "./shipment-route-coordinates";
import { ShipmentRouteLine } from "./shipment-route-line";
import { useMapShipments } from "./use-map-shipments";

const STATUS_TONE: Record<ShipmentStatus, PinTone | null> = {
  New: "muted",
  PartiallyAssigned: "muted",
  Assigned: "muted",
  InTransit: "brand",
  PartiallyCompleted: "brand",
  Delayed: "destructive",
  Completed: "success",
  ReadyToInvoice: "success",
  Invoiced: "success",
  Canceled: null,
};

export function ShipmentRouteOverlay({ enabled = true }: { enabled?: boolean }) {
  const { data } = useMapShipments(enabled);
  const highlightId = useCommandCenterStore.use.highlightId();
  const setHighlightId = useCommandCenterStore.use.setHighlightId();
  const [, setUrl] = useCommandCenterUrl();

  const shipments = data?.results ?? [];
  if (shipments.length === 0) return null;

  const hasHighlight = !!highlightId;

  return (
    <>
      {shipments.map((s) => {
        const tone = STATUS_TONE[s.status];
        if (!tone) return null;

        const stops = getShipmentStopsWithCoords(s as Shipment);
        if (stops.length < 2) return null;

        const isHighlighted = !!s.id && highlightId === s.id;
        const isDimmed = hasHighlight && !isHighlighted;
        const onMouseEnter = () => s.id && setHighlightId(s.id);
        const onMouseLeave = () => setHighlightId(null);
        const onClick = () => {
          if (!s.id) return;
          void setUrl({ expanded: s.id });
        };
        const title = `${(s as Shipment).proNumber ?? s.id ?? "Shipment"} · ${s.status}`;
        const currentIdx = pickCurrentStopIndex(stops);

        return (
          <ShipmentRouteGroup
            key={s.id}
            stops={stops}
            currentIdx={currentIdx}
            tone={tone}
            highlighted={isHighlighted}
            dimmed={isDimmed}
            title={title}
            onMouseEnter={onMouseEnter}
            onMouseLeave={onMouseLeave}
            onClick={onClick}
          />
        );
      })}
    </>
  );
}

function ShipmentRouteGroup({
  stops,
  currentIdx,
  tone,
  highlighted,
  dimmed,
  title,
  onMouseEnter,
  onMouseLeave,
  onClick,
}: {
  stops: StopWithCoord[];
  currentIdx: number;
  tone: PinTone;
  highlighted: boolean;
  dimmed: boolean;
  title: string;
  onMouseEnter: () => void;
  onMouseLeave: () => void;
  onClick: () => void;
}) {
  const segments = [];
  for (let i = 0; i < stops.length - 1; i++) {
    const dashed = stops[i].stop.status !== "Completed";
    segments.push(
      <ShipmentRouteLine
        key={`seg-${i}`}
        origin={stops[i].latlng}
        destination={stops[i + 1].latlng}
        tone={tone}
        highlighted={highlighted}
        dimmed={dimmed}
        dashed={dashed}
      />,
    );
  }

  return (
    <>
      {segments}
      {stops.map(({ latlng }, idx) => (
        <ShipmentEndpointPin key={`pin-${idx}`} position={latlng} dimmed={dimmed} title={title} />
      ))}
      <ShipmentCurrentPin
        position={stops[currentIdx].latlng}
        tone={tone}
        highlighted={highlighted}
        dimmed={dimmed}
        title={title}
        onMouseEnter={onMouseEnter}
        onMouseLeave={onMouseLeave}
        onClick={onClick}
      />
    </>
  );
}
