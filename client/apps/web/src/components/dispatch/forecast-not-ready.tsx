import { usePermission } from "@/hooks/use-permission";
import {
  forecastStanding,
  forecastStandingCopy,
  type ForecastStanding,
} from "@/lib/dispatch-forecast";
import type { DispatchForecastQuality } from "@/lib/graphql/dispatch-console";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { Link } from "react-router";

const DISPATCH_CONTROLS_PATH = "/admin/dispatch-controls/";

const STANDING_ALERT: Record<Exclude<ForecastStanding, "Ready">, "info" | "warning"> = {
  Unfitted: "info",
  InsufficientHistory: "info",
  NoSkill: "warning",
};

/**
 * Why there is no outlook yet: the forecast has not proven itself on the organization's own
 * history. The page and the console's Outlook tab say it the same way.
 */
export function ForecastNotReady({ quality }: { quality: DispatchForecastQuality }) {
  const t = useT();
  const standing = forecastStanding(quality);
  const copy = forecastStandingCopy(quality, t);
  const { allowed: canOpenControls } = usePermission(Resource.DispatchControl, Operation.Read);

  if (standing === "Ready") return null;

  return (
    <Alert size="sm" variant={STANDING_ALERT[standing]}>
      <AlertTitle>{t("Freight forecast: {0}", copy.title)}</AlertTitle>
      <AlertDescription className="flex flex-col gap-1">
        <span>{copy.description}</span>
        <span>
          {t(
            "The outlook and Anticipate freight use the forecast only once it is Ready, so drivers are not moved for freight it cannot yet predict.",
          )}
        </span>
        {canOpenControls ? (
          <Link
            to={DISPATCH_CONTROLS_PATH}
            className="text-brand ui-focus-ring self-start rounded-sm font-medium"
          >
            {t("See the forecast in Dispatch controls")}
          </Link>
        ) : null}
      </AlertDescription>
    </Alert>
  );
}
