/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { Switch } from "@/components/ui/switch";

interface LiveModeStatusProps {
  enabled: boolean;
  connected: boolean;
  showToggle?: boolean;
  onToggle?: (enabled: boolean) => void;
  autoRefresh?: boolean;
  onAutoRefreshToggle?: (autoRefresh: boolean) => void;
}

export function LiveModeStatus({
  enabled,
  connected,
  showToggle = false,
  onToggle,
  autoRefresh = false,
  onAutoRefreshToggle,
}: LiveModeStatusProps) {
  return (
    <div className="bg-blue-500/10 border border-blue-600/50 p-2 rounded-md flex justify-between items-center mb-4 w-full">
      <div className="flex items-center gap-3 text-blue-600">
        <div className="flex items-center gap-2">
          <div
            className={`w-2 h-2 rounded-full ${
              connected ? "bg-green-500 animate-pulse" : "bg-red-500"
            }`}
          />
          <span className="text-sm font-medium">
            Live Mode {connected ? "Active" : "Disconnected"}
          </span>
        </div>
      </div>

      {showToggle && (
        <div className="flex items-center gap-2">
          {onAutoRefreshToggle && (
            <div className="flex items-center gap-2">
              <span className="text-xs">Auto-refresh</span>
              <Switch
                checked={autoRefresh}
                onCheckedChange={onAutoRefreshToggle}
                size="sm"
              />
            </div>
          )}

          {onToggle && (
            <div className="flex items-center gap-2">
              <span className="text-xs">Live Mode</span>
              <Switch checked={enabled} onCheckedChange={onToggle} size="sm" />
            </div>
          )}
        </div>
      )}
    </div>
  );
}
