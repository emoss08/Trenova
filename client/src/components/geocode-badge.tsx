import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { queries } from "@/lib/queries";
import { truncateText } from "@/lib/utils";
import { decimalStringSchema, optionalStringSchema } from "@/types/helpers";
import { useQuery } from "@tanstack/react-query";
import { CheckIcon, CopyIcon } from "lucide-react";
import { lazy, Suspense } from "react";
import { z } from "zod";

const LazyMap = lazy(() => import("@/components/lazy-map"));

export const GeocodeBadgeSchema = z.object({
  longitude: decimalStringSchema.nullish(),
  latitude: decimalStringSchema.nullish(),
  placeId: optionalStringSchema,
  isGeocoded: z.boolean().default(false).nullish(),
});

export type GeocodeBadgeSchema = z.infer<typeof GeocodeBadgeSchema>;

export function GeocodedBadge({ longitude, latitude, placeId }: GeocodeBadgeSchema) {
  const position = {
    lat: latitude ?? 0,
    lng: longitude ?? 0,
  };

  const googleMapsData = useQuery({
    ...queries.integration.runtimeConfig("GoogleMaps"),
  });

  if (!googleMapsData.data?.apiKey) {
    return null;
  }

  return (
    <HoverCard>
      <HoverCardTrigger
        render={
          <div className="flex items-center justify-center">
            <div className="size-2 rounded-full bg-success" />
          </div>
        }
      />
      <HoverCardContent className="flex w-auto flex-col gap-2 p-2">
        <div className="flex flex-col gap-0.5">
          <Row label="Longitude" value={longitude} />
          <Row label="Latitude" value={latitude} />
          <Row label="Place ID" value={placeId} />
        </div>
        {placeId && (
          <div className="h-32 w-full overflow-hidden rounded-md border border-border">
            <Suspense
              fallback={
                <div className="flex h-full w-full animate-pulse items-center justify-center bg-muted text-xs text-muted-foreground">
                  Loading map...
                </div>
              }
            >
              <LazyMap apiKey={googleMapsData.data?.apiKey ?? ""} position={position} />
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
      className="group flex cursor-pointer items-center justify-between gap-4 text-sm"
      onClick={(e) => {
        e.stopPropagation();
        void copy(value);
      }}
    >
      <span className="text-muted-foreground">{label}</span>
      <span className="flex items-center gap-1 truncate font-mono">
        <span className="invisible group-hover:visible">
          {!isCopied ? <CopyIcon className="size-3" /> : <CheckIcon className="size-3" />}
        </span>
        {truncateText(value, 40)}
      </span>
    </div>
  );
}
