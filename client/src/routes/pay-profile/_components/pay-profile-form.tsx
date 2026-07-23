import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Button } from "@/components/ui/button";
import { FormControl, FormGroup } from "@/components/ui/form";
import {
  payCalcMethodChoices,
  payComponentKindChoices,
  payRevenueBasisChoices,
  payeeClassificationChoices,
  statusChoices,
} from "@/lib/choices";
import type { PayProfileFormValues } from "@/types/driver-pay";
import { Plus, Trash2 } from "lucide-react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";

export function PayProfileForm() {
  const { control } = useFormContext<PayProfileFormValues>();
  const componentsArray = useFieldArray({ control, name: "components" });

  return (
    <div className="flex flex-col gap-4">
      <FormGroup cols={2}>
        <FormControl>
          <SelectField
            control={control}
            name="status"
            label="Status"
            options={statusChoices}
            rules={{ required: true }}
            description="Inactive profiles keep their history but cannot be assigned to new drivers."
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="classification"
            label="Classification"
            options={payeeClassificationChoices}
            rules={{ required: true }}
            description="Determines W-2 vs 1099 treatment and GL expense account."
          />
        </FormControl>
        <FormControl className="col-span-2">
          <InputField
            control={control}
            name="name"
            label="Name"
            placeholder="e.g. OTR Company Driver - Standard"
            rules={{ required: true }}
            description="A short, unique name dispatchers and payroll staff will recognize when assigning drivers."
          />
        </FormControl>
        <FormControl className="col-span-2">
          <TextareaField
            control={control}
            name="description"
            label="Description"
            placeholder="When to use this pay package"
            description="Explain who this package is for and any negotiated terms so future admins know when to apply it."
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="guaranteedPeriodMinimum"
            label="Guaranteed Minimum / Period"
            decimalScale={2}
            fixedDecimalScale
            sideText="USD"
            description="Top-up applied when period gross falls below this floor."
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="perDiemDailyCap"
            label="Per Diem Daily Cap"
            decimalScale={2}
            fixedDecimalScale
            sideText="USD"
            description="Maximum non-taxable per diem per day when splitting pay for tax purposes (IRS cap applies)."
          />
        </FormControl>
      </FormGroup>

      <div className="flex items-center justify-between border-t pt-4">
        <div>
          <h3 className="text-sm font-semibold">Pay Components</h3>
          <p className="text-xs text-muted-foreground">
            Each component computes pay per completed move; team splits apply on top.
          </p>
        </div>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={() =>
            componentsArray.append({
              kind: "StopPay",
              method: "PerStop",
              description: "",
              rate: "",
              revenueBasis: null,
              bands: [],
              freeTimeMinutes: 120,
              minAmount: null,
              maxAmount: null,
              isActive: true,
            })
          }
        >
          <Plus className="size-3.5" />
          Add Component
        </Button>
      </div>

      {componentsArray.fields.map((field, index) => (
        <ComponentEditor
          key={field.id}
          index={index}
          onRemove={
            componentsArray.fields.length > 1 ? () => componentsArray.remove(index) : undefined
          }
        />
      ))}
    </div>
  );
}

