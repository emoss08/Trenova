import { Badge } from "@/components/ui/badge";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/ui/hover-card";
import { Icon } from "@/components/ui/icons";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { queries } from "@/lib/queries";
import { GeocodeBadeSchema } from "@/lib/schemas/geocode-schema";
import { truncateText } from "@/lib/utils";
import { faCheck, faCopy } from "@fortawesome/pro-solid-svg-icons";
import { useQuery } from "@tanstack/react-query";
import { lazy, Suspense } from "react";

const LazyMap = lazy(() => import("@/components/lazy-map"));

export function GeocodedBadge({
  longitude,
  latitude,
  placeId,
}: GeocodeBadeSchema) {
  const position = {
    lat: latitude ?? 0,
    lng: longitude ?? 0,
  };

  const googleMapsData = useQuery({
    ...queries.googleMaps.getAPIKey(),
  });

  return (
    <HoverCard>
      <HoverCardTrigger asChild>
        <div className="flex items-center justify-center">
          <Badge variant="active" className="text-xs">
            Geocoded
          </Badge>
        </div>
      </HoverCardTrigger>
      <HoverCardContent className="flex flex-col gap-2 p-2 w-auto">
        <div className="flex flex-col gap-0.5">
          <Row label="Longitude" value={longitude} />
          <Row label="Latitude" value={latitude} />
          <Row label="Place ID" value={placeId} />
        </div>
        {placeId && (
          <div className="h-32 w-full rounded-md overflow-hidden border border-border">
            <Suspense
              fallback={
                <div className="h-full w-full bg-muted animate-pulse flex items-center justify-center text-xs text-muted-foreground">
                  Loading map...
                </div>
              }
            >
              <LazyMap
                apiKey={googleMapsData.data?.apiKey ?? ""}
                position={position}
              />
            </Suspense>
          </div>
        )}
      </HoverCardContent>
    </HoverCard>
  );
}
function Row({ label, value }: { label: string; value: any }) {
  const { copy, isCopied } = useCopyToClipboard();

  return (
    <div
      className="group flex gap-4 text-sm justify-between items-center cursor-pointer"
      onClick={(e) => {
        e.stopPropagation();
        copy(value);
      }}
    >
      <span className="text-muted-foreground">{label}</span>
      <span className="font-mono truncate flex items-center gap-1">
        <span className="invisible group-hover:visible">
          {!isCopied ? (
            <Icon icon={faCopy} className="size-3" />
          ) : (
            <Icon icon={faCheck} className="size-3" />
          )}
        </span>
        {truncateText(value, 40)}
      </span>
    </div>
  );
}
