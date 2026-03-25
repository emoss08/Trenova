import { SelectField } from "@/components/fields/select-field";
import { FormSaveDock } from "@/components/form-save-dock";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { caseFormatChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { dataEntryControlSchema, type DataEntryControl } from "@/types/data-entry-control";
import { zodResolver } from "@hookform/resolvers/zod";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, useForm, useFormContext } from "react-hook-form";

export default function DataEntryControlForm() {
  const { data } = useSuspenseQuery({
    ...queries.dataEntryControl.get(),
  });

  const form = useForm<DataEntryControl>({
    resolver: zodResolver(dataEntryControlSchema),
    defaultValues: data,
  });

  const { handleSubmit, setError, reset } = form;

  const { mutateAsync } = useOptimisticMutation({
    queryKey: queries.dataEntryControl.get._def,
    mutationFn: async (values: DataEntryControl) =>
      apiService.dataEntryControlService.update(values),
    resourceName: "Data Entry Control",
    resetForm: reset,
    setFormError: setError,
    invalidateQueries: [queries.dataEntryControl.get._def],
  });

  const onSubmit = useCallback(
    async (values: DataEntryControl) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="flex flex-col gap-4 pb-14">
          <CaseFormattingForm />
          <FormSaveDock saveButtonContent="Save Changes" />
        </div>
      </Form>
    </FormProvider>
  );
}

function CaseFormattingForm() {
  const { control } = useFormContext<DataEntryControl>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Case Formatting Rules</CardTitle>
        <CardDescription>
          Control how text is automatically formatted when entering data. These rules apply
          system-wide to standardize codes, names, emails, and city fields.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              name="codeCase"
              label="Code Case"
              description="Formatting applied to code fields (e.g., equipment codes, fleet codes)."
              options={caseFormatChoices}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="nameCase"
              label="Name Case"
              description="Formatting applied to name fields (e.g., commodity names, hazmat names)."
              options={caseFormatChoices}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="emailCase"
              label="Email Case"
              description="Formatting applied to email address fields."
              options={caseFormatChoices}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="cityCase"
              label="City Case"
              description="Formatting applied to city name fields."
              options={caseFormatChoices}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}
