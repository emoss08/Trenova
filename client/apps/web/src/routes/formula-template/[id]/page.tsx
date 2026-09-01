import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { invalidateFormulaTemplate } from "@/lib/queries/formula-template";
import { api } from "@trenova/shared/lib/api";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
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
    <div className="space-y-4 p-4">
      <div className="flex items-center justify-between">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-8 w-32" />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <Skeleton className="h-[60vh]" />
        <div className="space-y-4">
          <Skeleton className="h-[30vh]" />
          <Skeleton className="h-[26vh]" />
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
    },
  });

  useEffect(() => {
    if (template && template.version !== seatedVersion) {
      form.reset(template as FormulaTemplateFormValues);
      setSeatedVersion(template.version ?? 0);
    }
  }, [template, seatedVersion, form]);

  const { mutateAsync } = useApiMutation({
    mutationFn: async (values: FormulaTemplateFormValues) => {
      return api.put<FormulaTemplate>(`/formula-templates/${id}/`, values);
    },
    onSuccess: async () => {
      toast.success("Formula template saved");
      await invalidateFormulaTemplate(queryClient);
    },
    form,
    resourceName: "Formula Template",
  });

  const isSeated = seatedVersion !== null;

  const onSave = form.handleSubmit(async (values) => {
    if (!isSeated) return;
    await mutateAsync(values);
  });

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

  return (
    <FormProvider {...form}>
      <FormulaStudio
        mode="edit"
        template={template ?? null}
        isSubmitting={form.formState.isSubmitting}
        onSave={() => void onSave()}
      />
    </FormProvider>
  );
}
