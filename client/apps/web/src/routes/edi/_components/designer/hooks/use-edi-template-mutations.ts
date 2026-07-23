import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type {
  CreateEDITemplateDraftRequest,
  CreateEDITemplateRequest,
  EDITemplate,
  EDITemplateActionRequest,
  EDITemplateValidationResponse,
  EDITemplateVersion,
  ReplaceEDITemplateScriptLibrariesRequest,
  ReplaceEDITemplateSegmentsRequest,
  UpdateEDITemplateVersionRequest,
} from "@/types/edi";
import { useQueryClient } from "@tanstack/react-query";

type MutationOptions<TData, TVariables> = {
  onSuccess?: (data: TData, variables: TVariables, context: unknown) => unknown;
  onError?: (error: unknown, variables: TVariables, context: unknown) => unknown;
};

type TemplateVersionMutationVariables<TRequest> = {
  templateId: string;
  versionId: string;
  request: TRequest;
};

export function useInvalidateEDITemplateQueries(templateId: string, versionId: string) {
  const queryClient = useQueryClient();

  return async () => {
    await queryClient.invalidateQueries({ queryKey: queries.edi.templates._def });
    if (templateId) {
      await queryClient.invalidateQueries({
        queryKey: queries.edi.template(templateId).queryKey,
      });
      await queryClient.invalidateQueries({
        queryKey: queries.edi.templateVersions(templateId).queryKey,
      });
    }
    if (templateId && versionId) {
      await queryClient.invalidateQueries({
        queryKey: queries.edi.templateVersion(templateId, versionId).queryKey,
      });
    }
  };
}

export function useCreateEDITemplateMutation(
  options?: MutationOptions<EDITemplate, CreateEDITemplateRequest>,
) {
  return useApiMutation({
    mutationFn: (request: CreateEDITemplateRequest) =>
      apiService.ediService.createTemplate(request),
    ...options,
  });
}

export function useCreateEDITemplateDraftMutation(
  options?: MutationOptions<
    EDITemplateVersion,
    { templateId: string; request: CreateEDITemplateDraftRequest }
  >,
) {
  return useApiMutation({
    mutationFn: ({ templateId, request }) =>
      apiService.ediService.createTemplateDraft(templateId, request),
    ...options,
  });
}

export function useSaveEDITemplateMetadataMutation(
  options?: MutationOptions<
    EDITemplateVersion,
    TemplateVersionMutationVariables<UpdateEDITemplateVersionRequest>
  >,
) {
  return useApiMutation({
    mutationFn: ({ templateId, versionId, request }) =>
      apiService.ediService.updateTemplateVersion(templateId, versionId, request),
    ...options,
  });
}

export function useSaveEDITemplateSegmentsMutation(
  options?: MutationOptions<
    EDITemplateVersion,
    TemplateVersionMutationVariables<ReplaceEDITemplateSegmentsRequest>
  >,
) {
  return useApiMutation({
    mutationFn: ({ templateId, versionId, request }) =>
      apiService.ediService.replaceTemplateSegments(templateId, versionId, request),
    ...options,
  });
}

export function useSaveEDITemplateScriptsMutation(
  options?: MutationOptions<
    EDITemplateVersion,
    TemplateVersionMutationVariables<ReplaceEDITemplateScriptLibrariesRequest>
  >,
) {
  return useApiMutation({
    mutationFn: ({ templateId, versionId, request }) =>
      apiService.ediService.replaceTemplateScriptLibraries(templateId, versionId, request),
    ...options,
  });
}

export function useValidateEDITemplateMutation(
  options?: MutationOptions<
    EDITemplateValidationResponse,
    Pick<TemplateVersionMutationVariables<never>, "templateId" | "versionId">
  >,
) {
  return useApiMutation({
    mutationFn: ({ templateId, versionId }) =>
      apiService.ediService.validateTemplateVersion(templateId, versionId),
    ...options,
  });
}

export function useCertifyEDITemplateMutation(
  options?: MutationOptions<
    EDITemplateVersion,
    TemplateVersionMutationVariables<EDITemplateActionRequest>
  >,
) {
  return useApiMutation({
    mutationFn: ({ templateId, versionId, request }) =>
      apiService.ediService.certifyTemplateVersion(templateId, versionId, request),
    ...options,
  });
}

export function useValidateAndCertifyEDITemplateMutation(
  options?: MutationOptions<
    { version: EDITemplateVersion; validation: EDITemplateValidationResponse },
    TemplateVersionMutationVariables<EDITemplateActionRequest>
  >,
) {
  return useApiMutation({
    mutationFn: async ({ templateId, versionId, request }) => {
      const validation = await apiService.ediService.validateTemplateVersion(templateId, versionId);
      if (validation.diagnostics.some((diagnostic) => diagnostic.severity === "Error")) {
        throw new Error("Template has validation errors");
      }
      const version = await apiService.ediService.certifyTemplateVersion(
        templateId,
        versionId,
        request,
      );
      return { version, validation };
    },
    ...options,
  });
}

export function useActivateEDITemplateMutation(
  options?: MutationOptions<
    EDITemplateVersion,
    TemplateVersionMutationVariables<EDITemplateActionRequest>
  >,
) {
  return useApiMutation({
    mutationFn: ({ templateId, versionId, request }) =>
      apiService.ediService.activateTemplateVersion(templateId, versionId, request),
    ...options,
  });
}

export function useArchiveEDITemplateMutation(
  options?: MutationOptions<
    EDITemplateVersion,
    TemplateVersionMutationVariables<EDITemplateActionRequest>
  >,
) {
  return useApiMutation({
    mutationFn: ({ templateId, versionId, request }) =>
      apiService.ediService.archiveTemplateVersion(templateId, versionId, request),
    ...options,
  });
}
