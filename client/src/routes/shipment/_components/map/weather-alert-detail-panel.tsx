import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { queries } from "@/lib/queries";
import {
  ACTIVITY_TYPE_COLORS,
  SEVERITY_BADGE_MAP,
  type WeatherAlertActivity,
  type WeatherAlertActivityType,
  type WeatherAlertFeature,
} from "@/types/weather-alert";
import { useQuery } from "@tanstack/react-query";
import { ControlPosition, MapControl } from "@vis.gl/react-google-maps";
import { ClockIcon, MapPinIcon, XIcon } from "lucide-react";

function formatUnixTimestamp(unix: number | null | undefined): string {
  if (!unix) return "—";
  return new Date(unix * 1000).toLocaleString([], {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    hour12: true,
    timeZoneName: "short",
  });
}

function formatActivityTime(unix: number): string {
  return new Date(unix * 1000).toLocaleString([], {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    hour12: true,
  });
}

function capitalizeFirst(str: string): string {
  return str.charAt(0).toUpperCase() + str.slice(1);
}

function ActivityTimeline({ activities }: { activities: WeatherAlertActivity[] }) {
  if (activities.length === 0) {
    return <p className="text-xs text-muted-foreground">No activity recorded</p>;
  }

  return (
    <div className="flex flex-col gap-0">
      {activities.map((activity, index) => {
        const color = ACTIVITY_TYPE_COLORS[activity.activityType as WeatherAlertActivityType];
        const isLast = index === activities.length - 1;

        return (
          <div key={activity.id} className="flex gap-2.5">
            <div className="flex flex-col items-center">
              <span
                className="mt-1 inline-block size-2.5 shrink-0 rounded-full"
                style={{ backgroundColor: color }}
              />
              {!isLast && <div className="w-px grow bg-border" />}
            </div>
            <div className="pb-3">
              <span className="text-xs font-medium text-foreground">
                {capitalizeFirst(activity.activityType)}
              </span>
              <p className="text-2xs text-muted-foreground">
                {formatActivityTime(activity.timestamp)}
              </p>
            </div>
          </div>
        );
      })}
    </div>
  );
}

export function WeatherAlertDetailPanel({
  alertId,
  feature,
  onClose,
}: {
  alertId: string;
  feature: WeatherAlertFeature;
  onClose: () => void;
}) {
  const { data, isLoading } = useQuery({
    ...queries.weatherAlert.detail(alertId),
    enabled: !!alertId,
  });

  const props = feature.properties;
  const severityVariant = SEVERITY_BADGE_MAP[props.severity ?? ""] ?? "outline";

  return (
    <MapControl position={ControlPosition.LEFT_TOP}>
      <div className="m-2.5 w-72 overflow-hidden rounded-lg border bg-background shadow-sm">
        <div className="flex items-start justify-between gap-2 p-3 pb-2">
          <div className="flex flex-col gap-1">
            <div className="flex items-center gap-1.5">
              <span className="text-sm font-semibold text-foreground">{props.event}</span>
              {props.severity && <Badge variant={severityVariant}>{props.severity}</Badge>}
            </div>
            {props.headline && (
              <p className="text-xs leading-snug text-muted-foreground">{props.headline}</p>
            )}
          </div>
          <button
            type="button"
            onClick={onClose}
            className="shrink-0 rounded-sm p-0.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            <XIcon className="size-3.5" />
          </button>
        </div>

        <ScrollArea className="h-[200px]">
          <div className="flex flex-col">
            {props.areaDesc && (
              <div className="flex items-start gap-1.5 px-3 pb-2">
                <MapPinIcon className="mt-0.5 size-3 shrink-0 text-muted-foreground" />
                <span className="text-xs text-muted-foreground">{props.areaDesc}</span>
              </div>
            )}

            <Separator className="mx-3" />

            {props.description && (
              <>
                <div className="px-3 py-2">
                  <p className="text-xs leading-relaxed text-foreground">{props.description}</p>
                </div>
                <Separator className="mx-3" />
              </>
            )}

            {props.instruction && (
              <>
                <div className="px-3 py-2">
                  <span className="text-2xs font-medium uppercase tracking-wider text-muted-foreground">
                    Recommended Action
                  </span>
                  <p className="mt-1 text-xs leading-relaxed text-foreground">
                    {props.instruction}
                  </p>
                </div>
                <Separator className="mx-3" />
              </>
            )}

            <div className="flex items-center gap-3 px-3 pt-2">
              <div className="flex items-center gap-1 text-muted-foreground">
                <ClockIcon className="size-3" />
                <span className="text-2xs">Effective</span>
              </div>
              <span className="text-2xs tabular-nums text-foreground">
                {formatUnixTimestamp(props.effective)}
              </span>
            </div>
            <div className="flex items-center gap-3 px-3 pt-1 pb-2">
              <div className="flex items-center gap-1 text-muted-foreground">
                <ClockIcon className="size-3" />
                <span className="text-2xs">Expires</span>
              </div>
              <span className="text-2xs tabular-nums text-foreground">
                {formatUnixTimestamp(props.expires)}
              </span>
            </div>

            <Separator className="mx-3" />

            <div className="p-3">
              <span className="text-2xs font-medium uppercase tracking-wider text-muted-foreground">
                Activity
              </span>
              <div className="mt-2">
                {isLoading ? (
                  <p className="text-xs text-muted-foreground">Loading activity...</p>
                ) : (
                  <ActivityTimeline activities={data?.activities ?? []} />
                )}
              </div>
            </div>
          </div>
        </ScrollArea>
      </div>
    </MapControl>
  );
}
