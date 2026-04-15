import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { useShipmentMapStore } from "@/stores/shipment-map-store";
import type { RainViewerFrame, WeatherLayerId, WeatherOption } from "@/types/shipment-map";
import { ControlPosition, MapControl } from "@vis.gl/react-google-maps";
import {
  ChevronDownIcon,
  CloudIcon,
  CloudRainIcon,
  GaugeIcon,
  PauseIcon,
  PlayIcon,
  SkipBackIcon,
  SkipForwardIcon,
  StepBackIcon,
  StepForwardIcon,
  ThermometerIcon,
  WindIcon,
} from "lucide-react";
import { useCallback, useMemo, useRef } from "react";

const WEATHER_OPTIONS: WeatherOption[] = [
  {
    id: "precipitation",
    label: "Radar",
    description: "Track rain, snow and sleet",
    icon: CloudRainIcon,
  },
  {
    id: "wind",
    label: "Wind Speed",
    description: "See sustained wind speed (wind gusts not indicated)",
    icon: WindIcon,
  },
  {
    id: "temperature",
    label: "Temperature",
    description: "Hourly temperature forecast",
    icon: ThermometerIcon,
  },
  {
    id: "clouds",
    label: "Cloud Cover",
    description: "Estimated cloud coverage worldwide",
    icon: CloudIcon,
  },
  {
    id: "pressure",
    label: "Pressure",
    description: "Atmospheric sea level pressure",
    icon: GaugeIcon,
  },
];

function formatFullDateTime(unixSeconds: number): string {
  const date = new Date(unixSeconds * 1000);
  const weekday = date.toLocaleDateString([], { weekday: "long" });
  const time = date.toLocaleTimeString([], {
    hour: "numeric",
    minute: "2-digit",
    hour12: true,
    timeZoneName: "short",
  });
  return `${weekday}, ${time}`;
}

function formatTickTime(unixSeconds: number): string {
  const date = new Date(unixSeconds * 1000);
  const minutes = date.getMinutes();
  return date.toLocaleTimeString([], {
    hour: "numeric",
    ...(minutes !== 0 && { minute: "2-digit" }),
    hour12: true,
  });
}

function getTimelineRange(
  frames: RainViewerFrame[],
): { firstTs: number; lastTs: number; fullRange: number } | null {
  if (frames.length < 2) return null;
  const startHour = new Date(frames[0].time * 1000);
  startHour.setMinutes(0, 0, 0);
  const endHour = new Date(frames[frames.length - 1].time * 1000);
  endHour.setMinutes(0, 0, 0);
  endHour.setHours(endHour.getHours() + 1);
  const firstTs = Math.floor(startHour.getTime() / 1000);
  const lastTs = Math.floor(endHour.getTime() / 1000);
  const fullRange = lastTs - firstTs;
  if (fullRange <= 0) return null;
  return { firstTs, lastTs, fullRange };
}

function getHourTicks(frames: RainViewerFrame[]): { label: string; position: number }[] {
  const range = getTimelineRange(frames);
  if (!range) return [];

  const ticks: { label: string; position: number }[] = [];
  for (let hourTs = range.firstTs; hourTs <= range.lastTs; hourTs += 3600) {
    const position = ((hourTs - range.firstTs) / range.fullRange) * 100;
    ticks.push({ label: formatTickTime(hourTs), position });
  }
  return ticks;
}

