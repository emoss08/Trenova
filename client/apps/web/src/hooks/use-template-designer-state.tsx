import { listEdiTemplatesGraphQL } from "@/lib/graphql/edi-templates";
import { queries } from "@/lib/queries";
import {
  getChangedTemplateUrlStatePatch,
  getTemplateDesignerSelectionPatch,
  type TemplateDesignerUrlStatePatch,
  useTemplateDesignerUrlState,
} from "@/routes/edi/_components/designer/hooks/use-edi-designer-url-state";
import {
  useInvalidateEDITemplateQueries,
  useValidateEDITemplateMutation,
} from "@/routes/edi/_components/designer/hooks/use-edi-template-mutations";
import {
  getTemplateDesignerVersionDraftKey,
  useTemplateDesignerStore,
} from "@/stores/template-designer-store";
import type { EDIDiagnostic, EDITemplate, EDITemplateVersion } from "@trenova/shared/types/edi";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { createContext, useCallback, useContext, useMemo, type ReactNode } from "react";
import { toast } from "sonner";

const emptyTemplates: EDITemplate[] = [];
const emptyVersions: EDITemplateVersion[] = [];
const templateDesignerTemplatePageSize = 25;

function templateMatchesCurrentFilters(
  template: EDITemplate,
  filters: {
    search: string;
    status: string;
    transactionSet: string;
    direction: string;
  },
) {
  const search = filters.search.trim().toLowerCase();
  const matchesSearch =
    !search ||
    template.name.toLowerCase().includes(search) ||
    (template.description ?? "").toLowerCase().includes(search) ||
    template.transactionSet.toLowerCase().includes(search);

  return (
    matchesSearch &&
    (!filters.status || template.status === filters.status) &&
    (!filters.transactionSet || template.transactionSet === filters.transactionSet) &&
    (!filters.direction || template.direction === filters.direction)
  );
}

export function useTemplateDesignerUrlActions() {
  const [templateUrlState, setTemplateUrlState] = useTemplateDesignerUrlState();

  const patchTemplateUrlState = useCallback(
    (patch: TemplateDesignerUrlStatePatch) => {
      const changedPatch = getChangedTemplateUrlStatePatch(templateUrlState, patch);
      if (!changedPatch) return;
      void setTemplateUrlState(changedPatch);
    },
    [setTemplateUrlState, templateUrlState],
  );

  return {
    templateUrlState,
    patchTemplateUrlState,
  };
}

export function useTemplateDesignerTemplateListInfiniteQuery() {
  const [templateUrlState] = useTemplateDesignerUrlState();
  const { templateSearch, templateStatus, templateTransactionSet, templateDirection } =
    templateUrlState;

  return useInfiniteQuery({
    queryKey: [
      ...queries.edi.templates._def,
      {
        query: templateSearch,
        status: templateStatus,
        transactionSet: templateTransactionSet,
        direction: templateDirection,
        limit: templateDesignerTemplatePageSize,
      },
    ],
    queryFn: async ({ pageParam }) =>
      listEdiTemplatesGraphQL({
        first: templateDesignerTemplatePageSize,
        after: pageParam,
        query: templateSearch,
        status: templateStatus,
        transactionSet: templateTransactionSet,
        direction: templateDirection,
      }),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) =>
      lastPage.hasNextPage && lastPage.endCursor ? lastPage.endCursor : undefined,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
  });
}

export function useSelectedTemplateDesignerIds() {
  const [templateUrlState] = useTemplateDesignerUrlState();

  return {
    selectedTemplateId: templateUrlState.templateId,
    selectedVersionId: templateUrlState.versionId,
    selectedSegmentId: templateUrlState.segmentId,
    selectedElementPosition: templateUrlState.elementPosition,
  };
}

export function useSelectedTemplateDesignerData() {
  const { selectedTemplateId, selectedVersionId } = useSelectedTemplateDesignerIds();
  const templatesQuery = useTemplateDesignerTemplateListInfiniteQuery();
  const templateQuery = useQuery({
    ...queries.edi.template(selectedTemplateId),
    enabled: !!selectedTemplateId,
  });
  const versionsQuery = useQuery({
    ...queries.edi.templateVersions(selectedTemplateId),
    enabled: !!selectedTemplateId,
  });
  const versionQuery = useQuery({
    ...queries.edi.templateVersion(selectedTemplateId, selectedVersionId),
    enabled: !!selectedTemplateId && !!selectedVersionId,
  });

  const templates = templatesQuery.data?.pages.flatMap((page) => page.results) ?? emptyTemplates;
  const selectedTemplate =
    templateQuery.data ?? templates.find((template) => template.id === selectedTemplateId);
  const versions = versionsQuery.data ?? selectedTemplate?.versions ?? emptyVersions;
  const selectedVersion =
    versionQuery.data ??
    versions.find((version) => version.id === selectedVersionId) ??
    selectedTemplate?.activeVersion ??
    selectedTemplate?.versions[0];

  return {
    templates,
    versions,
    selectedTemplate,
    selectedVersion,
    templatesQuery,
    templateQuery,
    versionsQuery,
    versionQuery,
  };
}

export function useTemplateDesignerDirtyState() {
  const segmentsDirty = useTemplateDesignerStore((state) => state.segmentsDirty);
  const scriptsDirty = useTemplateDesignerStore((state) => state.scriptsDirty);
  const metadataDirty = useTemplateDesignerStore((state) => state.metadataDirty);

  return {
    segmentsDirty,
    scriptsDirty,
    metadataDirty,
    hasUnsavedChanges: segmentsDirty || scriptsDirty || metadataDirty,
  };
}

