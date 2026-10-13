import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useBreadcrumbLabel } from "@/hooks/use-breadcrumb-label";
import { saveDemotesToDraft } from "@/lib/formula-template-material";
import { formulaTemplateRoutes } from "@/lib/formula-template-routes";
import { queries } from "@/lib/queries";
import { invalidateFormulaTemplate } from "@/lib/queries/formula-template";
import { api } from "@trenova/shared/lib/api";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
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
import { toast } from "sonner";
import { SaveDemoteDialog } from "../_components/studio/dialogs/save-demote-dialog";
import { FormulaStudio } from "../_components/studio/formula-studio";

function StudioSkeleton() {
  return (
    <div className="bg-canvas flex h-full flex-col overflow-hidden" aria-busy>
      <div className="flex h-13 items-center gap-3 border-b px-3">
        <Skeleton className="size-7 rounded-md" />
        <Skeleton className="h-5 w-48" />
        <Skeleton className="h-5 w-64 rounded-full max-[760px]:hidden" />
        <span className="flex-1" />
        <Skeleton className="h-7 w-32" />
        <Skeleton className="h-7 w-20" />
      </div>
      <div className="grid min-h-0 flex-1 grid-cols-[minmax(0,1fr)_minmax(360px,420px)] min-[1180px]:grid-cols-[248px_minmax(0,1fr)_minmax(360px,420px)]">
        <div className="hidden flex-col gap-4 border-r p-4 min-[1180px]:flex">
          <Skeleton className="h-4 w-28" />
          <Skeleton className="h-3.5 w-40" />
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-20 w-full rounded-lg" />
        </div>
        <div className="mx-auto flex w-full max-w-200 flex-col gap-4 px-9 pt-7">
          <Skeleton className="h-8 w-2/3" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-6 w-56" />
          <Skeleton className="mt-6 h-5 w-32" />
          <Skeleton className="h-40 w-full rounded-lg" />
          <Skeleton className="mt-6 h-5 w-28" />
          <Skeleton className="h-28 w-full rounded-lg" />
        </div>
        <div className="flex flex-col gap-3 border-l p-3">
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-36 w-full rounded-lg" />
          <Skeleton className="h-7 w-full" />
          <Skeleton className="h-7 w-full" />
          <Skeleton className="h-7 w-full" />
        </div>
      </div>
    </div>
  );
}

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
    ...queries.formulaTemplate.get(id ?? ""),
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
      if (template && updated.status !== template.status && updated.status === "Draft") {
        toast(t("Saved as v{0} · back to Draft", updated.currentVersionNumber ?? 1), {
          description: t("The approved version keeps rating shipments until this one is approved."),
        });
      } else {
        toast(
          template && updated.currentVersionNumber !== template.currentVersionNumber
            ? t("Saved as v{0}", updated.currentVersionNumber ?? 1)
            : t("Changes saved"),
        );
      }
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