export function WeatherTimeline({
  frames,
  currentIndex,
  onIndexChange,
  isPlaying,
  onTogglePlay,
  weatherLayer,
  onWeatherLayerChange,
}: {
  frames: RainViewerFrame[];
  currentIndex: number;
  onIndexChange: (index: number) => void;
  isPlaying: boolean;
  onTogglePlay: () => void;
  weatherLayer: WeatherLayerId;
  onWeatherLayerChange: (layer: WeatherLayerId) => void;
}) {
  const trackRef = useRef<HTMLDivElement>(null);
  const currentFrame = frames[currentIndex];
  const isLive = currentIndex === frames.length - 1;
  const weatherLayerOpen = useShipmentMapStore.use.weatherLayerOpen();
  const setWeatherLayerOpen = useShipmentMapStore.use.setWeatherLayerOpen();

  const activeOption = WEATHER_OPTIONS.find((o) => o.id === weatherLayer) ?? WEATHER_OPTIONS[0];

  const dateTimeLabel = useMemo(() => {
    if (!currentFrame) return "";
    return formatFullDateTime(currentFrame.time);
  }, [currentFrame]);

  const ticks = useMemo(() => getHourTicks(frames), [frames]);
  const range = useMemo(() => getTimelineRange(frames), [frames]);

  const progress = useMemo(() => {
    if (!range || !currentFrame) return 0;
    return ((currentFrame.time - range.firstTs) / range.fullRange) * 100;
  }, [range, currentFrame]);

  const handleTrackClick = useCallback(
    (e: React.MouseEvent<HTMLDivElement>) => {
      const track = trackRef.current;
      if (!track || !range) return;
      const rect = track.getBoundingClientRect();
      const ratio = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width));
      const clickedTs = range.firstTs + ratio * range.fullRange;

      let closest = 0;
      let minDiff = Infinity;
      for (let i = 0; i < frames.length; i++) {
        const diff = Math.abs(frames[i].time - clickedTs);
        if (diff < minDiff) {
          minDiff = diff;
          closest = i;
        }
      }
      onIndexChange(closest);
    },
    [frames, range, onIndexChange],
  );

  const handleWeatherLayerChange = useCallback(
    (layer: WeatherLayerId) => {
      onWeatherLayerChange(layer);
      setWeatherLayerOpen(false);
    },
    [onWeatherLayerChange],
  );

  return (
    <MapControl position={ControlPosition.BOTTOM_CENTER}>
      <div className="mb-2.5 w-[560px] rounded-md border bg-popover shadow-sm">
        <div className="flex items-center justify-between px-3 pt-2 pb-1">
          <Popover open={weatherLayerOpen} onOpenChange={setWeatherLayerOpen}>
            <PopoverTrigger
              render={
                <button
                  type="button"
                  className="flex items-center gap-2 rounded-md px-2 py-1 text-sm hover:bg-accent"
                />
              }
            >
              <activeOption.icon className="size-4 text-foreground fill-foreground" />
              <span>{activeOption.label}</span>
              <ChevronDownIcon className="size-3 text-muted-foreground" />
            </PopoverTrigger>
            <PopoverContent
              side="top"
              sideOffset={8}
              align="start"
              className="w-auto shadow-sm p-1 gap-0"
            >
              {WEATHER_OPTIONS.map((opt) => (
                <button
                  key={opt.id}
                  type="button"
                  onClick={() => handleWeatherLayerChange(opt.id)}
                  className="flex w-full items-start gap-1.5 px-3 py-2 text-left hover:bg-accent rounded-md"
                >
                  <opt.icon className="mt-px size-4 shrink-0 text-foreground fill-foreground" />
                  <div className="min-w-0">
                    <div className="text-sm font-medium">{opt.label}</div>
                    <div className="text-2xs text-muted-foreground">{opt.description}</div>
                  </div>
                </button>
              ))}
            </PopoverContent>
          </Popover>
          <div className="flex items-center gap-0.5">
            <Button
              variant="ghost"
              size="icon-xs"
              onClick={() => onIndexChange(0)}
              disabled={currentIndex === 0}
              title="Jump to earliest"
            >
              <SkipBackIcon className="size-3.5" />
            </Button>
            <Button
              variant="ghost"
              size="icon-xs"
              onClick={() => onIndexChange(Math.max(0, currentIndex - 1))}
              disabled={currentIndex === 0}
              title="Previous frame"
            >
              <StepBackIcon className="size-3.5" />
            </Button>
            <Button
              variant="ghost"
              size="icon-xs"
              onClick={onTogglePlay}
              title={isPlaying ? "Pause" : "Play"}
            >
              {isPlaying ? <PauseIcon className="size-3.5" /> : <PlayIcon className="size-3.5" />}
            </Button>
            <Button
              variant="ghost"
              size="icon-xs"
              onClick={() => onIndexChange(Math.min(frames.length - 1, currentIndex + 1))}
              disabled={isLive}
              title="Next frame"
            >
              <StepForwardIcon className="size-3.5" />
            </Button>
            <Button
              variant="ghost"
              size="icon-xs"
              onClick={() => onIndexChange(frames.length - 1)}
              disabled={isLive}
              title="Jump to live"
            >
              <SkipForwardIcon className="size-3.5" />
            </Button>
          </div>
          <div className="flex items-center gap-1.5 text-xs">
            <span
              className={cn(
                "inline-block size-2 rounded-full",
                isLive ? "bg-green-500" : "bg-muted-foreground",
              )}
            />
            <span className="tabular-nums text-muted-foreground">{dateTimeLabel}</span>
          </div>
        </div>
        <div className="px-3 pb-2.5">
          <div ref={trackRef} className="relative h-4 cursor-pointer" onClick={handleTrackClick}>
            <div className="absolute top-1.5 right-0 left-0 h-[4px] rounded-full bg-muted" />
            <div
              className="absolute top-1.5 left-0 h-[4px] rounded-full bg-brand"
              style={{ width: `${progress}%` }}
            />
            <div
              className="absolute top-0 size-4 -translate-x-1/2 rounded-full border border-border bg-background shadow-sm"
              style={{ left: `${progress}%` }}
            />
          </div>
          <div className="relative mt-0.5 h-3">
            {ticks.map((tick, i) => (
              <span
                key={tick.label}
                className="absolute whitespace-nowrap text-[10px] tabular-nums text-muted-foreground"
                style={{
                  left: `${tick.position}%`,
                  transform:
                    i === 0
                      ? "translateX(0)"
                      : i === ticks.length - 1
                        ? "translateX(-100%)"
                        : "translateX(-50%)",
                }}
              >
                {tick.label}
              </span>
            ))}
          </div>
        </div>
      </div>
    </MapControl>
  );
}
