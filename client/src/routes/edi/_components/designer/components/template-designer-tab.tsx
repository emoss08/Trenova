import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  TemplateDesignerValidationProvider,
  useCurrentTemplateInvalidation,
  useSelectDiagnostic,
  useSelectedTemplateDesignerData,
  useSelectedTemplateDesignerIds,
  useTemplateDesignerDirtyState,
  useTemplateDesignerSelectionNormalization,
  useTemplateDesignerUrlActions,
  useTemplateDesignerValidationAction,
  useTemplateDesignerVersionHydration,
} from "@/hooks/use-template-designer-state";
import {
  TemplateDesignerStoreProvider,
  useTemplateDesignerStore,
} from "@/stores/template-designer-store";
import {
  AlertTriangleIcon,
  ArchiveIcon,
  CheckCircle2Icon,
  ClipboardCheckIcon,
  CopyPlusIcon,
  ListChecksIcon,
  SaveIcon,
} from "lucide-react";
import { lazy, Suspense, useEffect } from "react";
import { toast } from "sonner";
import { ElementDesigner } from "../element/element-designer";
import {
  useActivateEDITemplateMutation,
  useArchiveEDITemplateMutation,
  useCreateEDITemplateDraftMutation,
  useSaveEDITemplateMetadataMutation,
  useSaveEDITemplateSegmentsMutation,
  useValidateAndCertifyEDITemplateMutation,
} from "../hooks/use-edi-template-mutations";
import { getReadOnlyReason, isTemplateVersionEditable } from "../utils/edi-designer-utils";
import VersionAndSegmentRail from "../versions/version-and-segment-rail";
import {
  DiagnosticsList,
  PanelHeader,
  ReadOnlyBanner,
  VersionStatusBadge,
} from "./designer-shared";
import { DesignerAsideSkeleton, DesignerPanelSkeleton } from "./designer-workspace-skeleton";
import { ValidationPanel } from "./validation-panel";

const ScriptLibraryEditor = lazy(() => import("../scripts/script-library-editor"));
const TemplateDesignerAside = lazy(() => import("./template-designer-aside"));
const TemplatePreviewPanel = lazy(() => import("./template-preview-panel"));

export default function TemplateDesignerTab() {
  return (
    <TemplateDesignerStoreProvider>
      <TemplateDesignerValidationProvider>
        <div className="grid h-full min-h-0 grid-cols-[310px_minmax(0,1fr)_250px] gap-3 overflow-hidden">
          <Suspense fallback={<DesignerAsideSkeleton />}>
            <TemplateDesignerAside />
          </Suspense>
          <main className="flex min-h-0 flex-col overflow-hidden rounded-md border bg-background">
            <TemplateDesignerSelectionSync />
            <TemplateDesignerHeader />
            <TemplateDesignerReadOnlyBanner />
            <TemplateDesignerEditor />
            <TemplateDesignerFooter />
          </main>
          <TemplateDesignerDiagnosticsAside />
        </div>
      </TemplateDesignerValidationProvider>
    </TemplateDesignerStoreProvider>
  );
}

function TemplateDesignerSelectionSync() {
  const { selectedVersion, hydrateVersion, hydratedVersionKey, selectedVersionDraftKey } =
    useTemplateDesignerVersionHydration();
  const normalizeSelection = useTemplateDesignerSelectionNormalization();

  useEffect(() => {
    hydrateVersion(selectedVersion);
  }, [hydrateVersion, hydratedVersionKey, selectedVersion, selectedVersionDraftKey]);

  useEffect(() => {
    normalizeSelection();
  }, [normalizeSelection]);

  return null;
}

