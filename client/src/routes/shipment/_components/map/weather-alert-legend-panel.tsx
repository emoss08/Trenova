import { cn } from "@/lib/utils";
import {
  ALERT_CATEGORY_CONFIG,
  type WeatherAlertCategory,
  type WeatherAlertFeature,
} from "@/types/weather-alert";
import { ControlPosition, MapControl } from "@vis.gl/react-google-maps";
import { ChevronDownIcon, TriangleAlertIcon } from "lucide-react";
import { useMemo, useState } from "react";

export function WeatherAlertLegendPanel({ features }: { features: WeatherAlertFeature[] }) {
  const [collapsed, setCollapsed] = useState(false);

  const categoryCounts = useMemo(() => {
    const counts = new Map<WeatherAlertCategory, number>();
    for (const feature of features) {
      const cat = feature.properties.alertCategory;
      counts.set(cat, (counts.get(cat) ?? 0) + 1);
    }
    return counts;
  }, [features]);

  const totalCount = features.length;

  return (
    <MapControl position={ControlPosition.LEFT_BOTTOM}>
      <div className="m-2.5 rounded-lg border bg-background shadow-sm">
        <button
          type="button"
          onClick={() => setCollapsed((p) => !p)}
          className="flex w-full items-center justify-between gap-2 px-3 py-2"
        >
          <div className="flex items-center gap-1.5">
            <TriangleAlertIcon className="size-3.5 text-muted-foreground" />
            <span className="text-xs font-semibold text-foreground">
              Public Alerts
            </span>
            <span className="text-2xs tabular-nums text-muted-foreground">({totalCount})</span>
          </div>
          <ChevronDownIcon
            className={cn(
              "size-3.5 text-muted-foreground transition-transform",
              collapsed && "-rotate-90",
            )}
          />
        </button>

        {!collapsed && categoryCounts.size > 0 && (
          <div className="flex flex-col gap-1 border-t px-3 pt-2 pb-2.5">
            {(Object.keys(ALERT_CATEGORY_CONFIG) as WeatherAlertCategory[]).map((cat) => {
              const count = categoryCounts.get(cat);
              if (!count) return null;
              const config = ALERT_CATEGORY_CONFIG[cat];
              return (
                <div key={cat} className="flex items-center justify-between gap-2">
                  <div className="flex items-center gap-2">
                    <span
                      className="inline-block size-3 shrink-0 rounded-sm"
                      style={{ backgroundColor: config.stroke }}
                    />
                    <span className="text-xs text-foreground">{config.label}</span>
                  </div>
                  <span className="text-2xs tabular-nums text-muted-foreground">{count}</span>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </MapControl>
  );
}
