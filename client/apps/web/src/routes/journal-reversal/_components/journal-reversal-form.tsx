import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { useFormContext } from "react-hook-form";

/**
 * The fields of a reversal request. Given the entry it reverses (`entryNumber`),
 * as when it is asked for from that entry's card, the entry is fixed and only
 * the date and the reason are asked for.
 */
export function JournalReversalForm({ entryNumber }: { entryNumber?: string } = {}) {
  const t = useT();

  const { control } = useFormContext();
  const fixed = entryNumber !== undefined;

  return (
    <div className="flex flex-col gap-6">
      <FormSection
        title={t("Reversal target")}
        description={
          fixed
            ? t(
                "A mirror entry posts on this date and takes back everything {0} booked",
                entryNumber,
              )
            : t("Specify the journal entry to reverse and the desired posting date")
        }
      >
        <FormGroup cols={fixed ? 1 : 2}>
          {!fixed && (
            <FormControl>
              <InputField
                control={control}
                name="originalJournalEntryId"
                label={t("Original journal entry ID")}
                rules={{ required: true }}
                placeholder={t("Enter the journal entry ID to reverse")}
                description={t("The ID of the posted journal entry you want to reverse.")}
              />
            </FormControl>
          )}
          <FormControl>
            <InputField
              control={control}
              name="requestedAccountingDate"
              label={t("Requested accounting date")}
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
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="reasonCode"
              label={t("Reason code")}
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