function TemplateDesignerHeader() {
  const { selectedTemplateId, selectedVersionId } = useSelectedTemplateDesignerIds();
  const { selectedTemplate, selectedVersion } = useSelectedTemplateDesignerData();
  const metadataDraft = useTemplateDesignerStore((state) => state.metadataDraft);
  const segmentsCount = useTemplateDesignerStore((state) => state.segmentsDraft.length);
  const resetDraftState = useTemplateDesignerStore((state) => state.resetDraftState);
  const clearDirtyState = useTemplateDesignerStore((state) => state.clearDirtyState);
  const setDiagnostics = useTemplateDesignerStore((state) => state.setDiagnostics);
  const { hasUnsavedChanges } = useTemplateDesignerDirtyState();
  const { patchTemplateUrlState } = useTemplateDesignerUrlActions();
  const invalidateTemplateQueries = useCurrentTemplateInvalidation();
  const isEditable = isTemplateVersionEditable(selectedVersion);
  const { validate, isValidating, canValidate } = useTemplateDesignerValidationAction();

  const createDraftMutation = useCreateEDITemplateDraftMutation({
    onSuccess: async (version) => {
      toast.success("Draft version created");
      resetDraftState();
      patchTemplateUrlState({ versionId: version.id });
      await invalidateTemplateQueries();
    },
    onError: () => toast.error("Failed to create draft version"),
  });

  const certifyMutation = useValidateAndCertifyEDITemplateMutation({
    onSuccess: async ({ validation }) => {
      setDiagnostics(validation.diagnostics);
      toast.success("Template version certified");
      clearDirtyState();
      await invalidateTemplateQueries();
    },
    onError: () => toast.error("Certification requires zero validation errors"),
  });

  const activateMutation = useActivateEDITemplateMutation({
    onSuccess: async () => {
      toast.success("Template version activated");
      clearDirtyState();
      await invalidateTemplateQueries();
    },
    onError: () => toast.error("Only certified versions can be activated"),
  });

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 border-b bg-muted/20 px-3 py-2">
      <div className="min-w-0">
        <div className="flex items-center gap-2">
          <span className="truncate text-sm font-semibold">
            {selectedTemplate?.name ?? "No template selected"}
          </span>
          {selectedVersion ? <VersionStatusBadge version={selectedVersion} /> : null}
          {hasUnsavedChanges && <Badge variant="warning">Unsaved</Badge>}
          {!isEditable && selectedVersion ? <Badge variant="outline">Read-only</Badge> : null}
        </div>
        <div className="text-xs text-muted-foreground">
          {selectedVersion
            ? `Version ${selectedVersion.versionNumber} / ${metadataDraft.x12Version} / ${metadataDraft.functionalGroupId} / ${segmentsCount} segments`
            : "Create or select an outbound X12 204 template."}
        </div>
      </div>
      <div className="flex max-w-full flex-wrap items-center gap-2">
        <Button
          type="button"
          variant="outline"
          onClick={() =>
            createDraftMutation.mutate({
              templateId: selectedTemplateId,
              request: {
                sourceVersionId: selectedVersion?.id,
                notes: "Draft cloned for template design changes",
              },
            })
          }
          isLoading={createDraftMutation.isPending}
          disabled={!selectedTemplateId || !selectedVersion || selectedVersion.status === "Draft"}
        >
          <CopyPlusIcon className="size-4" />
          New Draft
        </Button>
        <Button
          type="button"
          variant="outline"
          onClick={validate}
          isLoading={isValidating}
          disabled={!canValidate}
        >
          <ListChecksIcon className="size-4" />
          Validate
        </Button>
        <Button
          type="button"
          variant="outline"
          onClick={() =>
            certifyMutation.mutate({
              templateId: selectedTemplateId,
              versionId: selectedVersionId,
              request: { notes: "Certified from EDI designer" },
            })
          }
          isLoading={certifyMutation.isPending}
          disabled={!isEditable || hasUnsavedChanges}
        >
          <ClipboardCheckIcon className="size-4" />
          Certify
        </Button>
        <Button
          type="button"
          onClick={() =>
            activateMutation.mutate({
              templateId: selectedTemplateId,
              versionId: selectedVersionId,
              request: { notes: "Activated from EDI designer" },
            })
          }
          isLoading={activateMutation.isPending}
          disabled={selectedVersion?.status !== "Certified"}
        >
          <CheckCircle2Icon className="size-4" />
          Activate
        </Button>
      </div>
    </div>
  );
}

function TemplateDesignerReadOnlyBanner() {
  const { selectedVersion } = useSelectedTemplateDesignerData();
  const isEditable = isTemplateVersionEditable(selectedVersion);

  if (isEditable || !selectedVersion) return null;
  return <ReadOnlyBanner reason={getReadOnlyReason(selectedVersion)} />;
}

function TemplateDesignerEditor() {
  return (
    <div className="grid min-h-0 flex-1 grid-cols-[250px_minmax(0,1fr)] overflow-hidden max-md:grid-cols-1">
      <VersionAndSegmentRail />
      <Tabs
        defaultValue="elements"
        className="grid min-h-0 grid-rows-[auto_minmax(0,1fr)] gap-0 overflow-hidden"
      >
        <TemplateDesignerTabBar />
        <TabsContent value="elements" className="m-0 min-h-0 overflow-hidden">
          <ElementDesigner />
        </TabsContent>
        <TabsContent value="scripts" className="m-0 min-h-0 overflow-hidden">
          <Suspense fallback={<DesignerPanelSkeleton />}>
            <ScriptLibraryEditor />
          </Suspense>
        </TabsContent>
        <TabsContent value="validation" className="m-0 min-h-0 overflow-hidden">
          <ValidationPanel />
        </TabsContent>
        <TabsContent value="preview" className="m-0 min-h-0 overflow-hidden">
          <Suspense fallback={<DesignerPanelSkeleton />}>
            <TemplatePreviewPanel />
          </Suspense>
        </TabsContent>
      </Tabs>
    </div>
  );
}

