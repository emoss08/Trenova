import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { HoldReasonAutocompleteField } from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { holdSeverityChoices, holdTypeChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { HoldShipmentRequestSchema } from "@/lib/schemas/shipment-hold-schema";
import { cn } from "@/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { useEffect } from "react";
import { useFormContext, useWatch } from "react-hook-form";

export function ShipmentHoldForm() {
  const { control, setValue } = useFormContext<HoldShipmentRequestSchema>();
  const holdReasonId = useWatch({ control, name: "holdReasonId" });

  const { data: holdReason, isLoading } = useQuery({
    ...queries.holdReason.getById(holdReasonId),
    enabled: !!holdReasonId,
  });

  useEffect(() => {
    if (!isLoading && holdReason) {
      setValue("type", holdReason.type);
      setValue("severity", holdReason.defaultSeverity);
      setValue("blocksDispatch", holdReason.defaultBlocksDispatch);
      setValue("blocksDelivery", holdReason.defaultBlocksDelivery);
      setValue("blocksBilling", holdReason.defaultBlocksBilling);
      setValue("visibleToCustomer", holdReason.defaultVisibleToCustomer);
      setValue("notes", holdReason.description);
    }
  }, [holdReason, setValue, isLoading, holdReasonId]);

  return (
    <>
      <FormGroup>
        <FormControl>
          <HoldReasonAutocompleteField
            control={control}
            name="holdReasonId"
            label="Hold Reason"
            rules={{ required: true }}
            description="The type of hold to apply to the shipment."
            placeholder="Select hold type"
          />
        </FormControl>
      </FormGroup>
      <FormSection
        title="Hold Overrides"
        description="Override the default behavior of the hold reason."
        className={cn("border-t pt-4", !holdReasonId && "hidden")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="type"
              label="Hold Type"
              placeholder="Select Type"
              rules={{ required: true }}
              description="Choose the hold category to drive default behavior, gating, and reporting."
              options={holdTypeChoices}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="severity"
              label="Severity"
              placeholder="Select Severity"
              description="Starting impact level applied when users select this reason; adjustable per hold."
              options={holdSeverityChoices}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="blocksDispatch"
              label="Block Dispatch"
              description="Prevents assigning or dispatching power/trailer until this hold is cleared."
              position="left"
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="blocksDelivery"
              label="Block Delivery"
              description="Prevents marking stops delivered or closing freight until this hold clears."
              position="left"
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="blocksBilling"
              label="Block Billing"
              description="Prevents invoicing or moving to billable states while the hold is active."
              position="left"
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="visibleToCustomer"
              label="Visible to Customer"
              description="Makes this reason visible to customers in the portal."
              position="left"
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="notes"
              label="Notes"
              placeholder="Briefly explain when to use this reason"
              description="Short context explaining when to apply this reason; use customer-safe wording."
              rows={3}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </>
  );
}
