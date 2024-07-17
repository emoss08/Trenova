/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
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
