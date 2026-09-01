import { useApiMutation } from "@/hooks/use-api-mutation";
import { formulaTemplateRoutes } from "@/lib/formula-template-routes";
import { invalidateFormulaTemplate } from "@/lib/queries/formula-template";
import { api } from "@trenova/shared/lib/api";
import {
  DEFAULT_ROUNDING_MODE,
  DEFAULT_ROUNDING_PRECISION,
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
      roundingMode: DEFAULT_ROUNDING_MODE,
      roundingPrecision: DEFAULT_ROUNDING_PRECISION,
    },
  });

  const { mutate } = useApiMutation({
    mutationFn: async (values: FormulaTemplateFormValues) => {
      return api.post<FormulaTemplate>("/formula-templates/", values);
    },
    onSuccess: async (created) => {
      toast.success("Formula template created", {
        description: "It starts as a draft. Submit it for review when it is ready.",
      });
      await invalidateFormulaTemplate(queryClient);
      // Reset onto the saved record before navigating so the unsaved-changes
      // guard sees a clean form and lets the redirect through.
      form.reset(created as FormulaTemplateFormValues);
      if (created.id) {
        void navigate(formulaTemplateRoutes.edit(created.id), { replace: true });
      } else {
        void navigate(formulaTemplateRoutes.list);
      }
    },
    form,
    resourceName: "Formula Template",
  });

  // `mutate` reports failure through the hook's error handling; `mutateAsync`
  // would also reject, and a rejection nobody awaits is an unhandled promise.
  const onSave = form.handleSubmit((values) => mutate(values));

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