function ComponentEditor({ index, onRemove }: { index: number; onRemove?: () => void }) {
  const { control } = useFormContext<PayProfileFormValues>();
  const method = useWatch({ control, name: `components.${index}.method` });
  const kind = useWatch({ control, name: `components.${index}.kind` });
  const isPerMile =
    method === "PerLoadedMile" || method === "PerEmptyMile" || method === "PerTotalMile";
  const isPercent = method === "PercentOfRevenue";

  const bandsArray = useFieldArray({ control, name: `components.${index}.bands` });

  return (
    <div className="rounded-lg border p-3">
      <div className="flex items-start justify-between gap-2">
        <FormGroup cols={2} className="grow">
          <FormControl>
            <SelectField
              control={control}
              name={`components.${index}.kind`}
              label="Component"
              options={payComponentKindChoices}
              rules={{ required: true }}
              description="What is being paid — linehaul, stop pay, detention, hazmat premium, and so on."
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name={`components.${index}.method`}
              label="Method"
              options={payCalcMethodChoices}
              rules={{ required: true }}
              description="How the amount is calculated: per mile, percent of revenue, flat, per stop, or per hour."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name={`components.${index}.rate`}
              label={isPercent ? "Percent of Revenue" : "Rate"}
              placeholder={isPercent ? "e.g. 27.5" : "e.g. 0.55"}
              rules={{ required: true }}
              description={
                isPerMile
                  ? "Base rate per mile; overridden by mileage bands below when present."
                  : isPercent
                    ? "Percentage of the shipment's revenue this component pays, e.g. 27.5 pays 27.5%."
                    : "Dollar amount paid per unit of the chosen method (per shipment, per stop, per hour)."
              }
            />
          </FormControl>
          {isPercent && (
            <FormControl>
              <SelectField
                control={control}
                name={`components.${index}.revenueBasis`}
                label="Revenue Basis"
                options={payRevenueBasisChoices}
                rules={{ required: true }}
                description="Which revenue the percentage applies to: linehaul only, linehaul plus fuel surcharge, or total."
              />
            </FormControl>
          )}
          {kind === "Detention" && (
            <FormControl>
              <NumberField
                control={control}
                name={`components.${index}.freeTimeMinutes`}
                label="Free Time (minutes)"
                description="Detention pays only for dwell beyond this threshold."
              />
            </FormControl>
          )}
          {(kind === "Custom" || kind === "Bonus") && (
            <FormControl className={isPercent ? undefined : "col-span-1"}>
              <InputField
                control={control}
                name={`components.${index}.description`}
                label="Label"
                placeholder="Shown on the settlement statement"
                description="The exact wording drivers see for this line on their settlement statement."
              />
            </FormControl>
          )}
          <FormControl>
            <NumberField
              control={control}
              name={`components.${index}.minAmount`}
              label="Minimum per Move"
              decimalScale={2}
              fixedDecimalScale
              sideText="USD"
              description="Floor for this component on any single move; short runs are topped up to this amount."
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name={`components.${index}.maxAmount`}
              label="Maximum per Move"
              decimalScale={2}
              fixedDecimalScale
              sideText="USD"
              description="Cap for this component on any single move; anything above is not paid."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name={`components.${index}.isActive`}
              label="Active"
              outlined
              description="Inactive components are kept for history but skipped when computing pay."
            />
          </FormControl>
        </FormGroup>
        {onRemove && (
          <Button
            type="button"
            size="icon"
            variant="ghost"
            className="shrink-0 text-muted-foreground"
            onClick={onRemove}
            aria-label="Remove component"
          >
            <Trash2 className="size-4" />
          </Button>
        )}
      </div>

      {isPerMile && (
        <div className="mt-3 border-t pt-3">
          <div className="mb-2 flex items-center justify-between">
            <p className="text-xs font-medium">
              Mileage Bands
              <span className="ml-1 font-normal text-muted-foreground">
                (optional; sliding scale by length of haul)
              </span>
            </p>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              onClick={() =>
                bandsArray.append({
                  minMiles: bandsArray.fields.length > 0 ? 0 : 0,
                  maxMiles: 0,
                  rate: "",
                })
              }
            >
              <Plus className="size-3" />
              Add Band
            </Button>
          </div>
          {bandsArray.fields.length > 0 && (
            <div className="flex flex-col gap-2">
              {bandsArray.fields.map((band, bandIndex) => (
                <div key={band.id} className="grid grid-cols-[1fr_1fr_1fr_auto] items-end gap-2">
                  <NumberField
                    control={control}
                    name={`components.${index}.bands.${bandIndex}.minMiles`}
                    label={bandIndex === 0 ? "From (miles)" : undefined}
                    placeholder="0"
                    description={
                      bandIndex === 0 ? "Moves at or above this mileage use this band." : undefined
                    }
                  />
                  <NumberField
                    control={control}
                    name={`components.${index}.bands.${bandIndex}.maxMiles`}
                    label={bandIndex === 0 ? "To (0 = open-ended)" : undefined}
                    placeholder="0"
                    description={
                      bandIndex === 0
                        ? "Moves below this mileage use this band; 0 means no upper limit."
                        : undefined
                    }
                  />
                  <InputField
                    control={control}
                    name={`components.${index}.bands.${bandIndex}.rate`}
                    label={bandIndex === 0 ? "Rate / mile" : undefined}
                    placeholder="0.55"
                    description={
                      bandIndex === 0
                        ? "Per-mile rate paid when the move falls in this band."
                        : undefined
                    }
                  />
                  <Button
                    type="button"
                    size="icon"
                    variant="ghost"
                    className="text-muted-foreground"
                    onClick={() => bandsArray.remove(bandIndex)}
                    aria-label="Remove band"
                  >
                    <Trash2 className="size-3.5" />
                  </Button>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
