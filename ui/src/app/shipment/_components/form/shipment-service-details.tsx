import { AutocompleteField } from "@/components/fields/autocomplete";
import { ColorOptionValue } from "@/components/fields/select-components";
import { FormControl, FormGroup } from "@/components/ui/form";
import { EquipmentTypeSchema } from "@/lib/schemas/equipment-type-schema";
import { ServiceTypeSchema } from "@/lib/schemas/service-type-schema";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { ShipmentTypeSchema } from "@/lib/schemas/shipment-type-schema";
import { useFormContext } from "react-hook-form";

export default function ShipmentServiceDetails() {
  const { control } = useFormContext<ShipmentSchema>();

  return (
    <div className="flex flex-col gap-2">
      <h3 className="text-sm font-medium">Service Information</h3>
      <FormGroup cols={2}>
        <FormControl>
          <AutocompleteField<ShipmentTypeSchema, ShipmentSchema>
            name="shipmentTypeId"
            control={control}
            link="/shipment-types/"
            label="Shipment Type"
            rules={{ required: true }}
            placeholder="Select Shipment Type"
            description="Select the shipment type for the shipment."
            getOptionValue={(option) => option.id || ""}
            getDisplayValue={(option) => (
              <ColorOptionValue color={option.color} value={option.code} />
            )}
            renderOption={(option) => (
              <div className="flex flex-col gap-0.5 items-start size-full">
                <ColorOptionValue color={option.color} value={option.code} />
                {option?.description && (
                  <span className="text-2xs text-muted-foreground truncate w-full">
                    {option?.description}
                  </span>
                )}
              </div>
            )}
          />
        </FormControl>
        <FormControl>
          <AutocompleteField<ServiceTypeSchema, ShipmentSchema>
            name="serviceTypeId"
            control={control}
            link="/service-types/"
            label="Service Type"
            rules={{ required: true }}
            placeholder="Select Service Type"
            description="Select the service type for the shipment."
            getOptionValue={(option) => option.id || ""}
            getDisplayValue={(option) => (
              <ColorOptionValue color={option.color} value={option.code} />
            )}
            renderOption={(option) => (
              <div className="flex flex-col gap-0.5 items-start size-full">
                <ColorOptionValue color={option.color} value={option.code} />
                {option?.description && (
                  <span className="text-2xs text-muted-foreground truncate w-full">
                    {option?.description}
                  </span>
                )}
              </div>
            )}
          />
        </FormControl>
        <FormControl>
          <AutocompleteField<EquipmentTypeSchema, ShipmentSchema>
            name="tractorTypeId"
            control={control}
            label="Tractor Type"
            link="/equipment-types/"
            placeholder="Select Tractor Type"
            description="Select the type of tractor used, considering any special requirements (e.g., refrigeration)."
            getOptionValue={(option) => option.id || ""}
            getDisplayValue={(option) => (
              <ColorOptionValue color={option.color} value={option.code} />
            )}
            renderOption={(option) => (
              <div className="flex flex-col gap-0.5 items-start size-full">
                <ColorOptionValue color={option.color} value={option.code} />
                {option?.description && (
                  <span className="text-2xs text-muted-foreground truncate w-full">
                    {option?.description}
                  </span>
                )}
              </div>
            )}
          />
        </FormControl>
        <FormControl>
          <AutocompleteField<EquipmentTypeSchema, ShipmentSchema>
            name="trailerTypeId"
            control={control}
            label="Trailer Type"
            link="/equipment-types/"
            placeholder="Select Trailer Type"
            description="Select the type of trailer used, considering any special requirements (e.g., refrigeration)."
            getOptionValue={(option) => option.id || ""}
            getDisplayValue={(option) => (
              <ColorOptionValue color={option.color} value={option.code} />
            )}
            renderOption={(option) => (
              <div className="flex flex-col gap-0.5 items-start size-full">
                <ColorOptionValue color={option.color} value={option.code} />
                {option?.description && (
                  <span className="text-2xs text-muted-foreground truncate w-full">
                    {option?.description}
                  </span>
                )}
              </div>
            )}
          />
        </FormControl>
      </FormGroup>
    </div>
  );
}