export function useSelectedTemplateDesignerSegmentElement() {
  const { selectedSegmentId, selectedElementPosition } = useSelectedTemplateDesignerIds();
  const segmentsDraft = useTemplateDesignerStore((state) => state.segmentsDraft);
  const selectedSegment =
    segmentsDraft.find((segment) => segment.id === selectedSegmentId) ?? segmentsDraft[0];
  const selectedElement =
    selectedSegment?.elements.find((element) => element.position === selectedElementPosition) ??
    selectedSegment?.elements[0];

  return {
    segmentsDraft,
    selectedSegment,
    selectedElement,
  };
}

export function useTemplateDesignerVersionHydration() {
  const { selectedVersion } = useSelectedTemplateDesignerData();
  const hydrateVersion = useTemplateDesignerStore((state) => state.hydrateVersion);
  const hydratedVersionKey = useTemplateDesignerStore((state) => state.hydratedVersionKey);
  const selectedVersionDraftKey = getTemplateDesignerVersionDraftKey(selectedVersion);

  return {
    selectedVersion,
    hydrateVersion,
    hydratedVersionKey,
    selectedVersionDraftKey,
    segmentsReady: !!selectedVersionDraftKey && hydratedVersionKey === selectedVersionDraftKey,
  };
}

export function useTemplateDesignerSelectionNormalization() {
  const { selectedTemplateId, selectedVersionId, selectedSegmentId, selectedElementPosition } =
    useSelectedTemplateDesignerIds();
  const [templateUrlState] = useTemplateDesignerUrlState();
  const { templates, versions, selectedTemplate } = useSelectedTemplateDesignerData();
  const segmentsDraft = useTemplateDesignerStore((state) => state.segmentsDraft);
  const { patchTemplateUrlState } = useTemplateDesignerUrlActions();
  const { segmentsReady } = useTemplateDesignerVersionHydration();
  const templatesForSelection = useMemo(() => {
    if (
      !selectedTemplate ||
      templates.some((template) => template.id === selectedTemplate.id) ||
      !templateMatchesCurrentFilters(selectedTemplate, {
        search: templateUrlState.templateSearch,
        status: templateUrlState.templateStatus,
        transactionSet: templateUrlState.templateTransactionSet,
        direction: templateUrlState.templateDirection,
      })
    ) {
      return templates;
    }

    return [selectedTemplate, ...templates];
  }, [
    selectedTemplate,
    templateUrlState.templateDirection,
    templateUrlState.templateSearch,
    templateUrlState.templateStatus,
    templateUrlState.templateTransactionSet,
    templates,
  ]);

  return useCallback(() => {
    const selectionPatch = getTemplateDesignerSelectionPatch({
      templateId: selectedTemplateId,
      versionId: selectedVersionId,
      segmentId: selectedSegmentId,
      elementPosition: selectedElementPosition,
      templates: templatesForSelection,
      versions,
      segments: segmentsDraft,
      segmentsReady,
    });
    if (selectionPatch) patchTemplateUrlState(selectionPatch);
  }, [
    patchTemplateUrlState,
    selectedElementPosition,
    selectedSegmentId,
    selectedTemplateId,
    selectedVersionId,
    segmentsDraft,
    segmentsReady,
    templatesForSelection,
    versions,
  ]);
}

type TemplateDesignerValidationAction = {
  validate: () => void;
  isValidating: boolean;
  canValidate: boolean;
};

const TemplateDesignerValidationContext = createContext<TemplateDesignerValidationAction | null>(
  null,
);

export function TemplateDesignerValidationProvider({ children }: { children: ReactNode }) {
  const { selectedTemplateId, selectedVersionId } = useSelectedTemplateDesignerIds();
  const { hasUnsavedChanges } = useTemplateDesignerDirtyState();
  const setDiagnostics = useTemplateDesignerStore((state) => state.setDiagnostics);

  const { mutate: validateTemplate, isPending: isValidating } = useValidateEDITemplateMutation({
    onSuccess: (response) => {
      setDiagnostics(response.diagnostics);
      toast.success("Template validation complete");
    },
    onError: () => toast.error("Template validation failed"),
  });

  const action = useMemo<TemplateDesignerValidationAction>(
    () => ({
      validate: () =>
        validateTemplate({
          templateId: selectedTemplateId,
          versionId: selectedVersionId,
        }),
      isValidating,
      canValidate: !!selectedTemplateId && !!selectedVersionId && !hasUnsavedChanges,
    }),
    [hasUnsavedChanges, isValidating, selectedTemplateId, selectedVersionId, validateTemplate],
  );

  return (
    <TemplateDesignerValidationContext.Provider value={action}>
      {children}
    </TemplateDesignerValidationContext.Provider>
  );
}

export function useTemplateDesignerValidationAction() {
  const action = useContext(TemplateDesignerValidationContext);
  if (!action) {
    throw new Error(
      "useTemplateDesignerValidationAction must be used within TemplateDesignerValidationProvider",
    );
  }

  return action;
}

export function useCurrentTemplateInvalidation() {
  const { selectedTemplateId, selectedVersionId } = useSelectedTemplateDesignerIds();

  return useInvalidateEDITemplateQueries(selectedTemplateId, selectedVersionId);
}

export function useSelectDiagnostic() {
  const segmentsDraft = useTemplateDesignerStore((state) => state.segmentsDraft);
  const { patchTemplateUrlState } = useTemplateDesignerUrlActions();

  return useCallback(
    (diagnostic: EDIDiagnostic) => {
      const segment = segmentsDraft.find((item) => item.segmentId === diagnostic.segmentId);
      if (!segment) return;
      patchTemplateUrlState({
        segmentId: segment.id,
        ...(diagnostic.elementPosition > 0 ? { elementPosition: diagnostic.elementPosition } : {}),
      });
    },
    [patchTemplateUrlState, segmentsDraft],
  );
}
