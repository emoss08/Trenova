import { useT } from "@trenova/shared/i18n/use-t";
import { FuelSurchargeProgramAutocompleteField } from "@/components/autocomplete-fields";
import { NumberField } from "@/components/fields/number-field";
import { SwitchField } from "@/components/fields/switch-field";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import type { RateAgreement } from "@trenova/shared/types/rate";
import { InfoIcon } from "lucide-react";
import { useFormContext, useWatch } from "react-hook-form";

const NEW_BINDING = {
  fuelSurchargeProgramId: "",
  waived: false,
  pegPriceOverride: null,
  incrementRateOverride: null,
  capAmount: null,
};

/**
 * The contract's fuel terms.
 *
 * Fuel is where invoice disputes concentrate, and the reason is nearly always
 * that the contract said something the system did not know: a different peg, a
 * cap, or an all-in rate with no surcharge at all. Those three are stated here
 * rather than left to whoever reads the contract.
 */
export function FuelBindingForm() {
  const t = useT();

  const { control, setValue } = useFormContext<RateAgreement>();
  const binding = useWatch({ control, name: "fuelBinding" });
  const waived = Boolean(binding?.waived);

  if (!binding) {
    return (
      <div className="flex flex-col gap-3">
        <p className="text-muted-foreground text-sm">
          {t("This contract has no fuel terms of its own, so fuel comes from the customer's billing profile.")}
        </p>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="self-start"
          onClick={() => setValue("fuelBinding", NEW_BINDING as never, { shouldDirty: true })}
        >
          {t("Add fuel terms")}
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <FormSection
        title={t("Fuel Terms")}
        description={t("The program this contract's fuel surcharge reads, and the terms it negotiated over it.")}
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <FuelSurchargeProgramAutocompleteField
              control={control}
              rules={{ required: !waived }}
              name="fuelBinding.fuelSurchargeProgramId"
              label={t("Fuel Program")}
              placeholder={t("Select program")}
              description={t("Overrides whatever the customer's billing profile names, because one customer can hold several contracts")}
            />
          </FormControl>

          <FormControl cols="full">
            <SwitchField
              control={control}
              name="fuelBinding.waived"
              label={t("Fuel Included in Rate")}
              description={t("An all-in rate, with no surcharge billed at all")}
              outlined
            />
          </FormControl>
        </FormGroup>

        {waived ? (
          <Alert>
            <InfoIcon className="size-4" />
            <AlertDescription>
              {t("A waived binding cannot also change the program's terms — the two describe opposite intentions.")}
            </AlertDescription>
          </Alert>
        ) : (
          <FormGroup cols={3}>
            <FormControl>
              <NumberField
                control={control}
                name="fuelBinding.pegPriceOverride"
                label={t("Peg Price Override")}
                placeholder="0.00"
                sideText="$"
                decimalScale={3}
                description={t("The price the surcharge starts climbing from, when the contract negotiated its own")}
              />
            </FormControl>
            <FormControl>
              <NumberField
                control={control}
                name="fuelBinding.incrementRateOverride"
                label={t("Increment Rate Override")}
                placeholder="0.00"
                sideText="$"
                decimalScale={4}
                description={t("What each step above the peg adds, when the contract negotiated its own")}
              />
            </FormControl>
            <FormControl>
              <NumberField
                control={control}
                name="fuelBinding.capAmount"
                label={t("Cap Amount")}
                placeholder="0.00"
                sideText="$"
                decimalScale={2}
                thousandSeparator
                description={t("The most fuel this contract can ever be billed on one shipment")}
              />
            </FormControl>
          </FormGroup>
        )}

        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="self-start"
          onClick={() => setValue("fuelBinding", null, { shouldDirty: true })}
        >
          {t("Remove fuel terms")}
        </Button>
      </FormSection>
    </div>
  );
}
