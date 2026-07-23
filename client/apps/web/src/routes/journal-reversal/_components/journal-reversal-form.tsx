import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { useFormContext } from "react-hook-form";

export function JournalReversalForm() {
  const { control } = useFormContext();

  return (
    <div className="flex flex-col gap-6">
      <FormSection
        title="Reversal Target"
        description="Specify the journal entry to reverse and the desired posting date"
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="originalJournalEntryId"
              label="Original Journal Entry ID"
              rules={{ required: true }}
              placeholder="Enter the journal entry ID to reverse"
              description="The ID of the posted journal entry you want to reverse."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="requestedAccountingDate"
              label="Requested Accounting Date"
              rules={{ required: true }}
              type="date"
              description="The date the reversal should be posted to the general ledger."
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Reason"
        description="Explain why this journal entry needs to be reversed"
        className="border-t border-border pt-4"
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="reasonCode"
              label="Reason Code"
              rules={{ required: true }}
              placeholder="e.g., ERROR, DUPLICATE, ADJUSTMENT"
              description="A short classification code for the reversal reason."
            />
          </FormControl>
        </FormGroup>
        <FormControl>
          <TextareaField
            control={control}
            name="reasonText"
            label="Reason"
            rules={{ required: true }}
            placeholder="Provide a detailed reason for the reversal request"
            description="A detailed explanation that will be recorded in the audit trail."
          />
        </FormControl>
      </FormSection>
    </div>
  );
}
