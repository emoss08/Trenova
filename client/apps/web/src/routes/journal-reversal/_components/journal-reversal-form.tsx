import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { useFormContext } from "react-hook-form";

export function JournalReversalForm() {
  const t = useT();

  const { control } = useFormContext();

  return (
    <div className="flex flex-col gap-6">
      <FormSection
        title={t("Reversal Target")}
        description={t("Specify the journal entry to reverse and the desired posting date")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="originalJournalEntryId"
              label={t("Original Journal Entry ID")}
              rules={{ required: true }}
              placeholder={t("Enter the journal entry ID to reverse")}
              description={t("The ID of the posted journal entry you want to reverse.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="requestedAccountingDate"
              label={t("Requested Accounting Date")}
              rules={{ required: true }}
              type="date"
              description={t("The date the reversal should be posted to the general ledger.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Reason")}
        description={t("Explain why this journal entry needs to be reversed")}
        className="border-border border-t pt-4"
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="reasonCode"
              label={t("Reason Code")}
              rules={{ required: true }}
              placeholder={t("e.g., ERROR, DUPLICATE, ADJUSTMENT")}
              description={t("A short classification code for the reversal reason.")}
            />
          </FormControl>
        </FormGroup>
        <FormControl>
          <TextareaField
            control={control}
            name="reasonText"
            label={t("Reason")}
            rules={{ required: true }}
            placeholder={t("Provide a detailed reason for the reversal request")}
            description={t("A detailed explanation that will be recorded in the audit trail.")}
          />
        </FormControl>
      </FormSection>
    </div>
  );
}
