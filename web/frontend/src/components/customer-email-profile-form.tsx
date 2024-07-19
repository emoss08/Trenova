/**
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
import { useEmailProfiles } from "@/hooks/useQueries";
import {
  EmailFormatChoices,
  type CustomerFormValues as FormValues,
} from "@/types/customer";
import { useFormContext } from "react-hook-form";
import { SelectInput } from "./common/fields/select-input";
import { Form, FormControl, FormGroup } from "./ui/form";

export function CustomerEmailProfileForm() {
  const { control } = useFormContext<FormValues>();
  const { selectEmailProfile, isLoading, isError } = useEmailProfiles();

  return (
    <Form>
      <FormGroup>
        <FormControl>
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
        </FormControl>
        <FormControl>
          <SelectInput
            control={control}
            name="emailProfile.emailProfileId"
            isLoading={isLoading}
            isFetchError={isError}
            options={selectEmailProfile}
            label="Email Profile"
            placeholder="Select Email Profile"
            description="Choose an email profile to use for sending the emails to the customer."
            menuPlacement="bottom"
            menuPosition="fixed"
            hasPopoutWindow
            popoutLink="/admin/email-profiles/"
            popoutLinkLabel="Email Profile"
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="emailProfile.emailRecipients"
            label="Email Recipients"
            placeholder="Email Recipients"
            description="Comma seperated list of the email addresses to include in the (To) field."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="emailProfile.emailCcRecipients"
            label="Email Cc Recipients"
            placeholder="Blind Copy"
            description="Comma seperated list of the email addresses to include in the copy (CC) field."
          />
        </FormControl>
        <FormControl>
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
        </FormControl>
        <FormControl>
          <SelectInput
            control={control}
            rules={{ required: true }}
            name="emailProfile.emailFormat"
            options={EmailFormatChoices}
            label="Email Format"
            placeholder="Select Email Format"
            description="Choose the format for which the email will be sent."
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}
