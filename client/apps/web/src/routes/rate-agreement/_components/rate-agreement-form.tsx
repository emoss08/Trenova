import { useT } from "@trenova/shared/i18n/use-t";
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
  currencyChoices,
  rateAgreementStatusChoices,
  rateAgreementTypeChoices,
  ratePartyTypeChoices,
  rateRoundingModeChoices,
} from "@/lib/choices";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
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
  const t = useT();

  const { control } = useFormContext<RateAgreement>();
  const partyType = useWatch({ control, name: "partyType" });

  return (
    <div className="space-y-6">
      <FormSection
        title={t("General Information")}
        description={t("Who the contract is with and how it is identified across the system.")}
        className="border-b pb-4"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="partyType"
              label={t("Party Type")}
              placeholder={t("Select party type")}
              description={t("Whether this contract sets what a customer pays or what a carrier is paid")}
              options={ratePartyTypeChoices}
            />
          </FormControl>

          {partyType === "Carrier" ? (
            <FormControl>
              <CarrierAutocompleteField
                control={control}
                rules={{ required: true }}
                name="carrierId"
                label={t("Carrier")}
                placeholder={t("Select carrier")}
                description={t("The carrier this agreement pays")}
              />
            </FormControl>
          ) : (
            <FormControl>
              <CustomerAutocompleteField
                control={control}
                rules={{ required: true }}
                name="customerId"
                label={t("Customer")}
                placeholder={t("Select customer")}
                description={t("The customer this agreement bills")}
              />
            </FormControl>
          )}

          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="code"
              label={t("Code")}
              placeholder={t("e.g., ACME-2026")}
              description={t("Short identifier used to reference this contract internally. Must be unique across your organization.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="name"
              label={t("Name")}
              placeholder={t("e.g., Acme Freight Agreement")}
              description={t("The contract's name as somebody reading an invoice would recognize it")}
            />
          </FormControl>

          <FormControl cols="full">
            <TextareaField
              control={control}
              name="description"
              label={t("Description")}
              placeholder={t("What this agreement covers")}
              description={t("Anything the terms do not say that the next person will need")}
            />
          </FormControl>

          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="agreementType"
              label={t("Agreement Type")}
              placeholder={t("Select type")}
              description={t("Contract, published tariff, spot deal, project, or dedicated capacity")}
              options={rateAgreementTypeChoices}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="status"
              label={t("Status")}
              placeholder={t("Status")}
              description={t("Moved by the review actions (submit, approve, suspend), never by a save")}
              options={rateAgreementStatusChoices}
              isReadOnly
            />
          </FormControl>

          <FormControl>
            <InputField
              control={control}
              name="contractRef"
              label={t("Contract Reference")}
              placeholder={t("e.g., MSA-2026-114")}
              description={t("The signed document's own number, for when somebody asks")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="priority"
              label={t("Priority")}
              placeholder="0"
              description={t("Breaks a tie when two agreements cover the same lane equally narrowly. Higher wins.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Term")}
        description={t("When the contract is in force and how it renews.")}
        className="border-b pb-4"
      >
        <FormGroup cols={2}>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              rules={{ required: true }}
              name="effectiveFrom"
              label={t("Effective From")}
              placeholder={t("Select start date")}
              description={t("Shipments moving on or after this date price against this contract")}
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="effectiveTo"
              label={t("Effective To")}
              placeholder={t("Select end date")}
              description={t("Leave empty to run until somebody ends it")}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="autoRenew"
              label={t("Auto Renew")}
              description={t("The term rolls forward automatically unless somebody gives notice")}
              outlined
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="renewalNoticeDays"
              label={t("Renewal Notice")}
              placeholder="30"
              sideText="days"
              description={t("How much warning the contract requires before it ends")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Pricing Defaults")}
        description={t("The currency, guardrails, and rounding every lane inherits unless it sets its own.")}
        className={partyType ? "border-b pb-4" : undefined}
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="currency"
              label={t("Currency")}
              placeholder={t("Select currency")}
              description={t("What this contract is written in, converted at the rating date when it differs from yours")}
              options={currencyChoices}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="roundingMode"
              label={t("Rounding Mode")}
              placeholder={t("Select rounding")}
              description={t("How the final amount is rounded")}
              options={rateRoundingModeChoices}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="defaultMinCharge"
              label={t("Default Minimum Charge")}
              placeholder="0.00"
              sideText="$"
              decimalScale={2}
              thousandSeparator
              description={t("The floor a lane inherits when it does not set its own")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="defaultMaxCharge"
              label={t("Default Maximum Charge")}
              placeholder="0.00"
              sideText="$"
              decimalScale={2}
              thousandSeparator
              description={t("A ceiling a lane inherits when it does not set its own")}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="roundingPrecision"
              label={t("Rounding Precision")}
              placeholder="2"
              description={t("How many decimal places the rounded amount keeps")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      {partyType === "Carrier" && (
        <FormSection
          title={t("Margin Guardrails")}
          description={t("Limits on what this carrier may be paid relative to what the load sells for.")}
        >
          <FormGroup cols={2}>
            <FormControl>
              <NumberField
                control={control}
                name="marginFloorPercent"
                label={t("Margin Floor")}
                placeholder="12"
                sideText="%"
                decimalScale={2}
                description={t("The thinnest margin this carrier may be paid down to")}
              />
            </FormControl>
            <FormControl>
              <NumberField
                control={control}
                name="maxPayPercentOfSell"
                label={t("Maximum Pay")}
                placeholder="85"
                sideText="%"
                decimalScale={2}
                description={t("A ceiling on the pay as a share of what the customer is billed")}
              />
            </FormControl>
          </FormGroup>
        </FormSection>
      )}

      {partyType === "Customer" && (
        <FormSection
          title={t("Invoicing")}
          description={t("Where invoices for this contract's freight are sent.")}
        >
          <FormGroup cols={2}>
            <FormControl>
              <CustomerAutocompleteField
                control={control}
                name="billToCustomerId"
                label={t("Bill To")}
                placeholder={t("Select bill-to customer")}
                description={t("Redirects invoicing, for a customer whose parent settles the freight")}
              />
            </FormControl>
          </FormGroup>
        </FormSection>
      )}
    </div>
  );
}
