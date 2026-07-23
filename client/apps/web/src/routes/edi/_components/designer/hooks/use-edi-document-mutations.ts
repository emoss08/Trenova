import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type {
  EDIDocumentPreview,
  EDIX12Inspection,
  EDIMessage,
  EDIPartnerDocumentProfile,
  GenerateEDIDocumentRequest,
  InspectX12Request,
  PreviewEDIDocumentRequest,
  UpsertEDIPartnerDocumentProfileRequest,
} from "@trenova/shared/types/edi";
import { useQueryClient } from "@tanstack/react-query";

type MutationOptions<TData, TVariables> = {
  onSuccess?: (data: TData, variables: TVariables, context: unknown) => unknown;
  onError?: (error: unknown, variables: TVariables, context: unknown) => unknown;
};

export function useSaveEDIDocumentProfileMutation(
  options?: MutationOptions<
    EDIPartnerDocumentProfile,
    { profileId: string; request: UpsertEDIPartnerDocumentProfileRequest }
  >,
) {
  return useApiMutation({
    mutationFn: ({ profileId, request }) => {
      if (profileId) {
        return apiService.ediService.updatePartnerDocumentProfile(profileId, request);
      }
      return apiService.ediService.createPartnerDocumentProfile(request);
    },
    ...options,
  });
}

export function usePreviewEDIDocumentMutation(
  options?: MutationOptions<EDIDocumentPreview, PreviewEDIDocumentRequest>,
) {
  return useApiMutation({
    mutationFn: (request: PreviewEDIDocumentRequest) =>
      apiService.ediService.previewDocument(request),
    ...options,
  });
}

export function useGenerateEDIDocumentMutation(
  options?: MutationOptions<EDIMessage, GenerateEDIDocumentRequest>,
) {
  return useApiMutation({
    mutationFn: (request: GenerateEDIDocumentRequest) =>
      apiService.ediService.generateDocument(request),
    ...options,
  });
}

export function useInspectX12Mutation(
  options?: MutationOptions<EDIX12Inspection, InspectX12Request>,
) {
  return useApiMutation({
    mutationFn: (request: InspectX12Request) => apiService.ediService.inspectX12(request),
    resourceName: "X12 inspection",
    ...options,
  });
}

export function useInvalidateEDIDocumentProfiles() {
  const queryClient = useQueryClient();
  return async (profile?: EDIPartnerDocumentProfile) => {
    if (profile) {
      queryClient.setQueryData(
        [
          "autocomplete-option",
          "/edi/document-profiles/select-options/",
          "/edi/document-profiles/",
          profile.id,
        ],
        profile,
      );
    }

    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queries.edi.documentProfiles._def }),
      queryClient.invalidateQueries({
        queryKey: ["autocomplete-search", "/edi/document-profiles/select-options/"],
      }),
      queryClient.invalidateQueries({
        queryKey: ["autocomplete-option", "/edi/document-profiles/select-options/"],
      }),
    ]);
  };
}

export function useInvalidateEDIMessageArchive() {
  const queryClient = useQueryClient();
  return () => queryClient.invalidateQueries({ queryKey: queries.edi.messages._def });
}
