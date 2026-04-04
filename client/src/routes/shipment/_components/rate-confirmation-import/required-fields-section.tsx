import {
  CustomerAutocompleteField,
  FormulaTemplateAutocompleteField,
  ServiceTypeAutocompleteField,
  ShipmentTypeAutocompleteField,
  EquipmentTypeAutocompleteField,
} from "@/components/autocomplete-fields";
import { FormControl, FormGroup } from "@/components/ui/form";
import { equipmentClassSchema } from "@/types/equipment-type";
import { AlertCircleIcon } from "lucide-react";
import type { Control } from "react-hook-form";
import type { RequiredFieldsForm } from "./types";

type RequiredFieldsSectionProps = {
  control: Control<RequiredFieldsForm>;
  hasValues: boolean;
};

export function RequiredFieldsSection({ control, hasValues }: RequiredFieldsSectionProps) {
  return (
    <div className="px-3 py-2">
      {!hasValues && (
        <div className="mb-2 flex items-center gap-2 rounded-md bg-amber-500/[0.06] px-2.5 py-1.5">
          <AlertCircleIcon className="size-3 text-amber-500 shrink-0" />
          <span className="text-2xs text-amber-600 dark:text-amber-400">
            Complete these fields to create the shipment
          </span>
        </div>
      )}
      <FormGroup cols={2}>
        <FormControl>
          <CustomerAutocompleteField
            control={control}
            name="customerId"
            rules={{ required: true }}
            label="Customer"
            placeholder="Select Customer"
          />
        </FormControl>
        <FormControl>
          <ServiceTypeAutocompleteField
            control={control}
            name="serviceTypeId"
            rules={{ required: true }}
            label="Service Type"
            placeholder="Select Service Type"
          />
        </FormControl>
        <FormControl>
          <ShipmentTypeAutocompleteField
            control={control}
            name="shipmentTypeId"
            rules={{ required: true }}
            label="Shipment Type"
            placeholder="Select Shipment Type"
          />
        </FormControl>
        <FormControl>
          <FormulaTemplateAutocompleteField
            control={control}
            name="formulaTemplateId"
            rules={{ required: true }}
            label="Rating Method"
            placeholder="Select Rating Method"
          />
        </FormControl>
        <FormControl>
          <EquipmentTypeAutocompleteField
            control={control}
            name="tractorTypeId"
            label="Tractor Type"
            placeholder="Select Tractor Type"
            extraSearchParams={{
              classes: [equipmentClassSchema.enum.Tractor],
            }}
            clearable
          />
        </FormControl>
        <FormControl>
          <EquipmentTypeAutocompleteField
            control={control}
            name="trailerTypeId"
            label="Trailer Type"
            placeholder="Select Trailer Type"
            extraSearchParams={{
              classes: [equipmentClassSchema.enum.Trailer, equipmentClassSchema.enum.Container],
            }}
            clearable
          />
        </FormControl>
      </FormGroup>
    </div>
  );
}
