import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { TextareaField } from "@/components/common/fields/textarea";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { statusChoices } from "@/lib/choices";
import { equipManufacturerSchema } from "@/lib/validations/EquipmentSchema";
import { type EquipmentManufacturerFormValues as FormValues } from "@/types/equipment";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { Control, useForm } from "react-hook-form";
import {
  Credenza,
  CredenzaBody,
  CredenzaClose,
  CredenzaContent,
  CredenzaDescription,
  CredenzaFooter,
  CredenzaHeader,
  CredenzaTitle,
} from "./ui/credenza";
import { Form, FormControl, FormGroup } from "./ui/form";

export function EquipManuForm({ control }: { control: Control<FormValues> }) {
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
            description="Status of the Equip. Manufacturer"
            isClearable={false}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label="Name"
            maxLength={10}
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Name"
            autoComplete="name"
            description="Name for the Equip. Manufacturer"
          />
        </FormControl>
        <FormControl className="col-span-full">
          <TextareaField
            name="description"
            rules={{ required: true }}
            control={control}
            label="Description"
            placeholder="Description"
            description="Description of the Equip. Manufacturer"
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

export function EquipManuDialog({ onOpenChange, open }: TableSheetProps) {
  const { control, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(equipManufacturerSchema),
    defaultValues: {
      status: "A",
      name: "",
      description: "",
    },
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "POST",
    path: "/equipment-manufacturers/",
    successMessage: "Equip. Manufacturer created successfully.",
    queryKeysToInvalidate: ["equipment-manufacturer-table-data"],
    closeModal: true,
    errorMessage: "Failed to create new equip. manufacturer.",
  });

  const onSubmit = (values: FormValues) => {
    mutation.mutate(values);
  };

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>Create New Equip. Manufacturer</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Please fill out the form below to create a new Equip. Manufacturer.
        </CredenzaDescription>
        <CredenzaBody>
          <form onSubmit={handleSubmit(onSubmit)}>
            <EquipManuForm control={control} />
            <CredenzaFooter>
              <CredenzaClose asChild>
                <Button variant="outline" type="button">
                  Cancel
                </Button>
              </CredenzaClose>
              <Button type="submit" isLoading={mutation.isPending}>
                Save Changes
              </Button>
            </CredenzaFooter>
          </form>
        </CredenzaBody>
      </CredenzaContent>
    </Credenza>
  );
}
