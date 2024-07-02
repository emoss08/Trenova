import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { TextareaField } from "@/components/common/fields/textarea";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { severityChoices, statusChoices } from "@/lib/choices";
import { commentTypeSchema } from "@/lib/validations/DispatchSchema";
import { type CommentTypeFormValues as FormValues } from "@/types/dispatch";
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

export function CommentTypeForm({ control }: { control: Control<FormValues> }) {
  return (
    <Form>
      <FormGroup className="grid gap-6 md:grid-cols-1 lg:grid-cols-2">
        <FormControl>
          <SelectInput
            name="status"
            rules={{ required: true }}
            control={control}
            label="Status"
            options={statusChoices}
            placeholder="Select Status"
            description="Status of the Comment Type"
            isClearable={false}
          />
        </FormControl>
        <FormControl>
          <SelectInput
            name="severity"
            rules={{ required: true }}
            control={control}
            label="Severity"
            options={severityChoices}
            placeholder="Select Severity Level"
            description="Severity level of the Comment Type"
            isClearable={false}
          />
        </FormControl>
        <FormControl className="col-span-full">
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
            description="Unique name for the Comment Type"
          />
        </FormControl>
        <FormControl className="col-span-full">
          <TextareaField
            name="description"
            rules={{ required: true }}
            control={control}
            label="Description"
            placeholder="Description"
            description="Description of the Comment Type"
          />
        </FormControl>
      </FormGroup>
    </Form>
  );
}

export function CommentTypeDialog({ onOpenChange, open }: TableSheetProps) {
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(commentTypeSchema),
    defaultValues: {
      status: "Active",
      name: "",
      severity: "Low",
      description: "",
    },
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "POST",
    path: "/comment-types/",
    successMessage: "Comment Type created successfully.",
    queryKeysToInvalidate: "commentTypes",
    closeModal: true,
    reset,
    errorMessage: "Failed to create new comment type.",
  });

  const onSubmit = (values: FormValues) => {
    mutation.mutate(values);
  };

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>Create New Comment Type</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Please fill out the form below to create a new Comment Type.
        </CredenzaDescription>
        <CredenzaBody>
          <form onSubmit={handleSubmit(onSubmit)}>
            <CommentTypeForm control={control} />
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
