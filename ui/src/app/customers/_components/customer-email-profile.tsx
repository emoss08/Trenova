/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { InputField } from "@/components/fields/input-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Separator } from "@/components/ui/separator";
import { type CustomerSchema } from "@/lib/schemas/customer-schema";
import { useFormContext } from "react-hook-form";

export default function CustomerEmailProfile() {
  const { control } = useFormContext<CustomerSchema>();

  return (
    <div className="size-full">
      <div className="flex select-none flex-col px-4">
        <h2 className="mt-2 text-2xl font-semibold">Email Profile</h2>
        <p className="text-xs text-muted-foreground">
          Configure email settings for the customer.
        </p>
      </div>
      <Separator className="mt-2" />
      <div className="p-4">
        <FormGroup cols={1}>
          <FormControl>
            <InputField
              control={control}
              name="emailProfile.subject"
              placeholder="Subject"
              label="Subject"
              description="The subject line of the email that will be sent to the customer upon billing."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="emailProfile.comment"
              placeholder="Comment"
              label="Comment"
              description="The comment that will be sent to the customer upon billing."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="emailProfile.fromEmail"
              placeholder="From Email"
              label="From Email"
              description="The email address that will be used to send the email."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="emailProfile.blindCopy"
              placeholder="Blind Copy"
              label="Blind Copy"
              description="A comma separated list of email addresses that will receive a blind copy of the email."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="emailProfile.attachmentName"
              placeholder="Attachment Name"
              label="Attachment Name"
              description="The name of the attachment that will be sent to the customer upon billing."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="emailProfile.readReceipt"
              label="Read Receipt"
              position="left"
              description="Whether to request a read receipt for the email."
            />
          </FormControl>
        </FormGroup>
      </div>
    </div>
  );
}
