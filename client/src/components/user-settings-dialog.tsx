import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { TextareaField } from "@/components/common/fields/textarea";
import { statusChoices } from "@/lib/choices";
import { User } from "@/types/accounts";
import { type ServiceTypeFormValues as FormValues } from "@/types/shipment";
import { Control } from "react-hook-form";
import { Credenza, CredenzaBody, CredenzaContent } from "./ui/credenza";
import { Form, FormControl, FormGroup } from "./ui/form";
import UserProfilePage from "./user-settings/profile-page";

export function ServiceTypeForm({ control }: { control: Control<FormValues> }) {
  return (
    <Form>
      <FormGroup className="grid gap-6 md:grid-cols-1 lg:grid-cols-2 xl:grid-cols-2">
        <FormControl>
          <SelectInput
            name="status"
            rules={{ required: true }}
            control={control}
            label="Status"
            options={statusChoices}
            placeholder="Select Status"
            description="Status of the Service Type"
            isClearable={false}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="code"
            label="Code"
            maxLength={10}
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Code"
            autoComplete="code"
            description="Code for the Service Type"
          />
        </FormControl>
        <FormControl className="col-span-full">
          <TextareaField
            name="description"
            control={control}
            label="Description"
            placeholder="Description"
            description="Description of the Service Type"
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

type UserSettingsDialogProps = {
  onOpenChange: () => void;
  open: boolean;
  user: User;
};

export function UserSettingsDialog({
  onOpenChange,
  open,
  user,
}: UserSettingsDialogProps) {
  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent className="max-w-[1000px]">
        <CredenzaBody>
          <UserProfilePage user={user} />
        </CredenzaBody>
      </CredenzaContent>
    </Credenza>
  );
}
