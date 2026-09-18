import { useT } from "@trenova/shared/i18n/use-t";
import { RelativeTime } from "@/components/carrier-intelligence/relative-time";
import { carrierIntelProviderLabel, joinPresent } from "@/lib/carrier-intelligence";
import { canOverrideEquipmentVerification } from "@/lib/equipment-verification";
import type { CarrierEquipmentVerification } from "@/lib/graphql/carrier-intelligence";
import type {
  CarrierEquipmentVerificationResult,
  CarrierIntelUnitType,
} from "@trenova/graphql/generated/graphql";
import { Button } from "@trenova/shared/components/ui/button";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  CircleCheckIcon,
  CircleHelpIcon,
  CircleXIcon,
  SearchXIcon,
  TriangleAlertIcon,
  type LucideIcon,
} from "lucide-react";
import { useMemo } from "react";

type ResultMeta = {
  label: string;
  description: string;
  icon: LucideIcon;
  iconClassName: string;
};

export function useEquipmentVerificationLabels() {
  const t = useT();

  return useMemo(
    () => ({
      result: {
        Match: {
          label: t("Match"),
          description: t("The equipment is registered to this carrier."),
          icon: CircleCheckIcon,
          iconClassName: "text-success-foreground",
        },
        Mismatch: {
          label: t("Mismatch"),
          description: t("The equipment is registered to a different carrier."),
          icon: CircleXIcon,
          iconClassName: "text-danger-foreground",
        },
        NotFound: {
          label: t("Not found"),
          description: t("No carrier is registered to this equipment."),
          icon: SearchXIcon,
          iconClassName: "text-warning-foreground",
        },
        Unverifiable: {
          label: t("Unverifiable"),
          description: t("The equipment could not be checked against a registration."),
          icon: CircleHelpIcon,
          iconClassName: "text-muted-foreground",
        },
        ProviderError: {
          label: t("Provider error"),
          description: t("The provider failed to answer. Try again shortly."),
          icon: TriangleAlertIcon,
          iconClassName: "text-warning-foreground",
        },
      } satisfies Record<CarrierEquipmentVerificationResult, ResultMeta>,
      unitType: {
        Tractor: t("Tractor"),
        Trailer: t("Trailer"),
        Straight: t("Straight truck"),
      } satisfies Record<CarrierIntelUnitType, string>,
    }),
    [t],
  );
}

function identifierSummary(verification: CarrierEquipmentVerification): string {
  if (verification.vin) {
    return `VIN ${verification.vin}`;
  }
  if (verification.plateNumber) {
    return verification.plateState
      ? `${verification.plateNumber} (${verification.plateState})`
      : verification.plateNumber;
  }
  return verification.unitNumber ? `#${verification.unitNumber}` : "";
}

function Detail({ label, value }: { label: string; value: string | number | null | undefined }) {
  if (value === null || value === undefined || value === "") {
    return null;
  }
  return (
    <>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="min-w-0 break-words">{value}</dd>
    </>
  );
}

export function EquipmentVerificationCard({
  verification,
  canApprove,
  highlighted = false,
  onOverride,
}: {
  verification: CarrierEquipmentVerification;
  canApprove: boolean;
  highlighted?: boolean;
  onOverride: (verification: CarrierEquipmentVerification) => void;
}) {
  const t = useT();
  const labels = useEquipmentVerificationLabels();
  const meta = labels.result[verification.result];
  const Icon = meta.icon;
  const detail = verification.detail;
  const overridden = verification.overriddenAt !== null;

  return (
    <article
      data-verification-result={verification.result}
      data-highlighted={highlighted ? "true" : undefined}
      aria-label={t("Verification {0}", meta.label)}
      className={cn("flex flex-col gap-2 py-3", highlighted && "bg-muted/40 -mx-3 rounded-md px-3")}
    >
      <div className="flex items-start gap-2.5">
        <Icon
          className={cn(
            "mt-0.5 size-3.5 shrink-0",
            overridden ? "text-muted-foreground" : meta.iconClassName,
          )}
          aria-hidden
        />
        <div className="flex min-w-0 flex-1 flex-col gap-0.5">
          <div className="flex flex-wrap items-baseline gap-x-2 text-sm">
            <span className="font-medium" title={meta.description}>
              {meta.label}
            </span>
            {overridden ? <span className="text-muted-foreground">{t("Overridden")}</span> : null}
            <span className="text-muted-foreground text-xs">
              {labels.unitType[verification.unitType]}
            </span>
            <span className="text-muted-foreground truncate font-mono text-xs">
              {identifierSummary(verification)}
            </span>
          </div>
          {verification.mismatchReason ? (
            <p className="text-sm">{verification.mismatchReason}</p>
          ) : null}
          <span className="text-muted-foreground inline-flex items-center gap-1 text-xs">
            {t("Verified")} <RelativeTime timestamp={verification.verifiedAt} />
          </span>
        </div>
        {canApprove && canOverrideEquipmentVerification(verification) ? (
          <Button type="button" size="xs" variant="ghost" onClick={() => onOverride(verification)}>
            {t("Override")}
          </Button>
        ) : null}
      </div>

      <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 pl-6 text-xs">
        <Detail label={t("Expected DOT")} value={verification.expectedDotNumber} />
        <Detail
          label={t("Registered DOT")}
          value={
            verification.matchedDotNumbers.length > 0
              ? verification.matchedDotNumbers.join(", ")
              : null
          }
        />
        <Detail label={t("Registered to")} value={verification.matchedLegalName} />
        <Detail
          label={t("Vehicle")}
          value={detail ? joinPresent([detail.year, detail.make, detail.model], " ") : null}
        />
        <Detail label={t("Category")} value={detail?.category} />
        <Detail label={t("VIN")} value={detail?.vin} />
        <Detail
          label={t("Plate")}
          value={
            detail?.plateNumber
              ? joinPresent(
                  [detail.plateNumber, detail.plateState ? `(${detail.plateState})` : null],
                  " ",
                )
              : null
          }
        />
        <Detail label={t("Unit number")} value={detail?.unitNumber} />
        <Detail
          label={t("Provider")}
          value={verification.provider ? carrierIntelProviderLabel(verification.provider) : null}
        />
      </dl>

      {verification.overriddenAt ? (
        <p className="text-muted-foreground pl-6 text-xs">
          {t(
            "Overridden {0}: {1}",
            formatUnixDateTimeMedium(verification.overriddenAt),
            verification.overrideReason ?? "",
          )}
        </p>
      ) : null}
    </article>
  );
}
