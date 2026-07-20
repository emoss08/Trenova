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
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { FormControl, FormGroup } from "@/components/ui/form";
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
            label="Name"
            placeholder="Acme National DOE Program"
            rules={{ required: true }}
            maxLength={100}
            description="Shown on customer profiles and the fuel dashboard."
            disabled={disabled}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="code"
            label="Code"
            placeholder="FSC-DOE-STD"
            rules={{ required: true }}
            maxLength={50}
            description="Short unique identifier for this program."
            disabled={disabled}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="status"
            label="Status"
            options={fuelSurchargeProgramStatusChoices}
            description="Inactive programs stop applying surcharges immediately."
            isReadOnly={disabled}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="method"
            label="Method"
            rules={{ required: true }}
            options={fuelSurchargeMethodChoices}
            description="Formula methods compute rates from parameters; table methods use explicit price bands."
            isReadOnly={disabled}
          />
        </FormControl>
        <FormControl>
          <FuelIndexAutocompleteField
            control={control}
            name="fuelIndexId"
            label="Fuel Index"
            placeholder="Select Fuel Index"
            rules={{ required: true }}
            description="The weekly price series this program keys off (DOE region or custom index)."
          />
        </FormControl>
        <FormControl>
          <AccessorialChargeAutocompleteField
            control={control}
            name="accessorialChargeId"
            label="Accessorial Charge"
            placeholder="Select Accessorial Charge"
            rules={{ required: true }}
            description="The catalog charge the generated fuel surcharge line posts against."
          />
        </FormControl>
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="description"
            label="Description"
            placeholder="Contract terms, customer references, or maintenance notes"
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
  const { control } = useFormContext<FuelSurchargeProgramFormValues>();

  if (method === "TablePercent") {
    return (
      <FormGroup cols={3}>
        <FormControl>
          <SelectField
            control={control}
            name="percentBasis"
            label="Percentage Applies To"
            options={fuelSurchargePercentBasisChoices}
            description="What the band's percentage is taken from — check the customer's contract before changing."
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
          label="Peg Price"
          placeholder="1.20"
          rules={{ required: true }}
          decimalScale={4}
          sideText="$/gal"
          description="Fuel price at which the surcharge is zero."
          disabled={disabled}
        />
      </FormControl>
      {method === "PerMileStep" ? (
        <>
          <FormControl>
            <NumberField
              control={control}
              name="increment"
              label="Increment"
              placeholder="0.05"
              rules={{ required: true }}
              decimalScale={4}
              sideText="$/gal"
              description="Price step above the peg that triggers a rate increase."
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="incrementRate"
              label="Rate per Increment"
              placeholder="0.01"
              rules={{ required: true }}
              decimalScale={4}
              sideText="$/mi"
              description="Per-mile rate added for each full increment above the peg."
              disabled={disabled}
            />
          </FormControl>
        </>
      ) : (
        <FormControl>
          <NumberField
            control={control}
            name="milesPerGallon"
            label="Miles per Gallon"
            placeholder="6.5"
            rules={{ required: true }}
            decimalScale={2}
            sideText="mpg"
            description="Fleet MPG divisor: rate = (price − peg) ÷ MPG."
            disabled={disabled}
          />
        </FormControl>
      )}
    </FormGroup>
  );
}

function WeekAndRoundingSection({ disabled, method }: { disabled?: boolean; method: string }) {
  const { control } = useFormContext<FuelSurchargeProgramFormValues>();
  const isStepMethod = method === "PerMileStep";

  return (
    <Card className="gap-0 p-0">
      <CardHeader className="border-b pt-3 gap-0">
        <CardTitle className="text-sm font-medium">Week Resolution & Rounding</CardTitle>
        <p className="text-xs text-muted-foreground">
          Pins exactly which week&apos;s price applies and how rates round — the two most common
          fuel surcharge dispute sources
        </p>
      </CardHeader>
      <CardContent className="p-4">
        <FormGroup cols={3}>
          <FormControl>
            <SelectField
              control={control}
              name="dateBasis"
              label="Date Basis"
              options={fuelSurchargeDateBasisChoices}
              description="Which shipment date selects the price week."
              isReadOnly={disabled}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="priceEffectiveDay"
              label="Price Effective Day"
              options={fuelSurchargeEffectiveDayChoices}
              description="Monday's DOE price applies starting this weekday."
              isReadOnly={disabled}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="missingPriceFallback"
              label="Missing Price Behavior"
              options={fuelSurchargeFallbackChoices}
              description="What happens when the week's price hasn't published yet."
              isReadOnly={disabled}
            />
          </FormControl>
          {isStepMethod && (
            <FormControl>
              <SelectField
                control={control}
                name="stepRounding"
                label="Step Rounding"
                options={fuelSurchargeStepRoundingChoices}
                description="How partial increments above the peg count."
                isReadOnly={disabled}
              />
            </FormControl>
          )}
          <FormControl>
            <SelectField
              control={control}
              name="rateRounding"
              label="Rate Rounding"
              options={fuelSurchargeRateRoundingChoices}
              description="Rounding mode for computed rates and final amounts."
              isReadOnly={disabled}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="ratePrecision"
              label="Rate Precision"
              placeholder="4"
              decimalScale={0}
              description="Decimal places for the computed per-mile rate (0–6)."
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="minAmount"
              label="Minimum Amount"
              placeholder="No floor"
              decimalScale={2}
              sideText="$"
              description="Optional floor for the surcharge per shipment."
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="maxAmount"
              label="Maximum Amount"
              placeholder="No cap"
              decimalScale={2}
              sideText="$"
              description="Optional cap for the surcharge per shipment."
              disabled={disabled}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ApplicabilitySection() {
  const { control } = useFormContext<FuelSurchargeProgramFormValues>();

  return (
    <Card className="gap-0">
      <CardHeader className="border-b pb-3 gap-0">
        <CardTitle className="text-sm font-medium gap-0">Applicability</CardTitle>
        <p className="text-xs text-muted-foreground">
          Leave a filter empty to apply to all — the surcharge only generates when the shipment
          matches every non-empty filter
        </p>
      </CardHeader>
      <CardContent className="p-4">
        <FormGroup cols={2}>
          <FormControl>
            <ShipmentTypeMultiSelectField
              control={control}
              name="shipmentTypeIds"
              label="Shipment Types"
              placeholder="All shipment types"
            />
          </FormControl>
          <FormControl>
            <ServiceTypeMultiSelectField
              control={control}
              name="serviceTypeIds"
              label="Service Types"
              placeholder="All service types"
            />
          </FormControl>
          <FormControl>
            <EquipmentTypeMultiSelectField
              control={control}
              name="tractorTypeIds"
              label="Tractor Types"
              placeholder="All tractor types"
            />
          </FormControl>
          <FormControl>
            <EquipmentTypeMultiSelectField
              control={control}
              name="trailerTypeIds"
              label="Trailer Types"
              placeholder="All trailer types"
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}
