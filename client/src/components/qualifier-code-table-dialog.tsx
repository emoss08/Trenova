import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { TextareaField } from "@/components/common/fields/textarea";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { statusChoices } from "@/lib/choices";
import { qualifierCodeSchema } from "@/lib/validations/StopSchema";
import { type QualifierCodeFormValues as FormValues } from "@/types/stop";
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

export function QualifierCodeForm({
  control,
}: {
  control: Control<FormValues>;
}) {
  return (
    <Form className="flex-1 overflow-y-visible">
      <FormGroup className="grid gap-6 md:grid-cols-1 lg:grid-cols-2">
        <FormControl>
          <SelectInput
            name="status"
            rules={{ required: true }}
            control={control}
            label="Status"
            options={statusChoices}
            placeholder="Select Status"
            description="Status of the Qualifier Code"
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
            description="Code for the Qualifier Code"
          />
        </FormControl>
        <FormControl className="col-span-full">
          <TextareaField
            name="description"
            rules={{ required: true }}
            control={control}
            label="Description"
            placeholder="Description"
            description="Description of the Qualifier Code"
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

export function QualifierCodeDialog({ onOpenChange, open }: TableSheetProps) {
  const { control, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(qualifierCodeSchema),
    defaultValues: {
      status: "A",
      code: "",
      description: "",
    },
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "POST",
    path: "/qualifier-codes/",
    successMessage: "Qualifier Code created successfully.",
    queryKeysToInvalidate: "qualifierCodes",
    closeModal: true,
    errorMessage: "Failed to create new qualifier code.",
  });

  const onSubmit = (values: FormValues) => mutation.mutate(values);

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>Create New Qualifier Code</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Please fill out the form below to create a new Qualifier Code.
        </CredenzaDescription>
        <CredenzaBody>
          <form onSubmit={handleSubmit(onSubmit)}>
            <QualifierCodeForm control={control} />
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
