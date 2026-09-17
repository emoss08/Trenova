import { useT } from "@trenova/shared/i18n/use-t";
import type { CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import { coverageStanding, formatOptionalDecimalCurrency } from "@/lib/carrier-intelligence";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { ShieldAlertIcon, ShieldCheckIcon } from "lucide-react";
import {
  IntelDate,
  IntelField,
  IntelFieldGrid,
  IntelNumber,
  IntelSectionCard,
} from "./intel-section-card";
import { useCarrierIntelLabels } from "./use-carrier-intel-labels";

export type InsuranceCardProps = {
  profile: CarrierIntelProfile;
  provider: string | null | undefined;
  className?: string;
};

function CoverageRow({
  label,
  onFile,
  required,
}: {
  label: string;
  onFile: string | null;
  required: string | null;
}) {
  const t = useT();
  const standing = coverageStanding(onFile, required);
  const short = standing === "short";
  const missing = standing === "missing";

  return (
    <TableRow>
      <TableCell className="font-medium">{label}</TableCell>
      <TableCell
        className={cn(
          "text-right tabular-nums",
          (short || missing) && "font-medium text-red-700 dark:text-red-400",
        )}
      >
        {formatOptionalDecimalCurrency(onFile) ?? (missing ? t("None on file") : "-")}
      </TableCell>
      <TableCell className="text-right tabular-nums">
        {formatOptionalDecimalCurrency(required) ?? "-"}
      </TableCell>
      <TableCell className="text-right">
        {short || missing ? (
          <ShieldAlertIcon
            className="ml-auto size-4 text-red-600"
            aria-label={t("Below the required amount")}
          />
        ) : standing === "meets" ? (
          <ShieldCheckIcon
            className="ml-auto size-4 text-green-600"
            aria-label={t("Meets the required amount")}
          />
        ) : null}
      </TableCell>
    </TableRow>
  );
}

export function InsuranceCard({ profile, provider, className }: InsuranceCardProps) {
  const t = useT();
  const labels = useCarrierIntelLabels();
  const insurance = profile.insurance;

  return (
    <IntelSectionCard
      title={t("Insurance")}
      icon={ShieldCheckIcon}
      coverage={profile.coverage}
      provider={provider}
      emphasis={insurance?.pendingCancelAt ? "danger" : "none"}
      className={className}
      parts={[
        {
          section: "Insurance",
          hasData: insurance !== null,
          content: insurance ? (
            <div className="flex flex-col gap-3">
              {insurance.pendingCancelAt ? (
                <Alert variant="destructive">
                  <ShieldAlertIcon />
                  <AlertTitle>{t("Cancellation pending")}</AlertTitle>
                  <AlertDescription>
                    {t(
                      "A filing is set to cancel on {0}. Coverage lapses then unless it is replaced.",
                      formatUnixDateMedium(insurance.pendingCancelAt),
                    )}
                  </AlertDescription>
                </Alert>
              ) : null}
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("Coverage")}</TableHead>
                    <TableHead className="text-right">{t("On file")}</TableHead>
                    <TableHead className="text-right">{t("Required")}</TableHead>
                    <TableHead className="w-8" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <CoverageRow
                    label={t("BIPD")}
                    onFile={insurance.bipdOnFile}
                    required={insurance.bipdRequired}
                  />
                  <CoverageRow
                    label={t("Cargo")}
                    onFile={insurance.cargoOnFile}
                    required={insurance.cargoRequired}
                  />
                  <CoverageRow
                    label={t("Bond")}
                    onFile={insurance.bondOnFile}
                    required={insurance.bondRequired}
                  />
                </TableBody>
              </Table>
              <IntelFieldGrid>
                <IntelField label={t("Cancellations")}>
                  <IntelNumber value={insurance.cancelCount} />
                </IntelField>
                <IntelField label={t("Last canceled")}>
                  <IntelDate value={insurance.lastCanceledAt} />
                </IntelField>
              </IntelFieldGrid>
              {insurance.filings && insurance.filings.length > 0 ? (
                <div className="flex flex-col gap-1">
                  <h4 className="text-muted-foreground text-xs font-medium">{t("Filings")}</h4>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t("Type")}</TableHead>
                        <TableHead>{t("Insurer")}</TableHead>
                        <TableHead>{t("Policy")}</TableHead>
                        <TableHead className="text-right">{t("Coverage")}</TableHead>
                        <TableHead>{t("Effective")}</TableHead>
                        <TableHead>{t("Cancels")}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {insurance.filings.map((filing, index) => (
                        <TableRow key={`${filing.type}-${filing.policyNumber ?? index}-${index}`}>
                          <TableCell>{labels.filingType[filing.type]}</TableCell>
                          <TableCell>{filing.insurerName ?? "-"}</TableCell>
                          <TableCell className="font-mono text-xs">
                            {filing.policyNumber ?? "-"}
                          </TableCell>
                          <TableCell className="text-right tabular-nums">
                            {formatOptionalDecimalCurrency(filing.coverage) ?? "-"}
                          </TableCell>
                          <TableCell>
                            {filing.effectiveAt ? formatUnixDateMedium(filing.effectiveAt) : "-"}
                          </TableCell>
                          <TableCell
                            className={cn(
                              filing.cancelEffectiveAt &&
                                "font-medium text-red-700 dark:text-red-400",
                            )}
                          >
                            {filing.cancelEffectiveAt
                              ? `${formatUnixDateMedium(filing.cancelEffectiveAt)}${
                                  filing.cancelMethod ? ` · ${filing.cancelMethod}` : ""
                                }`
                              : "-"}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              ) : null}
            </div>
          ) : null,
        },
      ]}
    />
  );
}
