import { queries } from "@/lib/queries";
import type { StopSchema } from "@/lib/schemas/stop-schema";
import { useQuery } from "@tanstack/react-query";
import { getStopTypeLabel } from "../stop-utils";

export function LocationDisplay({
  location,
  type,
}: {
  location?: StopSchema["location"] | null;
  type: StopSchema["type"];
}) {
  const { data } = useQuery({
    ...queries.location.getById(location?.id || ""),
  });

  const displayLocation = data || location;

  if (!displayLocation) {
    return (
      <div className="text-sm text-primary">
        <span>{getStopTypeLabel(type)}</span>
      </div>
    );
  }

  return (
    <LocationDisplayOuter>
      <LocationDisplayInner>
        <span className="text-xs">{displayLocation.addressLine1}</span>
        <span className="text-2xs">({getStopTypeLabel(type)})</span>
      </LocationDisplayInner>
      <LocationDisplayAddress>
        {displayLocation.city}, {displayLocation.state?.abbreviation}{" "}
        {displayLocation.postalCode}
      </LocationDisplayAddress>
    </LocationDisplayOuter>
  );
}

function LocationDisplayInner({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex items-center gap-1 text-sm text-primary">
      {children}
    </div>
  );
}

function LocationDisplayAddress({ children }: { children: React.ReactNode }) {
  return <div className="text-2xs text-muted-foreground">{children}</div>;
}

function LocationDisplayOuter({ children }: { children: React.ReactNode }) {
  return <div className="flex-1">{children}</div>;
}
