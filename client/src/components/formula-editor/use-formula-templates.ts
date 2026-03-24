import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  createFormulaTemplate,
  deleteFormulaTemplate,
  getFormulaTemplate,
  listFormulaTemplates,
  type ListFormulaTemplatesParams,
  updateFormulaTemplate,
} from "@/lib/formula-template-api";
import type { FormulaTemplate } from "@/types/formula-template";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

const QUERY_KEY = "formula-templates";

export function useFormulaTemplates(params?: ListFormulaTemplatesParams) {
  return useQuery({
    queryKey: [QUERY_KEY, params],
    queryFn: () => listFormulaTemplates(params),
  });
}

export function useFormulaTemplate(id: string | undefined) {
  return useQuery({
    queryKey: [QUERY_KEY, id],
    queryFn: () => getFormulaTemplate(id!),
    enabled: !!id,
  });
}

export function useCreateFormulaTemplate() {
  const queryClient = useQueryClient();

  return useApiMutation({
    mutationFn: createFormulaTemplate,
    resourceName: "Formula Template",
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: [QUERY_KEY] });
      toast.success("Template created", {
        description: "Your formula template has been created successfully.",
      });
    },
  });
}

export function useUpdateFormulaTemplate() {
  const queryClient = useQueryClient();

  return useApiMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string;
      data: Partial<FormulaTemplate>;
    }) => updateFormulaTemplate(id, data),
    resourceName: "Formula Template",
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: [QUERY_KEY] });
      toast.success("Template updated", {
        description: "Your formula template has been updated successfully.",
      });
    },
  });
}

export function useDeleteFormulaTemplate() {
  const queryClient = useQueryClient();

  return useApiMutation({
    mutationFn: deleteFormulaTemplate,
    resourceName: "Formula Template",
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: [QUERY_KEY] });
      toast.success("Template deleted", {
        description: "The formula template has been deleted.",
      });
    },
  });
}

export function useToggleFormulaTemplateStatus() {
  const queryClient = useQueryClient();

  return useApiMutation({
    mutationFn: async (template: FormulaTemplate) => {
      const newStatus = template.status === "Active" ? "Inactive" : "Active";
      return updateFormulaTemplate(template.id!, {
        status: newStatus,
      });
    },
    resourceName: "Formula Template",
    onSuccess: (updated) => {
      void queryClient.invalidateQueries({ queryKey: [QUERY_KEY] });
      toast.success(
        updated.status === "Active"
          ? "Template activated"
          : "Template deactivated",
        {
          description: `"${updated.name}" is now ${updated.status.toLowerCase()}.`,
        },
      );
    },
  });
}

export function useDuplicateFormulaTemplate() {
  const queryClient = useQueryClient();

  return useApiMutation({
    mutationFn: async (template: FormulaTemplate) => {
      const { ...data } = template;
      return createFormulaTemplate({
        ...data,
        name: `${template.name} (Copy)`,
        status: "Draft",
      });
    },
    resourceName: "Formula Template",
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: [QUERY_KEY] });
      toast.success("Template duplicated", {
        description: "A copy of the template has been created as a draft.",
      });
    },
  });
}
