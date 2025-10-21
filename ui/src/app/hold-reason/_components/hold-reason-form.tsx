import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { holdSeverityChoices, holdTypeChoices } from "@/lib/choices";
import { HoldReasonSchema } from "@/lib/schemas/hold-reason-schema";
import { useFormContext } from "react-hook-form";

function HoldReasonFormOuter({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-col gap-4">{children}</div>;
}

export function HoldReasonForm() {
  const { control } = useFormContext<HoldReasonSchema>();

  return (
    <HoldReasonFormOuter>
      <FormGroup cols={2}>
        <FormControl cols="full">
          <SwitchField
            control={control}
            name="active"
            label="Active"
            description="Toggles whether this hold reason is available for use in the system."
            outlined
            position="left"
          />
        </FormControl>
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
          <InputField
            control={control}
            name="code"
            label="Reason Code"
            placeholder="ELD_OOS"
            rules={{ required: true }}
            maxLength={64}
            description="Stable identifier used by rules, APIs, and search; prefer UPPER_SNAKE_CASE."
          />
        </FormControl>

        <FormControl cols="full">
          <InputField
            control={control}
            name="label"
            label="Display Name"
            placeholder="ELD Out of Service"
            rules={{ required: true }}
            maxLength={100}
            description="Human-friendly name shown in boards, forms, and customer portals."
          />
        </FormControl>

        <FormControl cols="full">
          <TextareaField
            control={control}
            name="description"
            label="Details"
            placeholder="Briefly explain when to use this reason"
            description="Short context explaining when to apply this reason; use customer-safe wording."
          />
        </FormControl>
        <FormControl cols="full">
          <SelectField
            control={control}
            name="defaultSeverity"
            label="Default Severity"
            placeholder="Select Severity"
            description="Starting impact level applied when users select this reason; adjustable per hold."
            options={holdSeverityChoices}
            rules={{ required: true }}
          />
        </FormControl>
      </FormGroup>
      <FormSection
        title="Gating Rules"
        description="Select which actions this reason blocks by default; multiple can apply and stack."
      >
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="defaultBlocksDispatch"
              label="Block Dispatch"
              description="Prevents assigning or dispatching power/trailer until this hold is cleared."
              position="left"
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="defaultBlocksDelivery"
              label="Block Delivery"
              description="Prevents marking stops delivered or closing freight until this hold clears."
              position="left"
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="defaultBlocksBilling"
              label="Block Billing"
              description="Prevents invoicing or moving to billable states while the hold is active."
              position="left"
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="defaultVisibleToCustomer"
              label="Visible to Customer"
              description="Makes this reason visible to customers in the portal."
              position="left"
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </HoldReasonFormOuter>
  );
}
