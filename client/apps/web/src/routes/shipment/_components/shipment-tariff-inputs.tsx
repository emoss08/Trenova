import { TariffInputsFields } from "@/components/tariff-inputs/tariff-inputs-fields";
import { TariffInputsFieldsSkeleton } from "@/components/tariff-inputs/tariff-inputs-fields-skeleton";
import { useTemplateInputs } from "@/hooks/use-template-inputs";
import { describeApiError } from "@/lib/api-error-message";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { AlertTriangleIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useFormContext, useWatch } from "react-hook-form";

/**
 * The shipment's own values for its tariff's inputs. Adopting a contract writes the values it
 * resolved here, and every re-rate reads them back; a manual rate has no tariff and no inputs.
 */
export function ShipmentTariffInputs() {
  const t = useT();
  const { control } = useFormContext<Shipment>();
  const templateId = useWatch({ control, name: "formulaTemplateId" });
  const rateLocked = Boolean(useWatch({ control, name: "rateLocked" }));
  const declarations = useTemplateInputs(templateId);

  if (!templateId) {
    return (
      <p className="text-muted-foreground text-xs">
        {t("Choose a rating method to set the values its tariff reads.")}
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
    return <TariffInputsFieldsSkeleton fields={2} />;
  }

  return (
    <TariffInputsFields
      declarations={declarations.data}
      name="tariffInputs"
      disabled={rateLocked}
    />
  );
}
