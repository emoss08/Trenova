import { InputField } from "@/components/fields/input-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Separator } from "@trenova/shared/components/ui/separator";
import type { Customer } from "@trenova/shared/types/customer";
import { MailIcon, PaperclipIcon, SettingsIcon } from "lucide-react";
import { useFormContext } from "react-hook-form";

function SectionHeader({
  icon: Icon,
  title,
  description,
}: {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  description: string;
}) {
  return (
    <div className="flex items-center gap-3">
      <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
        <Icon className="size-4" />
      </div>
      <div>
        <h3 className="text-sm leading-none font-semibold tracking-tight">
          {title}
        </h3>
        <p className="mt-1 text-xs text-muted-foreground">{description}</p>
      </div>
    </div>
  );
}

export function CustomerEmailProfileForm() {
  const { control } = useFormContext<Customer>();

  return (
    <div className="space-y-6">
      <SectionHeader
        icon={MailIcon}
        title="Email Delivery"
        description="Configure how invoices are emailed to this customer's accounts payable team"
      />
      <FormGroup cols={2}>
        <FormControl>
          <InputField
            control={control}
            name="emailProfile.subject"
            label="Subject Line"
            placeholder="e.g., Invoice #{number} from {company}"
            description="Email subject used when sending invoices. Supports {number}, {customer}, and {company}."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="emailProfile.fromEmail"
            label="From Address"
            placeholder="e.g., billing@yourcompany.com"
            description="The sender address that appears on invoice emails. Must be a verified email domain in your organization's email settings."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="emailProfile.toRecipients"
            label="To Recipients"
            placeholder="e.g., ap@customer.com, billing@customer.com"
            description="Primary recipient addresses for invoice delivery. Separate multiple addresses with commas."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="emailProfile.ccRecipients"
            label="CC Recipients"
            placeholder="e.g., controller@customer.com"
            description="Carbon copy recipients who receive a copy of every invoice email. Useful for the customer's management or your internal billing team."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="emailProfile.bccRecipients"
            label="BCC Recipients"
            placeholder="e.g., billing-archive@yourcompany.com"
            description="Blind carbon copy recipients. Other recipients will not see these addresses — useful for internal archiving or compliance."
          />
        </FormControl>
      </FormGroup>

      <Separator />

      <SectionHeader
        icon={PaperclipIcon}
        title="Attachments & Content"
        description="Control the invoice attachment format and email body content"
      />
      <FormGroup cols={2}>
        <FormControl>
          <InputField
            control={control}
            name="emailProfile.attachmentName"
            label="Attachment Filename"
            placeholder="e.g., Invoice-{number}-{customer}.pdf"
            description="Filename for the PDF invoice attachment. Supports {number}, {customer}, and {company}."
          />
        </FormControl>
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="emailProfile.comment"
            label="Email Body"
            placeholder="e.g., Please find invoice {number} attached for {customer}."
            description="Default message included in the email body above the invoice details. Supports {number}, {customer}, and {company}."
          />
        </FormControl>
      </FormGroup>

      <Separator />

      <SectionHeader
        icon={SettingsIcon}
        title="Delivery Options"
        description="Email content preferences"
      />
      <FormGroup cols={1}>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="emailProfile.includeShipmentDetail"
            label="Include Shipment Details"
            description="Append a detailed breakdown of each shipment (origin, destination, dates, charges) in the email body below the invoice summary."
            position="left"
          />
        </FormControl>
        <FormControl className="min-h-[3em]">
          <SwitchField
            control={control}
            name="emailProfile.readReceipt"
            label="Request Read Receipt"
            description="Ask the recipient's email client to send a delivery/read confirmation. Note: many email clients and corporate mail servers silently ignore read receipt requests."
            position="left"
          />
        </FormControl>
      </FormGroup>
    </div>
  );
}
