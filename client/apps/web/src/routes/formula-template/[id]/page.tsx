import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useBreadcrumbLabel } from "@/hooks/use-breadcrumb-label";
import { useSeatedForm, type SeatedRecord } from "@/hooks/use-seated-form";
import { saveDemotesToDraft } from "@/lib/formula-template-material";
import { rulesSync } from "@trenova/shared/rules/runtime";
import { formulaTemplateRoutes } from "@/lib/formula-template-routes";
import { formulaTemplate } from "@/lib/queries/formula-template";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { invalidateFormulaTemplate } from "@/lib/queries/formula-template";
import { apiService } from "@/services/api";
import { Button } from "@trenova/shared/components/ui/button";
import {
  formulaTemplateSchema,
  type FormulaTemplate,
  type FormulaTemplateFormValues,
} from "@trenova/shared/types/formula-template";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertCircleIcon } from "@trenova/shared/components/icons";
import { useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { useNavigate, useParams } from "react-router";
import { SaveDemoteDialog } from "../_components/studio/dialogs/save-demote-dialog";
import { FormulaStudio } from "../_components/studio/formula-studio";
import { StudioSkeleton } from "@/components/tariff-studio/studio-skeletons";
import {
  NewerVersionProvider,
  useNewerVersionCompare,
} from "@/components/tariff-studio/newer-version";

export const prefetch: RoutePrefetch = ({ params }) => [
  formulaTemplate.context,
  ...(params.id ? [formulaTemplate.get(params.id)] : []),
];

const SAVED_FIELDS = ["name", "description", "source"] as const;

type TemplateSave = {
  values: FormulaTemplateFormValues;
  sent: FormulaTemplateFormValues;
};

function seatedTemplate(template: FormulaTemplate): SeatedRecord<FormulaTemplateFormValues> {
  return { values: template as FormulaTemplateFormValues, version: template.version ?? 0 };
}

/**
 * Each template gets its own seat: a jump to another one (a fork, a lineage link) starts from
 * nothing rather than comparing its version with the last template's.
 */
export function FormulaStudioEditPage() {
  const { id } = useParams<{ id: string }>();
  return <FormulaTemplateEditor key={id} />;
}

function FormulaTemplateEditor() {
  const t = useT();

  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [pendingDemotingSave, setPendingDemotingSave] = useState<FormulaTemplateFormValues | null>(
    null,
  );
  const compare = useNewerVersionCompare();

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
    defaultValues: { name: "", description: "", source: "", status: "Draft" },
  });

  const seat = useSeatedForm({
    form,
    record: template ? seatedTemplate(template) : null,
    fields: SAVED_FIELDS,
  });

  const { mutate } = useApiMutation({
    mutationFn: async ({ values }: TemplateSave) => {
      return apiService.formulaTemplateService.update(id ?? "", {
        description: values.description ?? "",
        source: values.source,
        version: seat.version ?? undefined,
      });
    },
    onSuccess: async (updated, { sent }) => {
      // Reseat the form on the saved record right away so the dirty flag —
      // and with it the unsaved-changes guard — clears without waiting for
      // the query refetch.
      seat.seatSaved(seatedTemplate(updated), sent);
      await invalidateFormulaTemplate(queryClient);
    },
    form,
    resourceName: "Formula Template",
  });

  const send = (values: FormulaTemplateFormValues) => mutate({ values, sent: form.getValues() });

  const onSave = form.handleSubmit((values) => {
    if (!seat.isSeated) return;
    // A newer version saved since the edits began would refuse this save;
    // the person sees what it changed and chooses before anything is sent.
    if (seat.held) {
      compare.open(form.getValues());
      return;
    }
    // A material edit to an approved template takes it out of production the
    // moment it is saved; that is worth a pause, not a toast after the fact.
    if (saveDemotesToDraft(rulesSync(), template ?? null, values)) {
      setPendingDemotingSave(values);
      return;
    }
    send(values);
  });

  const confirmDemotingSave = () => {
    if (pendingDemotingSave) {
      send(pendingDemotingSave);
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

  if (isLoading || !seat.isSeated) {
    return <StudioSkeleton />;
  }

  return (
    <FormProvider {...form}>
      <NewerVersionProvider
        seat={seat}
        compare={compare}
        values={() => form.getValues()}
        savedVersionNumber={template?.currentVersionNumber ?? null}
      >
        <FormulaStudio
          mode="edit"
          template={template ?? null}
          isSubmitting={form.formState.isSubmitting}
          onSave={() => void onSave()}
          onTemplateChanged={(changed) => seat.seat(seatedTemplate(changed))}
        />
      </NewerVersionProvider>

      <SaveDemoteDialog
        open={pendingDemotingSave !== null}
        template={template ?? null}
        onCancel={() => setPendingDemotingSave(null)}
        onConfirm={confirmDemotingSave}
      />
    </FormProvider>
  );
}
