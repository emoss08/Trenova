import { useApiMutation } from "@/hooks/use-api-mutation";
import { useBreadcrumbLabel } from "@/hooks/use-breadcrumb-label";
import { formulaLibraryRoutes } from "@/lib/formula-library-routes";
import { invalidateFormulaLibrary } from "@/lib/queries/formula-library";
import { formulaTemplate } from "@/lib/queries/formula-template";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { apiService } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "@trenova/shared/components/ui/toast";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  formulaLibrarySchema,
  type FormulaLibraryFormValues,
} from "@trenova/shared/types/formula-library";
import { useEffect, useState } from "react";
import { FormProvider, useForm, useFormState } from "react-hook-form";
import { useNavigate } from "react-router";
import { EMPTY_LIBRARY, libraryFormValues } from "../_components/studio/library-form";
import { LibraryStudio } from "../_components/studio/library-studio";

export const prefetch: RoutePrefetch = () => [formulaTemplate.context];

export function FormulaLibraryCreatePage() {
  const t = useT();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  useBreadcrumbLabel(t("New formula library"));
  const [createdId, setCreatedId] = useState<string | null>(null);

  const form = useForm<FormulaLibraryFormValues>({
    resolver: zodResolver(formulaLibrarySchema),
    defaultValues: EMPTY_LIBRARY,
  });

  const { mutate } = useApiMutation({
    mutationFn: (values: FormulaLibraryFormValues) =>
      apiService.formulaLibraryService.create({
        name: values.name.trim(),
        description: values.description ?? "",
        source: values.source,
      }),
    onSuccess: async (created) => {
      await invalidateFormulaLibrary(queryClient);
      // Reset onto the saved record, then leave once the form reports itself clean, so the
      // unsaved-changes guard is disarmed before the navigation it would otherwise block.
      form.reset(libraryFormValues(created));
      setCreatedId(created.id ?? "");
    },
    form,
    resourceName: "Formula Library",
  });

  const { isDirty } = useFormState({ control: form.control });
  useEffect(() => {
    if (createdId === null || isDirty) return;
    void navigate(createdId ? formulaLibraryRoutes.edit(createdId) : formulaLibraryRoutes.list, {
      replace: true,
    });
  }, [createdId, isDirty, navigate]);

  const onSave = () =>
    void form.handleSubmit(
      (values) => mutate(values),
      () => toast(t("Fix the highlighted fields to create the library")),
    )();

  return (
    <FormProvider {...form}>
      <LibraryStudio
        mode="create"
        library={null}
        isSubmitting={form.formState.isSubmitting}
        onSave={onSave}
      />
    </FormProvider>
  );
}
