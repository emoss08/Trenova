import { AutocompleteField } from "@/components/fields/autocomplete";
import { InputField } from "@/components/fields/input-field";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { FormControl, FormGroup } from "@/components/ui/form";
import { CommoditySchema } from "@/lib/schemas/commodity-schema";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { TableSheetProps } from "@/types/data-table";
import { ShipmentCommodity } from "@/types/shipment";
import { useCallback } from "react";
import {
  UseFieldArrayRemove,
  UseFieldArrayUpdate,
  useFormContext,
} from "react-hook-form";

interface CommodityDialogProps extends TableSheetProps {
  index: number;
  isEditing: boolean;
  update: UseFieldArrayUpdate<ShipmentSchema, "commodities">;
  remove: UseFieldArrayRemove;
  initialData?: ShipmentCommodity;
}

export function CommodityDialog({
  open,
  onOpenChange,
  isEditing,
  update,
  index,
  remove,
}: CommodityDialogProps) {
  const { getValues, reset } = useFormContext<ShipmentSchema>();

  const handleSave = () => {
    const formValues = getValues();
    const commodity = formValues.commodities?.[index];

    // Only proceed if we have valid commodity data
    if (commodity?.commodityId && commodity?.commodity) {
      const updatedCommodity = {
        commodityId: commodity.commodityId,
        commodity: commodity.commodity,
        pieces: commodity.pieces || 1,
        weight: commodity.weight || 0,
        // Preserve the existing ID if editing, otherwise it will be handled by the backend
        id: isEditing ? commodity.id : undefined,
        shipmentId: formValues?.id || "",
      };

      // Use the update function for both new and existing commodities
      update?.(index, updatedCommodity);
    }

    onOpenChange(false);
  };

  const handleClose = useCallback(() => {
    onOpenChange(false);

    if (!isEditing) {
      remove(index);
    } else {
      const originalValues = getValues();
      const commodities = originalValues?.commodities || [];

      reset(
        {
          commodities: [
            ...commodities.slice(0, index),
            commodities[index],
            ...commodities.slice(index + 1),
          ],
        },
        {
          keepValues: true,
        },
      );
    }
  }, [onOpenChange, remove, index, isEditing, reset, getValues]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{isEditing ? "Edit" : "Add"} Commodity</DialogTitle>
          <DialogDescription>
            {isEditing
              ? "Edit the existing commodity"
              : "Add a new commodity to the existing shipment."}
          </DialogDescription>
        </DialogHeader>
        <DialogBody>
          <CommodityForm index={index} />
        </DialogBody>
        <DialogFooter>
          <Button variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button onClick={handleSave}>Save</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function CommodityForm({ index }: { index: number }) {
  const { control, setValue } = useFormContext<ShipmentSchema>();

  return (
    <FormGroup>
      <FormControl>
        <AutocompleteField<CommoditySchema, ShipmentSchema>
          name={`commodities.${index}.commodityId`}
          control={control}
          link="/commodities/"
          label="Commodity"
          clearable
          rules={{ required: true }}
          placeholder="Select Commodity"
          description="Select the commodity to include in the shipment."
          getOptionValue={(option) => option.id || ""}
          getDisplayValue={(option) => option.name}
          renderOption={(option) => option.name}
          onOptionChange={(option) => {
            if (option) {
              setValue(`commodities.${index}.commodityId`, option.id || "");
              setValue(`commodities.${index}.commodity`, option);
            }
          }}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name={`commodities.${index}.pieces`}
          label="Pieces"
          type="number"
          rules={{ required: true, min: 1 }}
          placeholder="Pieces"
          description="Specify the number of pieces for this commodity."
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name={`commodities.${index}.weight`}
          placeholder="Weight"
          label="Weight"
          type="number"
          rules={{ required: true, min: 1 }}
          description="Enter the weight of a single piece of this commodity."
        />
      </FormControl>
    </FormGroup>
  );
}
