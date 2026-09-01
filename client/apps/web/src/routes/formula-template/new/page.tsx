import { useApiMutation } from "@/hooks/use-api-mutation";
import { invalidateFormulaTemplate } from "@/lib/queries/formula-template";
import { api } from "@trenova/shared/lib/api";
import {
  formulaTemplateSchema,
  type FormulaTemplate,
  type FormulaTemplateFormValues,
} from "@trenova/shared/types/formula-template";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { FormProvider, useForm } from "react-hook-form";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import { FormulaStudio } from "../_components/studio/formula-studio";

export function FormulaStudioCreatePage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const form = useForm<FormulaTemplateFormValues>({
    resolver: zodResolver(formulaTemplateSchema),
    defaultValues: {
      name: "",
      description: "",
      type: "FreightCharge",
      expression: "",
      status: "Draft",
      schemaId: "shipment",
      variableDefinitions: [],
      breakdownDefinitions: [],
      minCharge: null,
      maxCharge: null,
    },
  });

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: FormulaTemplateFormValues) => {
      return api.post<FormulaTemplate>("/formula-templates/", values);
    },
    onSuccess: async (created) => {
      toast.success("Formula template created", {
        description: "It starts as a draft. Submit it for review when it is ready.",
      });
      await invalidateFormulaTemplate(queryClient);
      if (created.id) {
        void navigate(`/billing/configuration-files/formula-templates/${created.id}/edit`, {
          replace: true,
        });
      } else {
        void navigate("/billing/configuration-files/formula-templates");
      }
    },
    form,
    resourceName: "Formula Template",
  });

  const onSave = form.handleSubmit(async (values) => {
    await mutateAsync(values);
  });

  return (
    <FormProvider {...form}>
      <FormulaStudio
        mode="create"
        template={null}
        isSubmitting={form.formState.isSubmitting}
        onSave={() => void onSave()}
      />
    </FormProvider>
  );
}
