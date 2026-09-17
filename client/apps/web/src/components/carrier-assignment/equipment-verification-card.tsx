import { useT } from "@trenova/shared/i18n/use-t";
import { canOverrideEquipmentVerification } from "@/lib/equipment-verification";
import type { CarrierEquipmentVerification } from "@/lib/graphql/carrier-intelligence";
import type {
  CarrierEquipmentVerificationResult,
  CarrierIntelUnitType,
} from "@trenova/graphql/generated/graphql";
import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  CircleCheckIcon,
  CircleHelpIcon,
  CircleXIcon,
  SearchXIcon,
  ShieldOffIcon,
  TriangleAlertIcon,
  type LucideIcon,
} from "lucide-react";
import { useMemo } from "react";

type ResultMeta = {
  label: string;
  description: string;
  variant: BadgeVariant;
  icon: LucideIcon;
};

export function useEquipmentVerificationLabels() {
  const t = useT();

  return useMemo(
    () => ({
      result: {
        Match: {
          label: t("Match"),
          description: t("The equipment is registered to this carrier."),
          variant: "active",
          icon: CircleCheckIcon,
        },
        Mismatch: {
          label: t("Mismatch"),
          description: t("The equipment is registered to a different carrier."),
          variant: "inactive",
          icon: CircleXIcon,
        },
        NotFound: {
          label: t("Not found"),
          description: t("No carrier is registered to this equipment."),
          variant: "warning",
          icon: SearchXIcon,
        },
        Unverifiable: {
          label: t("Unverifiable"),
          description: t("The equipment could not be checked against a registration."),
          variant: "secondary",
          icon: CircleHelpIcon,
        },
        ProviderError: {
          label: t("Provider error"),
          description: t("The provider failed to answer. Try again shortly."),
          variant: "orange",
          icon: TriangleAlertIcon,
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
  const vehicle = detail
    ? [detail.year, detail.make, detail.model].filter((part) => part !== null && part !== "")
    : [];

  return (
    <article
      data-verification-result={verification.result}
      aria-label={t("Verification {0}", meta.label)}
      className={cn(
        "bg-card flex flex-col gap-2 rounded-lg border p-3",
        highlighted && "ring-ring/40 ring-2",
      )}
    >
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="flex min-w-0 flex-col gap-1">
          <div className="flex flex-wrap items-center gap-1.5">
            <Badge variant={meta.variant} className="max-h-5" title={meta.description}>
              <Icon aria-hidden />
              {meta.label}
            </Badge>
            <span className="text-xs font-medium">{labels.unitType[verification.unitType]}</span>
            <span className="text-muted-foreground truncate font-mono text-xs">
              {identifierSummary(verification)}
            </span>
            {verification.overriddenAt ? (
              <Badge variant="teal" className="max-h-5">
                <ShieldOffIcon aria-hidden />
                {t("Overridden")}
              </Badge>
            ) : null}
          </div>
          <span className="text-muted-foreground text-2xs">
            {t("Verified {0}", formatUnixDateTimeMedium(verification.verifiedAt))}
          </span>
        </div>
        {canApprove && canOverrideEquipmentVerification(verification) ? (
          <Button
            type="button"
            size="xs"
            variant="outline"
            onClick={() => onOverride(verification)}
          >
            <ShieldOffIcon />
            {t("Override")}
          </Button>
        ) : null}
      </div>

      {verification.mismatchReason ? (
        <p
          className={cn(
            "text-sm",
            verification.result === "Mismatch" && !verification.overriddenAt
              ? "text-red-700 dark:text-red-400"
              : "text-foreground",
          )}
        >
          {verification.mismatchReason}
        </p>
      ) : null}

      <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
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
        <Detail label={t("Vehicle")} value={vehicle.length > 0 ? vehicle.join(" ") : null} />
        <Detail label={t("Category")} value={detail?.category} />
        <Detail label={t("VIN")} value={detail?.vin} />
        <Detail
          label={t("Plate")}
          value={
            detail?.plateNumber
              ? `${detail.plateNumber}${detail.plateState ? ` (${detail.plateState})` : ""}`
              : null
          }
        />
        <Detail label={t("Unit number")} value={detail?.unitNumber} />
        <Detail label={t("Provider")} value={verification.provider} />
      </dl>

      {verification.overriddenAt ? (
        <p className="bg-muted/50 text-muted-foreground rounded-md px-2 py-1.5 text-xs">
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
