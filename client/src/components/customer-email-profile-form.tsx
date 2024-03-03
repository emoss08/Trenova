/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { InputField } from "@/components/common/fields/input";
import { CustomerFormValues as FormValues } from "@/types/customer";
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
