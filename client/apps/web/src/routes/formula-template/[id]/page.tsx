import { useApiMutation } from "@/hooks/use-api-mutation";
import { saveDemotesToDraft } from "@/lib/formula-template-material";
import { queries } from "@/lib/queries";
import { invalidateFormulaTemplate } from "@/lib/queries/formula-template";
import { api } from "@trenova/shared/lib/api";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
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
import { AlertCircleIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { useNavigate, useParams } from "react-router";
import { toast } from "sonner";
import { FormulaStudio } from "../_components/studio/formula-studio";

function StudioSkeleton() {
  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="flex items-center justify-between gap-3 border-b px-4 py-2.5">
        <div className="flex min-w-0 items-center gap-3">
          <Skeleton className="size-7 rounded-md" />
          <Skeleton className="h-5 w-44" />
          <Skeleton className="h-5 w-16 rounded-full" />
          <Skeleton className="h-5 w-10 rounded-full" />
        </div>
        <div className="flex shrink-0 items-center gap-1.5">
          <Skeleton className="h-7 w-32" />
          <Skeleton className="h-7 w-24" />
          <Skeleton className="size-7" />
        </div>
      </div>

      <div className="flex min-h-0 flex-1">
        <div className="min-w-0 flex-[55] space-y-4 overflow-hidden border-r p-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Skeleton className="size-8 rounded-lg" />
              <div className="space-y-1.5">
                <Skeleton className="h-3.5 w-32" />
                <Skeleton className="h-3 w-52" />
              </div>
            </div>
            <Skeleton className="size-4" />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <Skeleton className="h-8" />
            <Skeleton className="h-8" />
            <Skeleton className="h-8" />
            <Skeleton className="h-8" />
          </div>
          <div className="space-y-2 pt-2">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <Skeleton className="size-8 rounded-lg" />
                <Skeleton className="h-3.5 w-24" />
              </div>
              <Skeleton className="h-6 w-28" />
            </div>
            <Skeleton className="h-44 w-full rounded-md" />
          </div>
          <div className="space-y-2 pt-2">
            <div className="flex items-center gap-3">
              <Skeleton className="size-8 rounded-lg" />
              <Skeleton className="h-3.5 w-36" />
            </div>
            <Skeleton className="h-9 w-full rounded-md" />
            <Skeleton className="h-9 w-full rounded-md" />
          </div>
        </div>

        <div className="flex min-w-0 flex-[45] flex-col overflow-hidden">
          <div className="flex gap-1 border-b px-2 pt-1.5 pb-1">
            <Skeleton className="h-6 w-24" />
            <Skeleton className="h-6 w-20" />
          </div>
          <div className="basis-[55%] space-y-3 border-b p-4">
            <Skeleton className="h-3 w-24" />
            <Skeleton className="h-9 w-40" />
            <Skeleton className="h-3.5 w-32" />
            <div className="space-y-1.5 pt-2">
              <Skeleton className="h-7 w-full rounded-md" />
              <Skeleton className="h-7 w-full rounded-md" />
              <Skeleton className="h-7 w-full rounded-md" />
            </div>
          </div>
          <div className="min-h-0 flex-1 space-y-2 p-3">
            <Skeleton className="h-8 w-full rounded-md" />
            <Skeleton className="mt-3 h-3 w-24" />
            <div className="space-y-1.5">
              <Skeleton className="h-7 w-full rounded-md" />
              <Skeleton className="h-7 w-full rounded-md" />
              <Skeleton className="h-7 w-full rounded-md" />
              <Skeleton className="h-7 w-full rounded-md" />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export function FormulaStudioEditPage() {
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
        toast.success("Formula template saved as a draft", {
          description:
            "The change returned it to Draft. Shipments cannot be rated with it until it is approved again.",
        });
      } else {
        toast.success("Formula template saved");
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
        <p className="text-sm font-medium">This formula template could not be loaded.</p>
        <Button
          variant="outline"
          size="sm"
          onClick={() => void navigate("/billing/configuration-files/formula-templates")}
        >
          Back to Formula Templates
        </Button>
      </div>
    );
  }

  if (isLoading || !isSeated) {
    return <StudioSkeleton />;
  }

  const inReview = template?.status === "InReview";

  return (
    <FormProvider {...form}>
      <FormulaStudio
        mode="edit"
        template={template ?? null}
        isSubmitting={form.formState.isSubmitting}
        onSave={() => void onSave()}
      />

      <AlertDialog
        open={pendingDemotingSave !== null}
        onOpenChange={(open) => {
          if (!open) setPendingDemotingSave(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="text-lg font-semibold">
              {inReview ? "Cancel the review and save?" : "Take this template out of production?"}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {inReview
                ? "This changes what the reviewer is looking at, so the template returns to Draft and must be submitted again."
                : "This changes what the template computes, so it returns to Draft and stops rating shipments until it is approved again. Name and description edits do not do this."}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="outline" size="default">
              Keep editing
            </AlertDialogCancel>
            <AlertDialogAction variant="destructive" size="default" onClick={confirmDemotingSave}>
              Save and return to Draft
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </FormProvider>
  );
}
