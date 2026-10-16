import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useBreadcrumbLabel } from "@/hooks/use-breadcrumb-label";
import { formulaTemplateRoutes } from "@/lib/formula-template-routes";
import { formulaTemplate, invalidateFormulaTemplate } from "@/lib/queries/formula-template";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { apiService } from "@/services/api";
import {
  DEFAULT_ROUNDING_MODE,
  DEFAULT_ROUNDING_PRECISION,
  formulaTemplateSchema,
  type FormulaTemplate,
  type FormulaTemplateFormValues,
} from "@trenova/shared/types/formula-template";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { FormProvider, useForm, useFormState } from "react-hook-form";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import { FormulaStudio, type StudioSaveExtras } from "../_components/studio/formula-studio";

export const prefetch: RoutePrefetch = () => [formulaTemplate.schema("shipment")];

export function FormulaStudioCreatePage() {
  const t = useT();

  const navigate = useNavigate();
  const queryClient = useQueryClient();
  useBreadcrumbLabel("New Formula Template");
  const [createdId, setCreatedId] = useState<string | null>(null);

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
    mutationFn: async ({
      values,
      extras,
    }: {
      values: FormulaTemplateFormValues;
      extras: StudioSaveExtras;
    }) =>
      apiService.formulaTemplateService.create({
        ...(values as FormulaTemplate),
        testCases: extras.testCases,
      }),
    onSuccess: async (created) => {
      await invalidateFormulaTemplate(queryClient);
      // Reset onto the saved record, then leave once the form has reported
      // itself clean: navigating in the same tick found the unsaved-changes
      // guard still armed, which flashed its dialog and cancelled the
      // redirect, so the next save created the template a second time.
      form.reset(created as FormulaTemplateFormValues);
      setCreatedId(created.id ?? "");
    },
    form,
    resourceName: "Formula Template",
  });

  const { isDirty } = useFormState({ control: form.control });
  useEffect(() => {
    if (createdId === null || isDirty) return;
    void navigate(createdId ? formulaTemplateRoutes.edit(createdId) : formulaTemplateRoutes.list, {
      replace: true,
    });
  }, [createdId, isDirty, navigate]);

  // `mutate` reports failure through the hook's error handling; `mutateAsync`
  // would also reject, and a rejection nobody awaits is an unhandled promise.
  const onSave = (extras: StudioSaveExtras) =>
    void form.handleSubmit(
      (values) => mutate({ values, extras }),
      () => {
        toast(t("Fix the highlighted fields to create the template"));
      },
    )();

  return (
    <FormProvider {...form}>
      <FormulaStudio
        mode="create"
        template={null}
        isSubmitting={form.formState.isSubmitting}
        onSave={onSave}
      />
    </FormProvider>
  );
}