function TemplateDesignerTabBar() {
  const { selectedTemplateId, selectedVersionId } = useSelectedTemplateDesignerIds();
  const { selectedVersion } = useSelectedTemplateDesignerData();
  const metadataDraft = useTemplateDesignerStore((state) => state.metadataDraft);
  const segmentsDraft = useTemplateDesignerStore((state) => state.segmentsDraft);
  const metadataDirty = useTemplateDesignerStore((state) => state.metadataDirty);
  const segmentsDirty = useTemplateDesignerStore((state) => state.segmentsDirty);
  const clearMetadataDirty = useTemplateDesignerStore((state) => state.clearMetadataDirty);
  const clearSegmentsDirty = useTemplateDesignerStore((state) => state.clearSegmentsDirty);
  const invalidateTemplateQueries = useCurrentTemplateInvalidation();
  const isEditable = isTemplateVersionEditable(selectedVersion);

  const saveMetadataMutation = useSaveEDITemplateMetadataMutation({
    onSuccess: async () => {
      toast.success("Version metadata saved");
      clearMetadataDirty();
      await invalidateTemplateQueries();
    },
    onError: () => toast.error("Failed to save version metadata"),
  });

  const saveSegmentsMutation = useSaveEDITemplateSegmentsMutation({
    onSuccess: async () => {
      toast.success("Draft segments saved");
      clearSegmentsDirty();
      await invalidateTemplateQueries();
    },
    onError: () => toast.error("Failed to save draft segments"),
  });

  return (
    <div className="flex flex-wrap items-center justify-between gap-2 border-b bg-sidebar px-1">
      <TabsList variant="underline">
        <TabsTrigger value="elements">Elements</TabsTrigger>
        <TabsTrigger value="scripts">Scripts</TabsTrigger>
        <TabsTrigger value="validation">Validation</TabsTrigger>
        <TabsTrigger value="preview">Preview</TabsTrigger>
      </TabsList>
      <div className="flex flex-wrap items-center gap-2">
        <Button
          type="button"
          variant="outline"
          size="xs"
          onClick={() =>
            saveMetadataMutation.mutate({
              templateId: selectedTemplateId,
              versionId: selectedVersionId,
              request: {
                x12Version: metadataDraft.x12Version,
                functionalGroupId: metadataDraft.functionalGroupId,
                notes: metadataDraft.versionNotes,
                version: selectedVersion?.version,
              },
            })
          }
          isLoading={saveMetadataMutation.isPending}
          disabled={!isEditable || !metadataDirty}
        >
          <SaveIcon className="size-4" />
          Save Metadata
        </Button>
        <Button
          type="button"
          size="xs"
          onClick={() =>
            saveSegmentsMutation.mutate({
              templateId: selectedTemplateId,
              versionId: selectedVersionId,
              request: {
                segments: segmentsDraft,
                version: selectedVersion?.version,
              },
            })
          }
          isLoading={saveSegmentsMutation.isPending}
          disabled={!isEditable || !segmentsDirty}
        >
          <SaveIcon className="size-4" />
          Save Draft
        </Button>
      </div>
    </div>
  );
}

function TemplateDesignerFooter() {
  const { selectedTemplateId, selectedVersionId } = useSelectedTemplateDesignerIds();
  const { selectedVersion } = useSelectedTemplateDesignerData();
  const clearDirtyState = useTemplateDesignerStore((state) => state.clearDirtyState);
  const invalidateTemplateQueries = useCurrentTemplateInvalidation();

  const archiveMutation = useArchiveEDITemplateMutation({
    onSuccess: async () => {
      toast.success("Template version archived");
      clearDirtyState();
      await invalidateTemplateQueries();
    },
    onError: () => toast.error("Failed to archive template version"),
  });

  return (
    <div className="flex flex-wrap items-center justify-between gap-2 border-t px-3 py-2">
      <div className="text-xs text-muted-foreground">
        Draft changes are explicit. Segment, element, and script edits are not sent until Save Draft
        is clicked.
      </div>
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() =>
          archiveMutation.mutate({
            templateId: selectedTemplateId,
            versionId: selectedVersionId,
            request: { notes: "Archived from EDI designer" },
          })
        }
        isLoading={archiveMutation.isPending}
        disabled={
          !selectedVersion || selectedVersion.status === "Active" || selectedVersion.isActive
        }
      >
        <ArchiveIcon className="size-4" />
        Archive Version
      </Button>
    </div>
  );
}

function TemplateDesignerDiagnosticsAside() {
  const diagnostics = useTemplateDesignerStore((state) => state.diagnostics);
  const selectDiagnostic = useSelectDiagnostic();

  return (
    <aside className="flex h-full min-h-0 flex-col overflow-hidden rounded-md border bg-background max-xl:hidden">
      <PanelHeader icon={<AlertTriangleIcon />} title="Diagnostics" />
      <ScrollArea className="min-h-0 flex-1" viewportClassName="min-h-0">
        <DiagnosticsList diagnostics={diagnostics} onSelect={selectDiagnostic} />
      </ScrollArea>
    </aside>
  );
}
