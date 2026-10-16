import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useBreadcrumbLabel } from "@/hooks/use-breadcrumb-label";
import { saveDemotesToDraft } from "@/lib/formula-template-material";
import { formulaTemplateRoutes } from "@/lib/formula-template-routes";
import { formulaTemplate } from "@/lib/queries/formula-template";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { invalidateFormulaTemplate } from "@/lib/queries/formula-template";
import { api } from "@trenova/shared/lib/api";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DEFAULT_ROUNDING_MODE,
  DEFAULT_ROUNDING_PRECISION,
  formulaTemplateSchema,
  type FormulaTemplate,
  type FormulaTemplateFormValues,
} from "@trenova/shared/types/formula-template";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertCircleIcon } from "@trenova/shared/components/icons";
import { useEffect, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { useNavigate, useParams } from "react-router";
import { SaveDemoteDialog } from "../_components/studio/dialogs/save-demote-dialog";
import { FormulaStudio } from "../_components/studio/formula-studio";
import { StudioSkeleton } from "../_components/studio/studio-skeletons";

export const prefetch: RoutePrefetch = ({ params }) => [
  formulaTemplate.schema("shipment"),
  ...(params.id ? [formulaTemplate.get(params.id)] : []),
];

export function FormulaStudioEditPage() {
  const t = useT();

  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [seatedVersion, setSeatedVersion] = useState<number | null>(null);
  const [pendingDemotingSave, setPendingDemotingSave] = useState<FormulaTemplateFormValues | null>(
    null,
  );

  const {
    data: template,
    isLoading,
    isError,
  } = useQuery({
    ...formulaTemplate.get(id ?? ""),
    enabled: !!id,
  });
  useBreadcrumbLabel(template?.name);

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

  useEffect(() => {
    if (template && template.version !== seatedVersion) {
      form.reset(template as FormulaTemplateFormValues);
      setSeatedVersion(template.version ?? 0);
    }
  }, [template, seatedVersion, form]);

  const { mutate } = useApiMutation({
    mutationFn: async (values: FormulaTemplateFormValues) => {
      return api.put<FormulaTemplate>(`/formula-templates/${id}/`, values);
    },
    onSuccess: async (updated) => {
      // Reseat the form on the saved record right away so the dirty flag —
      // and with it the unsaved-changes guard — clears without waiting for
      // the query refetch.
      form.reset(updated as FormulaTemplateFormValues);
      setSeatedVersion(updated.version ?? 0);
      await invalidateFormulaTemplate(queryClient);
    },
    form,
    resourceName: "Formula Template",
  });

  const isSeated = seatedVersion !== null;

  const onSave = form.handleSubmit((values) => {
    if (!isSeated) return;
    // A material edit to an approved template takes it out of production the
    // moment it is saved; that is worth a pause, not a toast after the fact.
    if (saveDemotesToDraft(template ?? null, values)) {
      setPendingDemotingSave(values);
      return;
    }
    mutate(values);
  });

  const confirmDemotingSave = () => {
    if (pendingDemotingSave) {
      mutate(pendingDemotingSave);
    }
    setPendingDemotingSave(null);
  };

  if (isError) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 text-center">
        <AlertCircleIcon className="text-destructive size-10" />
        <p className="text-sm font-medium">{t("This formula template could not be loaded.")}</p>
        <Button
          variant="outline"
          size="sm"
          onClick={() => void navigate(formulaTemplateRoutes.list)}
        >
          {t("Back to formula templates")}
        </Button>
      </div>
    );
  }

  if (isLoading || !isSeated) {
    return <StudioSkeleton />;
  }

  return (
    <FormProvider {...form}>
      <FormulaStudio
        mode="edit"
        template={template ?? null}
        isSubmitting={form.formState.isSubmitting}
        onSave={() => void onSave()}
      />

      <SaveDemoteDialog
        open={pendingDemotingSave !== null}
        template={template ?? null}
        onCancel={() => setPendingDemotingSave(null)}
        onConfirm={confirmDemotingSave}
      />
    </FormProvider>
  );
}
