import { ForecastChecked, ForecastHistoryMeters } from "@/components/dispatch/forecast-history";
import { usePermission } from "@/hooks/use-permission";
import { forecastStanding, forecastStandingCopy } from "@/lib/dispatch-forecast";
import type { DispatchForecastQuality } from "@/lib/graphql/dispatch-console";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet } from "@trenova/shared/components/ui/empty-sheet";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { ReactNode } from "react";
import { Link } from "react-router";

const DISPATCH_CONTROLS_PATH = "/admin/dispatch-controls/";

/**
 * Why there is no outlook yet, drawn as the page it will become: the forecast has not proven
 * itself on the organization's own history. It says what is missing, how far the history has
 * got, when that was last counted and whether it is being counted again now.
 */
export function ForecastNotReady({
  quality,
  sketch,
}: {
  quality: DispatchForecastQuality;
  /** A faint drawing of the surface the forecast will fill. */
  sketch: ReactNode;
}) {
  const t = useT();
  const standing = forecastStanding(quality);
  const copy = forecastStandingCopy(quality, t);
  const { allowed: canOpenControls } = usePermission(Resource.DispatchControl, Operation.Read);

  if (standing === "Ready") return null;

  return (
    <EmptySheet
      sketch={sketch}
      title={copy.title}
      description={[
        copy.description,
        t(
          "The outlook and Anticipate freight use the forecast only once it is Ready, so drivers are not moved for freight it cannot yet predict.",
        ),
      ].join("\n")}
      action={
        canOpenControls ? (
          <Button size="sm" nativeButton={false} render={<Link to={DISPATCH_CONTROLS_PATH} />}>
            {t("See the forecast in Dispatch controls")}
          </Button>
        ) : undefined
      }
    >
      {standing === "InsufficientHistory" ? (
        <ForecastHistoryMeters quality={quality} className="mt-3 w-full max-w-xs" />
      ) : null}
      <ForecastChecked quality={quality} className="mt-2 text-left" />
    </EmptySheet>
  );
}
