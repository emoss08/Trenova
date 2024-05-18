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
