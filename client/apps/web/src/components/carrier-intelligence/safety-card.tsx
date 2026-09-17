import { useT } from "@trenova/shared/i18n/use-t";
import type { CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import { fmcsaSafetyRatingTone, probabilityToPercent } from "@/lib/carrier-intelligence";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { formatNumber } from "@trenova/shared/i18n/format";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { formatPercent } from "@trenova/shared/lib/utils";
import { OctagonXIcon, ShieldIcon } from "lucide-react";
import {
  IntelDate,
  IntelField,
  IntelFieldGrid,
  IntelNumber,
  IntelSectionCard,
  IntelValue,
} from "./intel-section-card";

export type SafetyCardProps = {
  profile: CarrierIntelProfile;
  provider: string | null | undefined;
  className?: string;
};

export function SafetyCard({ profile, provider, className }: SafetyCardProps) {
  const t = useT();
  const safety = profile.safety;

  return (
    <IntelSectionCard
      title={t("Safety")}
      icon={ShieldIcon}
      coverage={profile.coverage}
      provider={provider}
      emphasis={safety?.outOfServiceOrder ? "danger" : "none"}
      className={className}
      parts={[
        {
          section: "Safety",
          hasData: safety !== null,
          content: safety ? (
            <div className="flex flex-col gap-3">
              {safety.outOfServiceOrder ? (
                <Alert variant="destructive">
                  <OctagonXIcon />
                  <AlertTitle>{t("Out-of-service order")}</AlertTitle>
                  <AlertDescription>
                    {safety.outOfServiceAt
                      ? t(
                          "The FMCSA placed this carrier out of service on {0}. It may not operate.",
                          formatUnixDateMedium(safety.outOfServiceAt),
                        )
                      : t("The FMCSA placed this carrier out of service. It may not operate.")}
                  </AlertDescription>
                </Alert>
              ) : null}
              <IntelFieldGrid>
                <IntelField label={t("Safety rating")}>
                  {safety.rating ? (
                    <Badge variant={fmcsaSafetyRatingTone(safety.rating)} className="max-h-5">
                      {safety.rating}
                    </Badge>
                  ) : (
                    <span className="text-muted-foreground">{t("Not rated")}</span>
                  )}
                </IntelField>
                <IntelField label={t("Rating date")}>
                  <IntelDate value={safety.ratingDate} />
                </IntelField>
                <IntelField label={t("Inspection Selection System (ISS)")}>
                  {safety.issValue !== null ? (
                    <span className="tabular-nums">
                      {formatNumber(safety.issValue)}
                      {safety.issRecommendation ? (
                        <span className="text-muted-foreground"> · {safety.issRecommendation}</span>
                      ) : null}
                    </span>
                  ) : (
                    <IntelValue value={safety.issRecommendation} />
                  )}
                </IntelField>
                <IntelField label={t("Risk score")}>
                  {safety.riskScore ? (
                    <span>
                      {safety.riskScore}
                      {safety.riskProbability !== null ? (
                        <span className="text-muted-foreground tabular-nums">
                          {" "}
                          · {formatPercent(probabilityToPercent(safety.riskProbability))}
                        </span>
                      ) : null}
                    </span>
                  ) : (
                    <span className="text-muted-foreground">-</span>
                  )}
                </IntelField>
                <IntelField label={t("Safety score")}>
                  <IntelNumber value={safety.safetyScore} options={{ maximumFractionDigits: 1 }} />
                </IntelField>
                <IntelField label={t("Out-of-service order")}>
                  {safety.outOfServiceOrder === null ? (
                    <span className="text-muted-foreground">-</span>
                  ) : safety.outOfServiceOrder ? (
                    <Badge variant="inactive" className="max-h-5">
                      {t("In effect")}
                    </Badge>
                  ) : (
                    <span>{t("None")}</span>
                  )}
                </IntelField>
                <IntelField label={t("Latest review")}>
                  <IntelValue value={safety.latestReviewType} />
                </IntelField>
                <IntelField label={t("Latest review date")}>
                  <IntelDate value={safety.latestReviewAt} />
                </IntelField>
              </IntelFieldGrid>
            </div>
          ) : null,
        },
      ]}
    />
  );
}
