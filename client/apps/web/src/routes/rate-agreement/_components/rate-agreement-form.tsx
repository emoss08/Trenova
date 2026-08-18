import {
  CarrierAutocompleteField,
  CustomerAutocompleteField,
} from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import {
  rateAgreementStatusChoices,
  rateAgreementTypeChoices,
  ratePartyTypeChoices,
  rateRoundingModeChoices,
} from "@/lib/choices";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import type { RateAgreement } from "@trenova/shared/types/rate";
import { useFormContext, useWatch } from "react-hook-form";

/**
 * The negotiated header terms — who the contract is with, when it is in force,
 * and the defaults every lane inherits.
 *
 * Status is shown but not editable: an agreement only moves between states
 * through the review actions, so that an editor cannot activate a contract by
 * resending it.
 */
export function RateAgreementForm() {
  const { control } = useFormContext<RateAgreement>();
  const partyType = useWatch({ control, name: "partyType" });

  return (
    <div className="flex flex-col gap-4">
      <FormGroup cols={2}>
        <FormControl>
          <SelectField
            control={control}
            rules={{ required: true }}
            name="partyType"
            label="Side"
            placeholder="Side"
            description="Whether this contract sets what a customer pays or what a carrier is paid"
            options={ratePartyTypeChoices}
          />
        </FormControl>

        {partyType === "Carrier" ? (
          <FormControl>
            <CarrierAutocompleteField
              control={control}
              rules={{ required: true }}
              name="carrierId"
              label="Carrier"
              placeholder="Carrier"
              description="The carrier this agreement pays"
            />
          </FormControl>
        ) : (
          <FormControl>
            <CustomerAutocompleteField
              control={control}
              rules={{ required: true }}
              name="customerId"
              label="Customer"
              placeholder="Customer"
              description="The customer this agreement bills"
            />
          </FormControl>
        )}

        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="code"
            label="Code"
            placeholder="ACME-2026"
            description="How this contract is referred to internally"
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label="Name"
            placeholder="Acme Freight Agreement"
            description="The contract's name as somebody reading an invoice would recognise it"
          />
        </FormControl>

        <FormControl cols="full">
          <TextareaField
            control={control}
            name="description"
            label="Description"
            placeholder="What this agreement covers"
            description="Anything the terms do not say that the next person will need"
          />
        </FormControl>

        <FormControl>
          <SelectField
            control={control}
            rules={{ required: true }}
            name="agreementType"
            label="Type"
            placeholder="Type"
            description="Contract, published tariff, spot deal, project or dedicated"
            options={rateAgreementTypeChoices}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="status"
            label="Status"
            placeholder="Status"
            description="Moved by the review actions, never by a save"
            options={rateAgreementStatusChoices}
            isReadOnly
          />
        </FormControl>

        <FormControl>
          <InputField
            control={control}
            name="contractRef"
            label="Contract reference"
            placeholder="MSA-2026-114"
            description="The signed document's own number, for when somebody asks"
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="priority"
            label="Priority"
            placeholder="0"
            description="Breaks a tie when two agreements cover the same lane equally narrowly"
          />
        </FormControl>
      </FormGroup>

      <FormGroup cols={2}>
        <FormControl>
          <AutoCompleteDateField
            control={control}
            rules={{ required: true }}
            name="effectiveFrom"
            label="In force from"
            placeholder="Start date"
            description="Shipments moving on or after this date price against this contract"
          />
        </FormControl>
        <FormControl>
          <AutoCompleteDateField
            control={control}
            name="effectiveTo"
            label="Until"
            placeholder="End date"
            description="Leave empty to run until somebody ends it"
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name="autoRenew"
            label="Renews automatically"
            description="Whether the term rolls forward unless somebody gives notice"
            outlined
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="renewalNoticeDays"
            label="Notice days"
            placeholder="30"
            description="How much warning the contract requires before it ends"
          />
        </FormControl>
      </FormGroup>

      <FormGroup cols={3}>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="currency"
            label="Currency"
            placeholder="USD"
            description="What this contract is written in, converted at the rating date when it differs from yours"
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="defaultMinCharge"
            label="Default minimum"
            placeholder=""
            description="The floor a lane inherits when it does not set its own"
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="defaultMaxCharge"
            label="Default maximum"
            placeholder=""
            description="A ceiling a lane inherits when it does not set its own"
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="roundingMode"
            label="Rounding"
            placeholder="Rounding"
            description="How the final amount is rounded"
            options={rateRoundingModeChoices}
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="roundingPrecision"
            label="Precision"
            placeholder="2"
            description="How many decimal places the rounded amount keeps"
          />
        </FormControl>
      </FormGroup>

      {partyType === "Carrier" && (
        <FormGroup cols={2}>
          <FormControl>
            <NumberField
              control={control}
              name="marginFloorPercent"
              label="Margin floor (%)"
              placeholder="12"
              description="The thinnest margin this carrier may be paid down to"
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="maxPayPercentOfSell"
              label="Maximum pay (% of sell)"
              placeholder="85"
              description="A ceiling on the pay as a share of what the customer is billed"
            />
          </FormControl>
        </FormGroup>
      )}

      {partyType === "Customer" && (
        <FormGroup cols={2}>
          <FormControl>
            <CustomerAutocompleteField
              control={control}
              name="billToCustomerId"
              label="Bill to"
              placeholder="Bill-to customer"
              description="Redirects invoicing, for a customer whose parent settles the freight"
            />
          </FormControl>
        </FormGroup>
      )}
    </div>
  );
}
