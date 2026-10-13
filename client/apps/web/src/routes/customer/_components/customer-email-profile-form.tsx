import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import type { Customer } from "@trenova/shared/types/customer";
import { useFormContext } from "react-hook-form";

export function CustomerEmailProfileForm() {
  const t = useT();

  const { control } = useFormContext<Customer>();

  return (
    <div className="flex flex-col gap-6">
      <FormSection
        title={t("Email delivery")}
        description={t(
          "Who receives this customer's invoices. Invoices are always sent from the sender on your organization's Billing email profile.",
        )}
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="emailProfile.subject"
              label={t("Subject line")}
              placeholder={t("e.g., Invoice #{number} from {company}")}
              description={t(
                "Replaces the subject from your invoice email template. Supports {number}, {customer}, {company}, {invoiceTotal}, {dueDate}, {shipmentPro} and {shipmentBol}.",
              )}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="emailProfile.toRecipients"
              label={t("To recipients")}
              placeholder={t("e.g., ap@customer.com, billing@customer.com")}
              description={t(
                "Required when invoices are emailed to this customer. Separate multiple addresses with commas.",
              )}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="emailProfile.ccRecipients"
              label={t("CC recipients")}
              placeholder={t("e.g., controller@customer.com")}
              description={t(
                "Carbon copy recipients who receive a copy of every invoice email. Useful for the customer's management or your internal billing team.",
              )}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="emailProfile.bccRecipients"
              label={t("BCC recipients")}
              placeholder={t("e.g., billing-archive@yourcompany.com")}
              description={t(
                "Blind carbon copy recipients. Other recipients will not see these addresses — useful for internal archiving or compliance.",
              )}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Attachments & content")}
        description={t("Control the invoice attachment format and email body content")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="emailProfile.attachmentName"
              label={t("Attachment filename")}
              placeholder={t("e.g., Invoice-{number}-{customer}.pdf")}
              description={t(
                "Filename for the PDF invoice attachment. Supports {number}, {customer} and {company}.",
              )}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="emailProfile.comment"
              label={t("Email body")}
              placeholder={t("e.g., Please find invoice {number} attached for {customer}.")}
              description={t(
                "Leave empty to use your invoice email template. Text here replaces the template's message for this customer. Supports {number}, {customer}, {company}, {invoiceTotal} and {dueDate}.",
              )}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection title={t("Delivery options")} description={t("Email content preferences")}>
        <FormGroup cols={1}>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="emailProfile.includeShipmentDetail"
              label={t("Include shipment details")}
              description={t(
                "Append a detailed breakdown of each shipment (origin, destination, dates, charges) in the email body below the invoice summary.",
              )}
              position="left"
            />
          </FormControl>
          <FormControl className="min-h-[3em]">
            <SwitchField
              control={control}
              name="emailProfile.readReceipt"
              label={t("Request read receipt")}
              description={t(
                "Ask the recipient's email client for a read confirmation and track when the email is opened. Many mail servers ignore read receipt requests, so opens are the more reliable signal.",
              )}
              position="left"
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
