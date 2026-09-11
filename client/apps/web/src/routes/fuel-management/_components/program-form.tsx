import { useT } from "@trenova/shared/i18n/use-t";
import {
  AccessorialChargeAutocompleteField,
  EquipmentTypeMultiSelectField,
  FuelIndexAutocompleteField,
  ServiceTypeMultiSelectField,
  ShipmentTypeMultiSelectField,
} from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Card, CardContent, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import {
  fuelSurchargeDateBasisChoices,
  fuelSurchargeEffectiveDayChoices,
  fuelSurchargeFallbackChoices,
  fuelSurchargeMethodChoices,
  fuelSurchargePercentBasisChoices,
  fuelSurchargeProgramStatusChoices,
  fuelSurchargeRateRoundingChoices,
  fuelSurchargeStepRoundingChoices,
} from "@/lib/choices";
import type { FuelSurchargeProgramFormValues } from "@/types/fuel-surcharge";
import { useFormContext, useWatch } from "react-hook-form";
import { BandTableEditor } from "./band-table-editor";
import { VirtualMatrixPreview } from "./virtual-matrix-preview";

const TABLE_METHODS = new Set(["TablePerMile", "TablePercent", "TableFlat"]);

export function ProgramForm({ disabled }: { disabled?: boolean }) {
  const t = useT();

  const { control } = useFormContext<FuelSurchargeProgramFormValues>();
  const method = useWatch({ control, name: "method" });
  const isTableMethod = TABLE_METHODS.has(method);

  return (
    <div className="flex flex-col gap-4">
      <FormGroup cols={2}>
        <FormControl>
          <InputField
            control={control}
            name="name"
            label={t("Name")}
            placeholder={t("Acme National DOE Program")}
            rules={{ required: true }}
            maxLength={100}
            description={t("Shown on customer profiles and the fuel dashboard.")}
            disabled={disabled}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="code"
            label={t("Code")}
            placeholder={t("FSC-DOE-STD")}
            rules={{ required: true }}
            maxLength={50}
            description={t("Short unique identifier for this program.")}
            disabled={disabled}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="status"
            label={t("Status")}
            options={fuelSurchargeProgramStatusChoices}
            description={t("Inactive programs stop applying surcharges immediately.")}
            isReadOnly={disabled}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="method"
            label={t("Method")}
            rules={{ required: true }}
            options={fuelSurchargeMethodChoices}
            description={t("Formula methods compute rates from parameters; table methods use explicit price bands.")}
            isReadOnly={disabled}
          />
        </FormControl>
        <FormControl>
          <FuelIndexAutocompleteField
            control={control}
            name="fuelIndexId"
            label={t("Fuel Index")}
            placeholder={t("Select Fuel Index")}
            rules={{ required: true }}
            description={t("The weekly price series this program keys off (DOE region or custom index).")}
          />
        </FormControl>
        <FormControl>
          <AccessorialChargeAutocompleteField
            control={control}
            name="accessorialChargeId"
            label={t("Accessorial Charge")}
            placeholder={t("Select Accessorial Charge")}
            rules={{ required: true }}
            description={t("The catalog charge the generated fuel surcharge line posts against.")}
          />
        </FormControl>
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="description"
            label={t("Description")}
            placeholder={t("Contract terms, customer references, or maintenance notes")}
            disabled={disabled}
          />
        </FormControl>
      </FormGroup>

      <MethodParameters disabled={disabled} method={method} />

      {isTableMethod ? (
        <BandTableEditor disabled={disabled} method={method} />
      ) : (
        <VirtualMatrixPreview disabled={disabled} />
      )}

      <WeekAndRoundingSection disabled={disabled} method={method} />
      <ApplicabilitySection />
    </div>
  );
}

function MethodParameters({ disabled, method }: { disabled?: boolean; method: string }) {
  const t = useT();

  const { control } = useFormContext<FuelSurchargeProgramFormValues>();

  if (method === "TablePercent") {
    return (
      <FormGroup cols={3}>
        <FormControl>
          <SelectField
            control={control}
            name="percentBasis"
            label={t("Percentage Applies To")}
            options={fuelSurchargePercentBasisChoices}
            description={t("What the band's percentage is taken from — check the customer's contract before changing.")}
            isReadOnly={disabled}
          />
        </FormControl>
      </FormGroup>
    );
  }

  if (TABLE_METHODS.has(method)) {
    return null;
  }

  return (
    <FormGroup cols={3}>
      <FormControl>
        <NumberField
          control={control}
          name="pegPrice"
          label={t("Peg Price")}
          placeholder="1.20"
          rules={{ required: true }}
          decimalScale={4}
          sideText={t("$/gal")}
          description={t("Fuel price at which the surcharge is zero.")}
          disabled={disabled}
        />
      </FormControl>
      {method === "PerMileStep" ? (
        <>
          <FormControl>
            <NumberField
              control={control}
              name="increment"
              label={t("Increment")}
              placeholder="0.05"
              rules={{ required: true }}
              decimalScale={4}
              sideText={t("$/gal")}
              description={t("Price step above the peg that triggers a rate increase.")}
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="incrementRate"
              label={t("Rate per Increment")}
              placeholder="0.01"
              rules={{ required: true }}
              decimalScale={4}
              sideText={t("$/mi")}
              description={t("Per-mile rate added for each full increment above the peg.")}
              disabled={disabled}
            />
          </FormControl>
        </>
      ) : (
        <FormControl>
          <NumberField
            control={control}
            name="milesPerGallon"
            label={t("Miles per Gallon")}
            placeholder="6.5"
            rules={{ required: true }}
            decimalScale={2}
            sideText="mpg"
            description={t("Fleet MPG divisor: rate = (price − peg) ÷ MPG.")}
            disabled={disabled}
          />
        </FormControl>
      )}
    </FormGroup>
  );
}

