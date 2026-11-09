import { UsStateAutocompleteField } from "@/components/autocomplete-fields";
import { InfoTooltip } from "@/components/info-tooltip";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useAnyFieldDirty } from "@/hooks/use-dirty-fields";
import { statusChoices } from "@/lib/choices";
import { selectOptionMetaString } from "@/lib/select-option-meta";
import { MAX_FACTOR_TEXT, type FactoringCompany } from "@/types/factoring-company";
import { Lock01Icon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";

import type { FocusEvent, ReactNode } from "react";
import { useFormContext } from "react-hook-form";
import type { FactorState, InvoiceRegion } from "./factoring-invoice-preview";

/** Which section has the person's attention, and so which part of the invoice to light. */
export type InvoiceAttention = (
  kind: "hover" | "focus",
  regions: readonly InvoiceRegion[] | null,
) => void;

const REMIT_FIELDS = [
  "name",
  "addressLine1",
  "addressLine2",
  "city",
  "stateId",
  "postalCode",
  "remittanceInstructions",
] as const;

type FactoringCompanyFormProps = {
  /** The agreement in force sets recourse and keeps it in step, so the switch is read-only. */
  recourseFromAgreement?: boolean;
  onOpenAgreement?: () => void;
  /** Hears the chosen state as the invoice prints it. */
  onState?: (state: FactorState) => void;
  onAttention?: InvoiceAttention;
  /** Customers with a notice (drafted, sent or acknowledged) naming the stored remit-to. */
  remitNotices?: number;
  onOpenCustomers?: () => void;
};

/**
 * A titled block of the form. An info icon beside the title says where its fields print, and while
 * the pointer or focus is inside it, it tells the invoice preview which part of the invoice is its
 * own.
 */
function DetailsCard({
  title,
  description,
  prints,
  regions,
  onAttention,
  children,
}: {
  title: string;
  description: string;
  /** Where the section's fields print on the invoice, or that they never do. */
  prints: string;
  regions: readonly InvoiceRegion[];
  onAttention?: InvoiceAttention;
  children: ReactNode;
}) {
  const leaveFocus = (event: FocusEvent<HTMLElement>) => {
    if (!event.currentTarget.contains(event.relatedTarget as Node | null)) {
      onAttention?.("focus", null);
    }
  };

  return (
    <section
      aria-label={title}
      onMouseEnter={() => onAttention?.("hover", regions)}
      onMouseLeave={() => onAttention?.("hover", null)}
      onFocus={() => onAttention?.("focus", regions)}
      onBlur={leaveFocus}
      className="min-w-0 rounded-lg border bg-card"
    >
      <header className="border-b px-4 py-3">
        <div className="flex min-w-0 flex-col gap-0.5">
          <div className="flex items-center gap-1">
            <h3 className="text-sm font-semibold">{title}</h3>
            <InfoTooltip label={title} className="-translate-y-0.5">
              {prints}
            </InfoTooltip>
          </div>
          <p className="text-xs text-muted-foreground">{description}</p>
        </div>
      </header>
      <div className="flex flex-col gap-3 p-4">{children}</div>
    </section>
  );
}

export function FactoringCompanyForm({
  recourseFromAgreement = false,
  onOpenAgreement,
  onState,
  onAttention,
  remitNotices = 0,
  onOpenCustomers,
}: FactoringCompanyFormProps = {}) {
  const t = useT();
  const { control, getValues } = useFormContext<FactoringCompany>();
  const remitChanged = useAnyFieldDirty(control, REMIT_FIELDS);

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <DetailsCard
        title={t("Identity")}
        description={t("How the factor is named on invoices and in customer billing profiles")}
        prints={t(
          "The name prints as the payee: in Remit To, in How to pay and in the notice of assignment.",
        )}
        regions={["name"]}
        onAttention={onAttention}
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="status"
              label={t("Status")}
              placeholder={t("Status")}
              description={t(
                "Only active factors can be assigned to customers. A factor customers are still assigned to cannot be made inactive.",
              )}
              options={statusChoices}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="code"
              label={t("Code")}
              placeholder={t("e.g., RTSFIN")}
              description={t("Short identifier, unique across your organization.")}
              maxLength={10}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              rules={{ required: true }}
              name="name"
              label={t("Name")}
              placeholder={t("e.g., RTS Financial Service, Inc.")}
              description={t(
                "The factor's legal name. It is printed as the payee on every invoice assigned to it.",
              )}
              maxLength={255}
            />
          </FormControl>
        </FormGroup>
      </DetailsCard>

      <DetailsCard
        title={t("Remit-to address")}
        description={t("Where customers send payment for invoices assigned to this factor")}
        prints={t("The address prints in Remit To and in How to pay.")}
        regions={["address"]}
        onAttention={onAttention}
      >
        {remitChanged && remitNotices > 0 ? (
          <Alert size="sm" variant="warning">
            <AlertDescription className="flex flex-wrap items-center justify-between gap-x-3 gap-y-2">
              <span>
                {t(
                  "{0, plural, one {# customer has} other {# customers have}} a notice naming the current remit-to. Invoices print the new one once you save, so send them an updated notice.",
                  remitNotices,
                )}
              </span>
              {onOpenCustomers ? (
                <Button type="button" size="sm" variant="outline" onClick={onOpenCustomers}>
                  {t("Open customers")}
                </Button>
              ) : null}
            </AlertDescription>
          </Alert>
        ) : null}
        <FormGroup cols={2}>
          <FormControl cols="full">
            <InputField
              control={control}
              rules={{ required: true }}
              name="addressLine1"
              label={t("Address line 1")}
              placeholder={t("e.g., PO Box 840267")}
              maxLength={150}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="addressLine2"
              label={t("Address line 2")}
              placeholder={t("Suite, floor, lockbox, etc.")}
              maxLength={150}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="city"
              label={t("City")}
              placeholder={t("City")}
              maxLength={100}
            />
          </FormControl>
          <FormControl>
            <UsStateAutocompleteField
              control={control}
              name="stateId"
              label={t("State")}
              placeholder={t("State")}
              onOptionChange={(option) =>
                onState?.({
                  abbreviation: option
                    ? selectOptionMetaString(option, "abbreviation") || null
                    : null,
                  countryName: option
                    ? selectOptionMetaString(option, "countryName") || null
                    : null,
                })
              }
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              rules={{ required: true }}
              name="postalCode"
              label={t("Postal code")}
              placeholder={t("e.g., 75284")}
            />
          </FormControl>
        </FormGroup>
      </DetailsCard>

      <DetailsCard
        title={t("Payment instructions and notice")}
        description={t("What every invoice assigned to this factor tells the customer")}
        prints={t(
          "Each line of the instructions prints in Remit To and in How to pay. The notice prints in its own box under the parties.",
        )}
        regions={["instructions", "notice"]}
        onAttention={onAttention}
      >
        <TextareaField
          control={control}
          name="remittanceInstructions"
          label={t("Remittance instructions")}
          placeholder={t("e.g., ACH: routing 111000614, account 8842091")}
          description={t(
            "Bank, lockbox or ACH details. Each line prints under the factor's address wherever an invoice says where to pay.",
          )}
          maxLength={MAX_FACTOR_TEXT}
          rows={3}
        />
        <TextareaField
          control={control}
          name="noticeOfAssignment"
          label={t("Notice of assignment")}
          placeholder={t(
            "Leave blank to print the standard notice: this invoice has been assigned to, and is payable only to, the factor.",
          )}
          description={t(
            "Use the exact wording your factoring agreement requires, if it gives one.",
          )}
          maxLength={MAX_FACTOR_TEXT}
          rows={4}
        />
      </DetailsCard>

      <DetailsCard
        title={t("Contact")}
        description={t("Printed under the remit-to address, and how your team reaches the factor")}
        prints={t("The email and phone print in Remit To, under the address, and in How to pay.")}
        regions={["contact"]}
        onAttention={onAttention}
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="email"
              type="email"
              label={t("Email")}
              placeholder={t("e.g., payments@factor.com")}
              maxLength={255}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="phoneNumber"
              label={t("Phone")}
              placeholder={t("e.g., (800) 555-0142")}
              maxLength={30}
            />
          </FormControl>
        </FormGroup>
      </DetailsCard>

      <DetailsCard
        title={t("Recourse")}
        description={t("Who carries the risk when a customer does not pay")}
        prints={t("Never printed. It decides who carries the risk of an unpaid invoice.")}
        regions={[]}
        onAttention={onAttention}
      >
        {recourseFromAgreement ? (
          <div className="flex items-center gap-3 rounded-md border border-dashed px-3 py-2.5">
            <Lock01Icon className="size-3.5 shrink-0 text-muted-foreground" />
            <span className="flex min-w-0 flex-1 flex-col gap-0.5">
              <span className="text-sm font-medium">
                {getValues("nonRecourse") ? t("Non-recourse") : t("Recourse")}
              </span>
              <span className="text-xs text-muted-foreground">
                {t(
                  "Set by the agreement in force and kept in step with it. Change it on the agreement.",
                )}
              </span>
            </span>
            {onOpenAgreement ? (
              <Button type="button" size="sm" variant="outline" onClick={onOpenAgreement}>
                {t("Open agreement")}
              </Button>
            ) : null}
          </div>
        ) : (
          <SwitchField
            control={control}
            name="nonRecourse"
            label={t("Non-recourse")}
            description={t(
              "The factor buys invoices outright and carries the credit risk. Its invoices leave the collections worklist and stop counting toward the customer's credit limit. Leave off for recourse factoring, where an unpaid invoice comes back to you.",
            )}
          />
        )}
      </DetailsCard>

      <DetailsCard
        title={t("Notes")}
        description={t("For your team only")}
        prints={t("Never printed. Only your team sees these.")}
        regions={[]}
        onAttention={onAttention}
      >
        <TextareaField
          control={control}
          name="notes"
          label={t("Notes")}
          placeholder={t("e.g., 90-day chargeback, 3% fee")}
          maxLength={MAX_FACTOR_TEXT}
          rows={3}
        />
      </DetailsCard>
    </div>
  );
}
