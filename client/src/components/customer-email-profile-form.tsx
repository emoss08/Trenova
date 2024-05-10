import { InputField } from "@/components/common/fields/input";
import { type CustomerFormValues as FormValues } from "@/types/customer";
import { useFormContext } from "react-hook-form";
import { CheckboxInput } from "./common/fields/checkbox";

export function CustomerEmailProfileForm() {
  const { control } = useFormContext<FormValues>();

  return (
    <>
      <div className="my-4 grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
        <div className="flex w-full max-w-sm flex-col justify-between gap-0.5">
          <div className="min-h-[4em]">
            <InputField
              control={control}
              name="emailProfile.subject"
              label="Subject"
              autoCapitalize="none"
              autoCorrect="off"
              type="text"
              placeholder="Subject"
              description="Enter the subject line for the email."
              maxLength={10}
            />
          </div>
        </div>
        <div className="flex w-full max-w-sm flex-col justify-between gap-0.5">
          <div className="min-h-[4em]">
            <InputField
              control={control}
              name="emailProfile.comment"
              label="Comment"
              autoCapitalize="none"
              autoCorrect="off"
              type="text"
              placeholder="Comment"
              description="Provide any additional comments regarding the email or the recipient."
            />
          </div>
        </div>
        <div className="flex w-full max-w-sm flex-col justify-between gap-0.5">
          <div className="min-h-[4em]">
            <InputField
              control={control}
              name="emailProfile.fromAddress"
              label="From Address"
              autoCapitalize="none"
              autoCorrect="off"
              type="text"
              placeholder="From Address"
              description="Specify the sender's email address."
            />
          </div>
        </div>
        <div className="flex w-full max-w-sm flex-col justify-between gap-0.5">
          <div className="min-h-[4em]">
            <InputField
              control={control}
              name="emailProfile.blindCopy"
              label="Blind Copy"
              autoCapitalize="none"
              autoCorrect="off"
              type="text"
              placeholder="Blind Copy"
              description="Enter an email address to receive a blind copy (Bcc) of the email."
            />
          </div>
        </div>
        <div className="flex w-full max-w-sm flex-col justify-between gap-0.5">
          <div className="min-h-[4em]">
            <InputField
              control={control}
              name="emailProfile.readReceiptTo"
              label="Read Receipt To"
              autoCapitalize="none"
              autoCorrect="off"
              type="text"
              placeholder="Read Receipt To"
              description="Designate an email address to receive a notification when the email is opened by the recipient."
            />
          </div>
        </div>
        <div className="flex w-full max-w-sm flex-col justify-between gap-0.5">
          <div className="min-h-[4em]">
            <InputField
              control={control}
              name="emailProfile.attachmentName"
              label="Attachment Name"
              autoCapitalize="none"
              autoCorrect="off"
              type="text"
              placeholder="Attachment Name"
              description="Define the name for any attachment included with the email."
            />
          </div>
        </div>
        <div className="flex w-full max-w-sm flex-col justify-between gap-0.5">
          <div className="min-h-[4em]">
            <CheckboxInput
              control={control}
              label="Read Receipt?"
              disabled
              name="emailProfile.readReceipt"
              description="Toggle this option to request a read receipt."
            />
          </div>
        </div>
      </div>
    </>
  );
}
