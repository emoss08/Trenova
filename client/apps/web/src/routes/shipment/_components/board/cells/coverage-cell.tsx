import { useT } from "@trenova/shared/i18n/use-t";
import { AlertTriangleIcon } from "@trenova/shared/components/icons";
import { formatClockDurationMs } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { Shipment } from "@trenova/shared/types/shipment";
import { resolveCoverage } from "@/lib/shipment-board/coverage";
import { useDriverHos } from "../use-driver-hos";
import { IdentityAvatar } from "../identity-avatar";

export function CoverageCell({ shipment }: { shipment: Shipment }) {
  const t = useT();
  const coverage = resolveCoverage(shipment);
  const driveRemainingMs = useDriverHos(coverage.kind === "driver" ? coverage.id : null);

  if (coverage.kind === "uncovered") {
    return (
      <span className="text-warning inline-flex items-center gap-1 text-sm font-medium">
        <AlertTriangleIcon className="size-3.5" aria-hidden />
        {t("Needs coverage")}
      </span>
    );
  }

  if (coverage.kind === "tendered") {
    return (
      <div className="flex min-w-0 items-center gap-2">
        <span
          aria-hidden
          className="border-border-strong size-6 shrink-0 rounded-sm border border-dashed opacity-70"
        />
        <span className="text-muted-foreground truncate text-sm">{t("Tendered · awaiting")}</span>
      </div>
    );
  }

  const isCarrier = coverage.kind === "carrier";
  const detail = [
    coverage.detail,
    driveRemainingMs != null ? t("{0} HOS", formatClockDurationMs(driveRemainingMs)) : null,
  ]
    .filter(Boolean)
    .join(" · ");

  return (
    <div className="flex min-w-0 items-center gap-2">
      <IdentityAvatar
        id={coverage.id}
        initials={coverage.initials}
        shape={isCarrier ? "square" : "circle"}
        className="size-6 text-2xs"
      />
      <div className="flex min-w-0 flex-col">
        <span className="truncate text-sm font-medium">{coverage.name}</span>
        {detail ? (
          <span
            className={cn(
              "text-muted-foreground truncate font-mono text-xs tabular-nums",
              "in-data-[density=compact]:hidden",
            )}
          >
            {detail}
          </span>
        ) : null}
      </div>
    </div>
  );
}
