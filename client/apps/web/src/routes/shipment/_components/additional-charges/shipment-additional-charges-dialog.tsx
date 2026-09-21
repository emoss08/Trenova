"use no memo";
import { useT } from "@trenova/shared/i18n/use-t";
import { AccessorialChargeAutocompleteField } from "@/components/autocomplete-fields";
import { ChargeSplitEditor } from "@/components/billing/charge-split-editor";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { accessorialChargeMethodChoices } from "@/lib/choices";
import type { SelectOption as GraphQLSelectOption } from "@/lib/graphql/select-options";
import { chargeLineTotal } from "@trenova/shared/lib/charge-split";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useRef } from "react";
import { type Control, type FieldValues, useFormContext, useWatch } from "react-hook-form";

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
  const t = useT();

  const { control, setValue, getValues, trigger } = useFormContext<Shipment>();
  const method = useWatch({ control, name: `additionalCharges.${index}.method` });
  const amount = useWatch({ control, name: `additionalCharges.${index}.amount` });
  const unit = useWatch({ control, name: `additionalCharges.${index}.unit` });
  const customerId = useWatch({ control, name: "customerId" });
  const billToCustomerId = useWatch({ control, name: "billToCustomerId" });
  const customer = useWatch({ control, name: "customer" });
  const billToCustomer = useWatch({ control, name: "billToCustomer" });
  const payerId = billToCustomerId || customerId;
  const payerLabel = billToCustomerId ? billToCustomer?.name : customer?.name;
  // A percentage accessorial is priced against the freight charge on the
  // server, so its dollar total is unknown here and only a percent split can
  // be checked against it.
  const percentOnly = method === "Percentage";
  const lineTotal = percentOnly ? null : chargeLineTotal({ method, amount, unit });
  const lastAppliedChargeIdRef = useRef<string | null>(
    isEditing ? (getValues(`additionalCharges.${index}.accessorialChargeId`) ?? null) : null,
  );

  function handleChargeSelected(option: GraphQLSelectOption | null) {
    if (option) {
      const opts = { shouldDirty: true, shouldValidate: true };
      const legacy = option as unknown as { id?: string; method?: string; amount?: unknown };
      const metaMethod = (option.meta?.["method"] as string) ?? legacy.method;
      const metaAmount = option.meta?.["amount"] ?? legacy.amount;
      if (lastAppliedChargeIdRef.current !== option.id) {
        if (metaMethod) setValue(`additionalCharges.${index}.method`, metaMethod as never, opts);
        if (metaAmount != null)
          setValue(`additionalCharges.${index}.amount`, metaAmount as never, opts);
        lastAppliedChargeIdRef.current = option.id ?? null;
      }
      setValue(`additionalCharges.${index}.accessorialCharge`, option as never);
    }
  }

  async function handleSave() {
    const isValid = await trigger([
      `additionalCharges.${index}.accessorialChargeId`,
      `additionalCharges.${index}.unit`,
      `additionalCharges.${index}.method`,
      `additionalCharges.${index}.amount`,
      `additionalCharges.${index}.allocations`,
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
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {isEditing ? t("Edit Additional Charge") : t("Add Additional Charge")}
          </DialogTitle>
          <DialogDescription>
            {isEditing
              ? t("Update the accessorial charge details")
              : t("Select an accessorial charge and configure its billing details")}
          </DialogDescription>
        </DialogHeader>
        <FormGroup cols={2}>
          <FormControl className="col-span-2">
            <AccessorialChargeAutocompleteField
              control={control}
              name={`additionalCharges.${index}.accessorialChargeId`}
              label={t("Accessorial Charge")}
              clearable
              rules={{ required: true }}
              placeholder={t("Select Accessorial Charge")}
              description={t(
                "Accessorial charges are additional fees charged for services such as detention, fuel surcharge, and more.",
              )}
              onOptionChange={handleChargeSelected}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name={`additionalCharges.${index}.unit`}
              label={t("Unit")}
              rules={{ required: true, min: 1 }}
              placeholder={t("Unit")}
              description={t(
                "Quantity of units this charge applies to (number of pallets, hours of detention, etc.)",
              )}
              sideText={t("unit(s)")}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name={`additionalCharges.${index}.method`}
              label={t("Method")}
              options={accessorialChargeMethodChoices}
              rules={{ required: true }}
              placeholder={t("Select Method")}
              description={t(
                "Calculation method for this charge (flat rate, per mile, percentage of linehaul, etc.)",
              )}
            />
          </FormControl>
          <FormControl className="col-span-2">
            <NumberField
              control={control}
              name={`additionalCharges.${index}.amount`}
              label={t("Amount")}
              decimalScale={2}
              rules={{ required: true, min: 1 }}
              placeholder={t("Amount")}
              sideText={t("USD")}
              description={t(
                "Dollar value per unit for this accessorial service, used to calculate total charges for billing and settlement",
              )}
            />
          </FormControl>
          <FormControl className="col-span-2">
            <ChargeSplitEditor
              control={control as unknown as Control<FieldValues>}
              name={`additionalCharges.${index}.allocations`}
              chargeAmount={lineTotal}
              percentOnly={percentOnly}
              defaultPayer={payerId ? { id: payerId, label: payerLabel ?? t("customer") } : null}
            />
          </FormControl>
        </FormGroup>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onCancel}>
            {t("Cancel")}
          </Button>
          <Button type="button" onClick={handleSave}>
            {t("Save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
