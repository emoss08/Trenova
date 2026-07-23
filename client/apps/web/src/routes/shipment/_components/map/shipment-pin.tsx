import { cn } from "@trenova/shared/lib/utils";
import { AdvancedMarker } from "@vis.gl/react-google-maps";

export type PinTone = "brand" | "destructive" | "success" | "warning" | "muted";

const TONE_VAR: Record<PinTone, string> = {
  brand: "var(--brand)",
  destructive: "var(--destructive)",
  success: "var(--success)",
  warning: "var(--warning)",
  muted: "var(--muted-foreground)",
};

type EndpointProps = {
  position: google.maps.LatLngLiteral;
  dimmed?: boolean;
  title?: string;
};

export function ShipmentEndpointPin({ position, dimmed = false, title }: EndpointProps) {
  return (
    <AdvancedMarker position={position} zIndex={20} title={title}>
      <span
        aria-hidden
        className="block rounded-full transition-opacity"
        style={{
          width: 9,
          height: 9,
          background: "var(--card)",
          border: "1.5px solid var(--muted-foreground)",
          boxShadow: "0 1px 2px rgba(0,0,0,0.2)",
          opacity: dimmed ? 0.25 : 1,
        }}
      />
    </AdvancedMarker>
  );
}

type CurrentProps = {
  position: google.maps.LatLngLiteral;
  tone: PinTone;
  highlighted?: boolean;
  dimmed?: boolean;
  title?: string;
  onMouseEnter?: () => void;
  onMouseLeave?: () => void;
  onClick?: () => void;
};

export function ShipmentCurrentPin({
  position,
  tone,
  highlighted = false,
  dimmed = false,
  title,
  onMouseEnter,
  onMouseLeave,
  onClick,
}: CurrentProps) {
  const color = TONE_VAR[tone];
  const size = highlighted ? 20 : 12;

  return (
    <AdvancedMarker
      position={position}
      zIndex={highlighted ? 100 : 30}
      title={title}
      onClick={onClick}
    >
      <div
        className={cn(
          "cc-pulse-pin relative flex items-center justify-center transition-opacity",
          highlighted && "is-highlighted",
        )}
        onMouseEnter={onMouseEnter}
        onMouseLeave={onMouseLeave}
        style={{
          width: size,
          height: size,
          opacity: dimmed ? 0.25 : 1,
          color,
        }}
      >
        <span
          aria-hidden
          className="block size-full rounded-full"
          style={{
            background: color,
            border: highlighted ? "2px solid var(--card)" : "1.5px solid var(--card)",
            boxShadow: "0 2px 4px rgba(0,0,0,0.3)",
          }}
        />
      </div>
    </AdvancedMarker>
  );
}
