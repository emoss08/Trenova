import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { holdSeverityChoices, holdTypeChoices } from "@/lib/choices";
import type { HoldReason } from "@/types/hold-reason";
import { useFormContext } from "react-hook-form";

export function HoldReasonForm({ disabled }: { disabled?: boolean }) {
  const t = useT();

  const { control } = useFormContext<HoldReason>();

  return (
    <div className="flex flex-col gap-4">
      <FormGroup cols={2}>
        <FormControl cols="full">
          <SwitchField
            control={control}
            name="active"
            label={t("Active")}
            description={t("Toggles whether this hold reason is available for use in the system.")}
            outlined
            position="left"
            disabled={disabled}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="type"
            label={t("Hold Type")}
            placeholder={t("Select Type")}
            rules={{ required: true }}
            description={t(
              "Choose the hold category to drive default behavior, gating, and reporting.",
            )}
            options={holdTypeChoices}
            isReadOnly={disabled}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="code"
            label={t("Reason Code")}
            placeholder="ELD_OOS"
            rules={{ required: true }}
            maxLength={64}
            description={t(
              "Stable identifier used by rules, APIs, and search; prefer UPPER_SNAKE_CASE.",
            )}
            disabled={disabled}
          />
        </FormControl>
        <FormControl cols="full">
          <InputField
            control={control}
            name="label"
            label={t("Display Name")}
            placeholder={t("ELD Out of Service")}
            rules={{ required: true }}
            maxLength={100}
            description={t("Human-friendly name shown in boards, forms, and customer portals.")}
            disabled={disabled}
          />
        </FormControl>
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="description"
            label={t("Details")}
            placeholder={t("Briefly explain when to use this reason")}
            description={t(
              "Short context explaining when to apply this reason; use customer-safe wording.",
            )}
            disabled={disabled}
          />
        </FormControl>
        <FormControl cols="full">
          <SelectField
            control={control}
            name="defaultSeverity"
            label={t("Default Severity")}
            placeholder={t("Select Severity")}
            description={t(
              "Starting impact level applied when users select this reason; adjustable per hold.",
            )}
            options={holdSeverityChoices}
            rules={{ required: true }}
            isReadOnly={disabled}
          />
        </FormControl>
      </FormGroup>
      <FormSection
        title={t("Gating Rules")}
        description={t(
          "Select which actions this reason blocks by default; multiple can apply and stack.",
        )}
      >
        <FormGroup cols={2}>
          <FormControl>
            <SwitchField
              control={control}
              name="defaultBlocksDispatch"
              label={t("Block Dispatch")}
              description={t(
                "Prevents assigning or dispatching power/trailer until this hold is cleared.",
              )}
              position="left"
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="defaultBlocksDelivery"
              label={t("Block Delivery")}
              description={t(
                "Prevents marking stops delivered or closing freight until this hold clears.",
              )}
              position="left"
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="defaultBlocksBilling"
              label={t("Block Billing")}
              description={t(
                "Prevents invoicing or moving to billable states while the hold is active.",
              )}
              position="left"
              disabled={disabled}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="defaultVisibleToCustomer"
              label={t("Visible to Customer")}
              description={t("Makes this reason visible to customers in the portal.")}
              position="left"
              disabled={disabled}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
