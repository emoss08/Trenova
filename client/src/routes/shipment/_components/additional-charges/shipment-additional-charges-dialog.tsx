"use no memo";
import { AccessorialChargeAutocompleteField } from "@/components/autocomplete-fields";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { FormControl, FormGroup } from "@/components/ui/form";
import { accessorialChargeMethodChoices } from "@/lib/choices";
import type { AccessorialCharge } from "@/types/accessorial-charge";
import type { Shipment } from "@/types/shipment";
import { useRef } from "react";
import { useFormContext } from "react-hook-form";

export function AdditionalChargeDialog({
  open,
  onCancel,
  onSave,
  index,
  isEditing,
  update,
}: {
  open: boolean;
  onCancel: () => void;
  onSave: () => void;
  index: number;
  isEditing: boolean;
  update: (index: number, value: any) => void;
}) {
  const { control, setValue, getValues, trigger } = useFormContext<Shipment>();
  const lastAppliedChargeIdRef = useRef<string | null>(
    isEditing ? (getValues(`additionalCharges.${index}.accessorialChargeId`) ?? null) : null,
  );

  function handleChargeSelected(option: AccessorialCharge | null) {
    if (option) {
      const opts = { shouldDirty: true, shouldValidate: true };
      if (lastAppliedChargeIdRef.current !== option.id) {
        setValue(`additionalCharges.${index}.method`, option.method, opts);
        setValue(`additionalCharges.${index}.amount`, option.amount, opts);
        lastAppliedChargeIdRef.current = option.id ?? null;
      }
      setValue(`additionalCharges.${index}.accessorialCharge`, option);
    }
  }

  async function handleSave() {
    const isValid = await trigger([
      `additionalCharges.${index}.accessorialChargeId`,
      `additionalCharges.${index}.unit`,
      `additionalCharges.${index}.method`,
      `additionalCharges.${index}.amount`,
    ]);

    if (!isValid) {
      return;
    }

    const values = getValues(`additionalCharges.${index}`);
    update(index, values);
    onSave();
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(isOpen) => {
        if (!isOpen) onCancel();
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>
            {isEditing ? "Edit Additional Charge" : "Add Additional Charge"}
          </DialogTitle>
          <DialogDescription>
            {isEditing
              ? "Update the accessorial charge details"
              : "Select an accessorial charge and configure its billing details"}
          </DialogDescription>
        </DialogHeader>
        <FormGroup cols={2}>
          <FormControl className="col-span-2">
            <AccessorialChargeAutocompleteField
              control={control}
              name={`additionalCharges.${index}.accessorialChargeId`}
              label="Accessorial Charge"
              clearable
              rules={{ required: true }}
              placeholder="Select Accessorial Charge"
              description="Accessorial charges are additional fees charged for services such as detention, fuel surcharge, and more."
              onOptionChange={handleChargeSelected}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name={`additionalCharges.${index}.unit`}
              label="Unit"
              rules={{ required: true, min: 1 }}
              placeholder="Unit"
              description="Quantity of units this charge applies to (number of pallets, hours of detention, etc.)"
              sideText="unit(s)"
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name={`additionalCharges.${index}.method`}
              label="Method"
              options={accessorialChargeMethodChoices}
              rules={{ required: true }}
              placeholder="Select Method"
              description="Calculation method for this charge (flat rate, per mile, percentage of linehaul, etc.)"
            />
          </FormControl>
          <FormControl className="col-span-2">
            <NumberField
              control={control}
              name={`additionalCharges.${index}.amount`}
              label="Amount"
              decimalScale={2}
              rules={{ required: true, min: 1 }}
              placeholder="Amount"
              sideText="USD"
              description="Dollar value per unit for this accessorial service, used to calculate total charges for billing and settlement"
            />
          </FormControl>
        </FormGroup>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onCancel}>
            Cancel
          </Button>
          <Button type="button" onClick={handleSave}>
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