function WeekAndRoundingSection({ disabled, method }: { disabled?: boolean; method: string }) {
  const t = useT();

  const { control } = useFormContext<FuelSurchargeProgramFormValues>();
  const isStepMethod = method === "PerMileStep";

  return (
    <Card className="gap-0 p-0">
      <CardHeader className="gap-0 border-b pt-3">
        <CardTitle className="text-sm font-medium">{t("Week Resolution & Rounding")}</CardTitle>
        <p className="text-muted-foreground text-xs">
          {t("Pins exactly which week's price applies and how rates round — the two most common fuel surcharge dispute sources")}
        </p>
      </CardHeader>
      <CardContent className="p-4">
        <FormGroup cols={3}>
          <FormControl>
            <SelectField
              control={control}
              name="dateBasis"
              label={t("Date Basis")}
              options={fuelSurchargeDateBasisChoices}
              description={t("Which shipment date selects the price week.")}
              isReadOnly={disabled}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="priceEffectiveDay"
              label={t("Price Effective Day")}
              options={fuelSurchargeEffectiveDayChoices}
              description={t("Monday's DOE price applies starting this weekday.")}
              isReadOnly={disabled}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="missingPriceFallback"
              label={t("Missing Price Behavior")}
              options={fuelSurchargeFallbackChoices}
              description={t("What happens when the week's price hasn't published yet.")}
              isReadOnly={disabled}
            />
          </FormControl>
          {isStepMethod && (
            <FormControl>
              <SelectField
                control={control}
                name="stepRounding"
                label={t("Step Rounding")}
                options={fuelSurchargeStepRoundingChoices}
                description={t("How partial increments above the peg count.")}
                isReadOnly={disabled}
              />
            </FormControl>
          )}
          <FormControl>
            <SelectField
              control={control}
              name="rateRounding"
              label={t("Rate Rounding")}
              options={fuelSurchargeRateRoundingChoices}
              description={t("Rounding mode for computed rates and final amounts.")}
              isReadOnly={disabled}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="ratePrecision"
              label={t("Rate Precision")}
              placeholder="4"
              decimalScale={0}
              description={t("Decimal places for the computed per-mile rate (0–6).")}
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="minAmount"
              label={t("Minimum Amount")}
              placeholder={t("No floor")}
              decimalScale={2}
              sideText="$"
              description={t("Optional floor for the surcharge per shipment.")}
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="maxAmount"
              label={t("Maximum Amount")}
              placeholder={t("No cap")}
              decimalScale={2}
              sideText="$"
              description={t("Optional cap for the surcharge per shipment.")}
              disabled={disabled}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ApplicabilitySection() {
  const t = useT();

  const { control } = useFormContext<FuelSurchargeProgramFormValues>();

  return (
    <Card className="gap-0">
      <CardHeader className="gap-0 border-b pb-3">
        <CardTitle className="gap-0 text-sm font-medium">{t("Applicability")}</CardTitle>
        <p className="text-muted-foreground text-xs">
          {t("Leave a filter empty to apply to all — the surcharge only generates when the shipment matches every non-empty filter")}
        </p>
      </CardHeader>
      <CardContent className="p-4">
        <FormGroup cols={2}>
          <FormControl>
            <ShipmentTypeMultiSelectField
              control={control}
              name="shipmentTypeIds"
              label={t("Shipment Types")}
              placeholder={t("All shipment types")}
            />
          </FormControl>
          <FormControl>
            <ServiceTypeMultiSelectField
              control={control}
              name="serviceTypeIds"
              label={t("Service Types")}
              placeholder={t("All service types")}
            />
          </FormControl>
          <FormControl>
            <EquipmentTypeMultiSelectField
              control={control}
              name="tractorTypeIds"
              label={t("Tractor Types")}
              placeholder={t("All tractor types")}
            />
          </FormControl>
          <FormControl>
            <EquipmentTypeMultiSelectField
              control={control}
              name="trailerTypeIds"
              label={t("Trailer Types")}
              placeholder={t("All trailer types")}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}
