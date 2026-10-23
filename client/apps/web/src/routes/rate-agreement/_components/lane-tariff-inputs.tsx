import { TariffInputsFields } from "@/components/tariff-inputs/tariff-inputs-fields";
import { TariffInputsFieldsSkeleton } from "@/components/tariff-inputs/tariff-inputs-fields-skeleton";
import { useTemplateInputs } from "@/hooks/use-template-inputs";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { AlertTriangleIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { describeApiError } from "@/lib/api-error-message";
import type { RateAgreement } from "@trenova/shared/types/rate";
import type { TariffInputValues } from "@trenova/shared/types/tariff";
import { useFormContext, useWatch } from "react-hook-form";

/**
 * One lane's terms: the inputs its tariff declares, as fields under `rules.${index}.inputs`.
 * A value the lane leaves unset takes the agreement's, then the tariff's default, which each
 * field's placeholder shows.
 */
export function LaneTariffInputs({
  index,
  templateId,
}: {
  index: number;
  templateId: string | null | undefined;
}) {
  const t = useT();
  const { control } = useFormContext<RateAgreement>();
  const agreementInputs = useWatch({ control, name: "inputs" }) as TariffInputValues | undefined;
  const agreementCurrency = useWatch({ control, name: "currency" });
  const ruleCurrency = useWatch({ control, name: `rules.${index}.currency` });
  const declarations = useTemplateInputs(templateId);

  if (!templateId) {
    return (
      <p className="text-muted-foreground text-xs">
        {t("Choose a tariff to set this lane's terms.")}
      </p>
    );
  }

  if (declarations.isError) {
    return (
      <Alert variant="destructive" size="sm">
        <AlertTriangleIcon className="size-4" />
        <AlertDescription>
          {describeApiError(declarations.error, t("The tariff's inputs could not be read."))}
        </AlertDescription>
      </Alert>
    );
  }

  if (!declarations.data) {
    return <TariffInputsFieldsSkeleton />;
  }

  return (
    <TariffInputsFields
      declarations={declarations.data}
      name={`rules.${index}.inputs`}
      inherited={agreementInputs}
      inheritedFrom="agreement"
      currency={ruleCurrency || agreementCurrency}
    />
  );
}
