import {
  FORECAST_STANDING_TONE,
  forecastStanding,
  forecastStandingCopy,
  formatLoads,
} from "@/lib/dispatch-forecast";
import type { DispatchForecastQuality } from "@/lib/graphql/dispatch-console";
import { dispatchConsoleQueries } from "@/lib/queries/dispatch-console";
import type { DispatchControl } from "@/types/dispatch-control";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { useFormContext, useWatch } from "react-hook-form";
import { FreightForecastSkeleton } from "./freight-forecast-skeleton";

/**
 * How the organization's freight forecast did when it was last tested on its own history.
 * Anticipate freight acts only on a forecast that is Ready, so this is where a dispatcher
 * learns whether turning it on changes anything yet.
 */
export function FreightForecastCard() {
  const t = useT();
  const { data, isPending, isError, refetch, isRefetching } = useQuery(
    dispatchConsoleQueries.forecastQuality(),
  );

  if (isPending) {
    return <FreightForecastSkeleton />;
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Freight forecast")}</CardTitle>
        <CardDescription>
          {t(
            "Where freight usually appears, learned from your shipment history and tested every night on the weeks it has not seen. Anticipate freight changes plans only while the forecast is Ready.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="flex max-w-prose flex-col gap-4">
        {isError ? (
          <Alert size="sm" variant="destructive">
            <AlertDescription className="flex items-center justify-between gap-3">
              <span>{t("The freight forecast could not be loaded.")}</span>
              <Button
                type="button"
                variant="outline"
                size="sm"
                isLoading={isRefetching}
                onClick={() => void refetch()}
              >
                {t("Try again")}
              </Button>
            </AlertDescription>
          </Alert>
        ) : (
          <ForecastQualityBody quality={data} />
        )}
      </CardContent>
    </Card>
  );
}

function ForecastQualityBody({ quality }: { quality: DispatchForecastQuality }) {
  const t = useT();
  const { control } = useFormContext<DispatchControl>();
  const anticipating = useWatch({ control, name: "anticipateFreight" });
  const standing = forecastStanding(quality);
  const copy = forecastStandingCopy(quality, t);

  return (
    <>
      <div className="flex flex-col items-start gap-1.5">
        <Badge variant={FORECAST_STANDING_TONE[standing]}>{copy.title}</Badge>
        <p className="text-sm">{copy.description}</p>
      </div>

      {anticipating && standing !== "Ready" ? (
        <Alert size="sm" variant="warning">
          <AlertDescription>
            {t("Anticipate freight is on, but plans ignore the forecast until it is Ready.")}
          </AlertDescription>
        </Alert>
      ) : null}

      {quality.fittedAt ? (
        <DescriptionList columns={2} aria-label={t("Forecast figures")}>
          {standing !== "InsufficientHistory" ? (
            <DescriptionItem label={t("History")} numeric>
              {t(
                "{0, plural, one {# week} other {# weeks}}, {1, plural, one {# load} other {# loads}}",
                quality.historyWeeks,
                quality.historyLoads,
              )}
            </DescriptionItem>
          ) : null}
          {standing === "Ready" ? (
            <DescriptionItem label={t("Average miss")} numeric>
              {t(
                "{0} loads per area and slot, against {1} repeating last week",
                formatLoads(quality.meanAbsoluteError),
                formatLoads(quality.naiveMeanAbsoluteError),
              )}
            </DescriptionItem>
          ) : null}
          <DescriptionItem label={t("Last fitted")}>
            {formatUnixDateTimeMedium(quality.fittedAt)}
          </DescriptionItem>
        </DescriptionList>
      ) : null}
    </>
  );
}
