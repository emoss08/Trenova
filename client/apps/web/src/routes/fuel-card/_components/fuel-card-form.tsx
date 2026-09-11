import { useT } from "@trenova/shared/i18n/use-t";
import {
  TractorAutocompleteField,
  WorkerAutocompleteField,
} from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { InfoPopover } from "@/components/info-popover";
import { fuelCardProviderChoices, fuelCardStatusChoices } from "@/lib/choices";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import type { FuelCardFormValues } from "@trenova/shared/types/fuel-card";
import { BanIcon } from "lucide-react";
import { useFormContext } from "react-hook-form";

const EDITABLE_STATUS_OPTIONS = fuelCardStatusChoices.filter(
  (choice) => choice.value !== "Cancelled",
);

type FuelCardFormProps = {
  isEdit: boolean;
  cancelled?: boolean;
  cancelReason?: string | null;
};

export function FuelCardForm({ isEdit, cancelled = false, cancelReason }: FuelCardFormProps) {
  const t = useT();

  const { control } = useFormContext<FuelCardFormValues>();
  const locked = isEdit && cancelled;

  return (
    <div className="flex flex-col gap-6">
      {locked ? (
        <Alert>
          <BanIcon className="size-4" />
          <AlertTitle>{t("This card is cancelled")}</AlertTitle>
          <AlertDescription>
            {t("Cancelling is permanent, so nothing here can be changed. {0}", cancelReason ? ` Reason given: ${cancelReason}` : "")}
          </AlertDescription>
        </Alert>
      ) : null}

      <FormSection
        title={t("Card")}
        description={t("Enough to recognise the card on a statement. The full card number is never stored.")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="provider"
              label={t("Provider")}
              options={fuelCardProviderChoices}
              rules={{ required: true }}
              placeholder={t("Select a provider")}
              isReadOnly={locked}
              description={t("Who issued the card. Statement imports use this to pick the column layout.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="lastFour"
              label={
                <span className="inline-flex items-center gap-1">
                  {t("Last four digits")}
                  <InfoPopover title={t("Why only the last four")}>
                    {t("Statements identify a card by its last four digits, and that is all a purchase needs to be matched back to it. Storing the full number would make this table a payment-card record with the handling rules that come with one.")}
                  </InfoPopover>
                </span>
              }
              placeholder="4821"
              maxLength={4}
              inputMode="numeric"
              rules={{ required: true }}
              readOnly={locked}
              description={t("The four digits printed at the end of the card number.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="label"
              label={t("Label")}
              placeholder={t("e.g. Unit 118 card")}
              maxLength={100}
              rules={{ required: true }}
              readOnly={locked}
              description={t("How the card is referred to in pickers and on purchases.")}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="status"
              label={t("Status")}
              options={EDITABLE_STATUS_OPTIONS}
              rules={{ required: true }}
              placeholder={t("Select a status")}
              isReadOnly={locked}
              description={t("Suspend a card to stop it matching new purchases without losing its history. Cancelling is a separate, permanent action.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="externalCardId"
              label={t("Provider card ID")}
              placeholder={t("e.g. CMD-00918273")}
              maxLength={100}
              readOnly={locked}
              description={t("The provider's own masked token for the card, if the statement carries one.")}
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="expiresAt"
              label={t("Expires")}
              placeholder={t("Expiry date")}
              clearable
              readOnly={locked}
              description={t("Printed on the card. Leave empty when the provider does not set one.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection
        title={t("Assignment")}
        description={t("Who carries the card and which unit it fuels. Imported rows that name only the card fall back to this tractor.")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <WorkerAutocompleteField<FuelCardFormValues>
              control={control}
              name="assignedWorkerId"
              label={t("Driver")}
              placeholder={t("Select a driver")}
              clearable
              disabled={locked}
              description={t("The driver who normally carries this card.")}
            />
          </FormControl>
          <FormControl>
            <TractorAutocompleteField<FuelCardFormValues>
              control={control}
              name="assignedTractorId"
              label={t("Tractor")}
              placeholder={t("Select a tractor")}
              clearable
              disabled={locked}
              description={t("The unit purchases on this card are booked to when the statement names no tractor.")}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="notes"
              label={t("Notes")}
              placeholder={t("e.g. Replacement for the card lost in March")}
              maxLength={2000}
              readOnly={locked}
              description={t("Anything the next person maintaining this card should know.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
