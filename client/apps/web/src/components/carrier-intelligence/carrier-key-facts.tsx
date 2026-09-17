import {
  coverageStanding,
  formatOptionalDecimalCurrency,
  profileAuthorityAgeDays,
} from "@/lib/carrier-intelligence";
import type { CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatPercent } from "@trenova/shared/lib/utils";
import {
  BooleanValue,
  EmptyValue,
  Fact,
  FactGrid,
  Muted,
  dateOrDash,
  numberOrDash,
  textOrDash,
} from "./intel-facts";
import { StatusDot } from "./status-dot";
import { useIntelAgeFormatter } from "./use-intel-age";

export function OutOfServiceRate({
  rate,
  national,
}: {
  rate: number | null | undefined;
  national: number | null | undefined;
}) {
  const t = useT();
  if (rate === null || rate === undefined) {
    return <EmptyValue />;
  }
  const hasNational = national !== null && national !== undefined;
  const above = hasNational && rate > national;
  return (
    <span
      className="inline-flex items-center gap-2"
      data-above-national={above ? "true" : undefined}
    >
      {above ? <StatusDot tone="medium" /> : null}
      <span>{formatPercent(rate)}</span>
      {hasNational ? <Muted>{t("national {0}", formatPercent(national))}</Muted> : null}
    </span>
  );
}

export function CoverageAmount({
  onFile,
  required,
}: {
  onFile: string | null | undefined;
  required: string | null | undefined;
}) {
  const t = useT();
  const standing = coverageStanding(onFile, required);
  const onFileText = formatOptionalDecimalCurrency(onFile);
  const requiredText = formatOptionalDecimalCurrency(required);
  if (!onFileText && !requiredText) {
    return <EmptyValue />;
  }
  return (
    <span className="inline-flex items-center gap-2" data-coverage={standing}>
      {standing === "short" || standing === "missing" ? <StatusDot tone="critical" /> : null}
      <span>{onFileText ?? t("None on file")}</span>
      {requiredText ? <Muted>{t("of {0} required", requiredText)}</Muted> : null}
    </span>
  );
}

export type CarrierKeyFactsProps = {
  profile: CarrierIntelProfile;
  className?: string;
};

export function CarrierKeyFacts({ profile, className }: CarrierKeyFactsProps) {
  const t = useT();
  const formatAge = useIntelAgeFormatter();
  const { authority, insurance, inspections, operations } = profile;

  const activeAuthorities = authority
    ? [
        authority.common?.status === "Active" ? t("Common") : null,
        authority.contract?.status === "Active" ? t("Contract") : null,
        authority.broker?.status === "Active" ? t("Broker") : null,
      ].filter((value): value is string => value !== null)
    : [];

  const age = formatAge(profileAuthorityAgeDays(profile));
  const cargo = operations?.cargoCarried?.length ? operations.cargoCarried.join(", ") : null;

  return (
    <FactGrid className={className}>
      <Fact label={t("USDOT status")}>{textOrDash(profile.identity?.usdotStatus)}</Fact>
      <Fact label={t("Operating authority")}>
        {authority ? (
          activeAuthorities.length > 0 ? (
            activeAuthorities.join(", ")
          ) : (
            <span className="inline-flex items-center gap-2">
              <StatusDot tone="critical" />
              {t("None active")}
            </span>
          )
        ) : (
          <EmptyValue />
        )}
      </Fact>
      <Fact label={t("Authority age")}>{age ?? <EmptyValue />}</Fact>
      <Fact label={t("Safety rating")}>
        {profile.safety ? (
          (profile.safety.rating ?? <Muted>{t("Not rated")}</Muted>)
        ) : (
          <EmptyValue />
        )}
      </Fact>
      <Fact label={t("BIPD insurance")}>
        <CoverageAmount onFile={insurance?.bipdOnFile} required={insurance?.bipdRequired} />
      </Fact>
      <Fact label={t("Cargo insurance")}>
        <CoverageAmount onFile={insurance?.cargoOnFile} required={insurance?.cargoRequired} />
      </Fact>
      <Fact label={t("Bond")}>
        <CoverageAmount onFile={insurance?.bondOnFile} required={insurance?.bondRequired} />
      </Fact>
      <Fact label={t("Cargo carried")}>
        <span title={cargo ?? undefined}>{textOrDash(cargo)}</span>
      </Fact>
      <Fact label={t("Driver out-of-service")}>
        <OutOfServiceRate
          rate={inspections?.driverOosRate}
          national={inspections?.nationalDriverOosRate}
        />
      </Fact>
      <Fact label={t("Vehicle out-of-service")}>
        <OutOfServiceRate
          rate={inspections?.vehicleOosRate}
          national={inspections?.nationalVehicleOosRate}
        />
      </Fact>
      <Fact label={t("Power units")}>{numberOrDash(profile.fleet?.powerUnits)}</Fact>
      <Fact label={t("Drivers")}>{numberOrDash(profile.fleet?.drivers)}</Fact>
      <Fact label={t("Hazmat")}>
        <BooleanValue value={operations?.hazmatCarrier} />
      </Fact>
      <Fact label={t("MCS-150 filed")}>
        {operations?.mcs150At ? (
          <span>
            {dateOrDash(operations.mcs150At)}
            {operations.mcs150Mileage !== null ? (
              <Muted>
                {" · "}
                {t("{0} mi", formatNumber(operations.mcs150Mileage))}
              </Muted>
            ) : null}
          </span>
        ) : (
          <EmptyValue />
        )}
      </Fact>
    </FactGrid>
  );
}
