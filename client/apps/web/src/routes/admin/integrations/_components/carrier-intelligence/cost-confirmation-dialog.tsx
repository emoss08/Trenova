import {
  carrierIntelProviderLabel,
  formatOptionalDecimalCurrency,
} from "@/lib/carrier-intelligence";
import type { CarrierIntelCostEstimate } from "@/lib/graphql/carrier-intel-settings";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { enrollmentPolicyChoices } from "./carrier-intel-settings-schema";

export function CostConfirmationDialog({
  estimate,
  open,
  isSaving,
  onConfirm,
  onCancel,
}: {
  estimate: CarrierIntelCostEstimate | null;
  open: boolean;
  isSaving: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  const t = useT();

  const policyLabel =
    enrollmentPolicyChoices.find((choice) => choice.value === estimate?.policy)?.label ?? "";

  return (
    <AlertDialog
      open={open}
      onOpenChange={(next) => {
        if (!next && !isSaving) {
          onCancel();
        }
      }}
    >
      <AlertDialogContent className="sm:max-w-md">
        <AlertDialogHeader>
          <AlertDialogTitle>{t("Confirm monitoring cost")}</AlertDialogTitle>
          <AlertDialogDescription>
            {t(
              "Monitoring is billed per enrolled carrier each month. Review the estimate before enrolling these carriers.",
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        {estimate ? (
          <dl className="border-border grid grid-cols-2 gap-x-4 gap-y-2 rounded-md border p-3 text-sm">
            <dt className="text-muted-foreground">{t("Provider")}</dt>
            <dd className="text-right font-medium">
              {carrierIntelProviderLabel(estimate.provider)}
            </dd>
            <dt className="text-muted-foreground">{t("Enrollment policy")}</dt>
            <dd className="text-right font-medium">{t(policyLabel)}</dd>
            <dt className="text-muted-foreground">{t("Carriers to monitor")}</dt>
            <dd className="text-right font-medium" data-testid="cost-estimate-subjects">
              {estimate.subjectCount.toLocaleString()}
            </dd>
            <dt className="text-muted-foreground">{t("Per carrier")}</dt>
            <dd className="text-right font-medium">
              {formatOptionalDecimalCurrency(estimate.perSubject) ?? "-"}
            </dd>
            <dt className="text-muted-foreground">{t("Estimated monthly cost")}</dt>
            <dd className="text-right text-base font-semibold" data-testid="cost-estimate-monthly">
              {formatOptionalDecimalCurrency(estimate.monthlyMonitoring) ?? "-"}
            </dd>
          </dl>
        ) : null}
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isSaving}>{t("Cancel")}</AlertDialogCancel>
          <Button
            type="button"
            onClick={onConfirm}
            isLoading={isSaving}
            loadingText={t("Saving...")}
            disabled={!estimate}
          >
            {t("Confirm and save")}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
