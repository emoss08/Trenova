import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  type ediDiagnosticSchema,
  type ediTemplateElementSchema,
  type ediTemplateScriptLibrarySchema,
  type ediTemplateSegmentSchema,
} from "@/types/edi";
import {
  AlertTriangleIcon,
  ArchiveIcon,
  CheckCircle2Icon,
  ClipboardCheckIcon,
  CopyPlusIcon,
  ListChecksIcon,
  SaveIcon,
} from "lucide-react";
import { lazy, Suspense, useCallback, useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import type { z } from "zod";
import { ElementDesigner } from "../element/element-designer";
import {
  getChangedTemplateUrlStatePatch,
  getTemplateDesignerSelectionPatch,
  type TemplateDesignerUrlStatePatch,
  useTemplateDesignerUrlState,
} from "../hooks/use-edi-designer-url-state";
import {
  useActivateEDITemplateMutation,
  useArchiveEDITemplateMutation,
  useCreateEDITemplateDraftMutation,
  useInvalidateEDITemplateQueries,
  useSaveEDITemplateMetadataMutation,
  useSaveEDITemplateScriptsMutation,
  useSaveEDITemplateSegmentsMutation,
  useValidateAndCertifyEDITemplateMutation,
  useValidateEDITemplateMutation,
} from "../hooks/use-edi-template-mutations";
import { useEDITemplateQueries } from "../hooks/use-edi-template-queries";
import {
  buildEDIDocumentContextQuery,
  cloneSegments,
  getReadOnlyReason,
  isTemplateVersionEditable,
} from "../utils/edi-designer-utils";
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

type TemplateSegment = z.infer<typeof ediTemplateSegmentSchema>;
type TemplateElement = z.infer<typeof ediTemplateElementSchema>;
type TemplateScriptLibrary = z.infer<typeof ediTemplateScriptLibrarySchema>;
type TemplateDiagnostic = z.infer<typeof ediDiagnosticSchema>;

export default function TemplateDesignerTab() {
  const [templateUrlState, setTemplateUrlState] = useTemplateDesignerUrlState();
  const {
    templateSearch,
    templateStatus,
    templateId: selectedTemplateId,
    versionId: selectedVersionId,
    segmentId: selectedSegmentId,
    elementPosition: selectedElementPosition,
    templateTransactionSet,
    templateDirection,
  } = templateUrlState;

  const patchTemplateUrlState = useCallback(
    (patch: TemplateDesignerUrlStatePatch) => {
      const changedPatch = getChangedTemplateUrlStatePatch(templateUrlState, patch);
      if (!changedPatch) return;
      void setTemplateUrlState(changedPatch);
    },
    [setTemplateUrlState, templateUrlState],
  );
  const setSelectedVersionId = useCallback(
    (versionId: string) => patchTemplateUrlState({ versionId }),
    [patchTemplateUrlState],
  );
  const setSelectedElementPosition = useCallback(
    (elementPosition: number) => patchTemplateUrlState({ elementPosition }),
    [patchTemplateUrlState],
  );
  const [segmentsDraft, setSegmentsDraft] = useState<TemplateSegment[]>([]);
  const [scriptDraft, setScriptDraft] = useState<TemplateScriptLibrary[]>([]);
  const [versionNotes, setVersionNotes] = useState("");
  const [x12Version, setX12Version] = useState("004010");
  const [functionalGroupId, setFunctionalGroupId] = useState("SM");
  const [segmentsDirty, setSegmentsDirty] = useState(false);
  const [scriptsDirty, setScriptsDirty] = useState(false);
  const [metadataDirty, setMetadataDirty] = useState(false);
  const [diagnostics, setDiagnostics] = useState<TemplateDiagnostic[]>([]);
  const [hydratedVersionKey, setHydratedVersionKey] = useState("");
  const clearDirtyState = useCallback(() => {
    setSegmentsDirty(false);
    setScriptsDirty(false);
    setMetadataDirty(false);
  }, []);
  const handleTemplateSelect = useCallback(
    (templateId: string) => {
      clearDirtyState();
      patchTemplateUrlState({
        templateId,
        versionId: "",
        segmentId: "",
        elementPosition: 0,
      });
    },
    [clearDirtyState, patchTemplateUrlState],
  );
  const handleTemplateCreated = useCallback(
    (templateId: string, versionId: string) => {
      clearDirtyState();
      patchTemplateUrlState({
        templateId,
        versionId,
        segmentId: "",
        elementPosition: 0,
      });
    },
    [clearDirtyState, patchTemplateUrlState],
  );
  const templatesQueryString = useMemo(() => {
    return buildEDIDocumentContextQuery({
      limit: 100,
      query: templateSearch,
      status: templateStatus,
      transactionSet: templateTransactionSet,
      direction: templateDirection,
    });
  }, [templateDirection, templateSearch, templateStatus, templateTransactionSet]);

  const { templatesQuery, templateQuery, versionsQuery, versionQuery } = useEDITemplateQueries({
    templatesQueryString,
    selectedTemplateId,
    selectedVersionId,
  });

  const templates = useMemo(
    () => templatesQuery.data?.results ?? [],
    [templatesQuery.data?.results],
  );
  const selectedTemplate =
    templateQuery.data ?? templates.find((template) => template.id === selectedTemplateId);
  const versions = useMemo(
    () => versionsQuery.data ?? selectedTemplate?.versions ?? [],
    [selectedTemplate?.versions, versionsQuery.data],
  );
  const selectedVersion =
    versionQuery.data ??
    versions.find((version) => version.id === selectedVersionId) ??
    selectedTemplate?.activeVersion ??
    selectedTemplate?.versions[0];
  const selectedVersionDraftKey = selectedVersion
    ? `${selectedVersion.id}:${selectedVersion.version}`
    : "";
  const isEditable = isTemplateVersionEditable(selectedVersion);
  const hasUnsavedChanges = segmentsDirty || scriptsDirty || metadataDirty;
  const canValidate = !!selectedTemplateId && !!selectedVersionId && !hasUnsavedChanges;
  const readOnlyReason = getReadOnlyReason(selectedVersion);
  const selectedSegment =
    segmentsDraft.find((segment) => segment.id === selectedSegmentId) ?? segmentsDraft[0];
  const selectedElement =
    selectedSegment?.elements.find((element) => element.position === selectedElementPosition) ??
    selectedSegment?.elements[0];

  useEffect(() => {
    if (!selectedVersion || !selectedVersionDraftKey) return;
    if (selectedVersionDraftKey === hydratedVersionKey) return;
    if (segmentsDirty || scriptsDirty || metadataDirty) return;
    setSegmentsDraft(cloneSegments(selectedVersion.segments));
    setScriptDraft(selectedVersion.scriptLibraries.map((library) => ({ ...library })));
    setVersionNotes(selectedVersion.notes ?? "");
    setX12Version(selectedVersion.x12Version);
    setFunctionalGroupId(selectedVersion.functionalGroupId);
    setDiagnostics([]);
    setHydratedVersionKey(selectedVersionDraftKey);
  }, [
    hydratedVersionKey,
    metadataDirty,
    scriptsDirty,
    segmentsDirty,
    selectedVersion,
    selectedVersionDraftKey,
  ]);

  useEffect(() => {
    const selectionPatch = getTemplateDesignerSelectionPatch({
      templateId: selectedTemplateId,
      versionId: selectedVersionId,
      segmentId: selectedSegmentId,
      elementPosition: selectedElementPosition,
      templates,
      versions,
      segments: segmentsDraft,
      segmentsReady:
        !!selectedVersionDraftKey && hydratedVersionKey === selectedVersionDraftKey,
    });
    if (selectionPatch) patchTemplateUrlState(selectionPatch);
  }, [
    hydratedVersionKey,
    patchTemplateUrlState,
    segmentsDraft,
    selectedElementPosition,
    selectedSegmentId,
    selectedTemplateId,
    selectedVersionDraftKey,
    selectedVersionId,
    templates,
    versions,
  ]);

  const invalidateTemplateQueries = useInvalidateEDITemplateQueries(
    selectedTemplateId,
    selectedVersionId,
  );

  const createDraftMutation = useCreateEDITemplateDraftMutation({
    onSuccess: async (version) => {
      toast.success("Draft version created");
      clearDirtyState();
      setSelectedVersionId(version.id);
      await invalidateTemplateQueries();
    },
    onError: () => toast.error("Failed to create draft version"),
  });

  const saveMetadataMutation = useSaveEDITemplateMetadataMutation({
    onSuccess: async () => {
      toast.success("Version metadata saved");
      setMetadataDirty(false);
      await invalidateTemplateQueries();
    },
    onError: () => toast.error("Failed to save version metadata"),
  });

  const saveSegmentsMutation = useSaveEDITemplateSegmentsMutation({
    onSuccess: async () => {
      toast.success("Draft segments saved");
      setSegmentsDirty(false);
      await invalidateTemplateQueries();
    },
    onError: () => toast.error("Failed to save draft segments"),
  });

  const saveScriptsMutation = useSaveEDITemplateScriptsMutation({
    onSuccess: async () => {
      toast.success("Script libraries saved");
      setScriptsDirty(false);
      await invalidateTemplateQueries();
    },
    onError: () => toast.error("Failed to save script libraries"),
  });

  const validateMutation = useValidateEDITemplateMutation({
    onSuccess: (response) => {
      setDiagnostics(response.diagnostics);
      toast.success("Template validation complete");
    },
    onError: () => toast.error("Template validation failed"),
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

  const archiveMutation = useArchiveEDITemplateMutation({
    onSuccess: async () => {
      toast.success("Template version archived");
      clearDirtyState();
      await invalidateTemplateQueries();
    },
    onError: () => toast.error("Failed to archive template version"),
  });

  const updateSegment = (
    segmentId: string,
    updater: (segment: TemplateSegment) => TemplateSegment,
  ) => {
    if (!isEditable) return;
    setSegmentsDraft((current) =>
      current.map((segment) => (segment.id === segmentId ? updater(segment) : segment)),
    );
    setSegmentsDirty(true);
  };

  const updateElement = (
    position: number,
    updater: (element: TemplateElement) => TemplateElement,
  ) => {
    if (!selectedSegment || !isEditable) return;
    updateSegment(selectedSegment.id, (segment) => ({
      ...segment,
      elements: segment.elements.map((element) =>
        element.position === position ? updater(element) : element,
      ),
    }));
  };

  const handleDiagnosticSelect = useCallback(
    (diagnostic: TemplateDiagnostic) => {
      const segment = segmentsDraft.find((item) => item.segmentId === diagnostic.segmentId);
      if (!segment) return;
      patchTemplateUrlState({
        segmentId: segment.id,
        ...(diagnostic.elementPosition > 0
          ? { elementPosition: diagnostic.elementPosition }
          : {}),
      });
    },
    [patchTemplateUrlState, segmentsDraft],
  );

  return (
    <div className="grid min-h-[calc(100vh-14rem)] grid-cols-[310px_minmax(0,1fr)_380px] gap-3 max-xl:grid-cols-[280px_minmax(0,1fr)] max-lg:grid-cols-1">
      <Suspense fallback={<DesignerAsideSkeleton />}>
        <TemplateDesignerAside
          templates={templates}
          selectedTemplateId={selectedTemplateId}
          selectedVersionId={selectedVersionId}
          onSelectTemplate={handleTemplateSelect}
          onTemplateCreated={handleTemplateCreated}
        />
      </Suspense>
      <main className="flex min-h-0 flex-col rounded-md border bg-background">
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
                ? `Version ${selectedVersion.versionNumber} / ${selectedVersion.x12Version} / ${functionalGroupId} / ${segmentsDraft.length} segments`
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
              disabled={
                !selectedTemplateId || !selectedVersion || selectedVersion.status === "Draft"
              }
            >
              <CopyPlusIcon className="size-4" />
              New Draft
            </Button>
            <Button
              type="button"
              variant="outline"
              onClick={() =>
                validateMutation.mutate({
                  templateId: selectedTemplateId,
                  versionId: selectedVersionId,
                })
              }
              isLoading={validateMutation.isPending}
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
        {!isEditable && selectedVersion ? <ReadOnlyBanner reason={readOnlyReason} /> : null}
        <div className="grid min-h-0 flex-1 grid-cols-[280px_minmax(0,1fr)] max-md:grid-cols-1">
          <VersionAndSegmentRail
            versions={versions}
            selectedVersionId={selectedVersionId}
            onVersionSelect={(versionId) => {
              clearDirtyState();
              patchTemplateUrlState({
                versionId,
                segmentId: "",
                elementPosition: 0,
              });
            }}
            segments={segmentsDraft}
            diagnostics={diagnostics}
            selectedSegmentId={selectedSegment?.id ?? ""}
            onSegmentSelect={(segment) => {
              patchTemplateUrlState({
                segmentId: segment.id,
                elementPosition: segment.elements[0]?.position ?? 0,
              });
            }}
          />
          <Tabs defaultValue="elements" className="min-h-0 gap-0">
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
                        x12Version,
                        functionalGroupId,
                        notes: versionNotes,
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
            <TabsContent value="elements" className="min-h-0">
              <ElementDesigner
                version={selectedVersion}
                x12Version={x12Version}
                functionalGroupId={functionalGroupId}
                notes={versionNotes}
                onMetadataChange={(patch) => {
                  if (!isEditable) return;
                  if (patch.x12Version !== undefined) setX12Version(patch.x12Version);
                  if (patch.functionalGroupId !== undefined)
                    setFunctionalGroupId(patch.functionalGroupId);
                  if (patch.notes !== undefined) setVersionNotes(patch.notes);
                  setMetadataDirty(true);
                }}
                segment={selectedSegment}
                element={selectedElement}
                diagnostics={diagnostics}
                isEditable={isEditable}
                onSegmentChange={updateSegment}
                onElementSelect={setSelectedElementPosition}
                onElementChange={updateElement}
              />
            </TabsContent>
            <TabsContent value="scripts" className="min-h-0">
              <Suspense fallback={<DesignerPanelSkeleton />}>
                <ScriptLibraryEditor
                  libraries={scriptDraft}
                  isEditable={isEditable}
                  onChange={(libraries) => {
                    setScriptDraft(libraries);
                    setScriptsDirty(true);
                  }}
                  onSave={() =>
                    saveScriptsMutation.mutate({
                      templateId: selectedTemplateId,
                      versionId: selectedVersionId,
                      request: {
                        scriptLibraries: scriptDraft,
                        version: selectedVersion?.version,
                      },
                    })
                  }
                  isSaving={saveScriptsMutation.isPending}
                />
              </Suspense>
            </TabsContent>
            <TabsContent value="validation" className="min-h-0">
              <ValidationPanel
                diagnostics={diagnostics}
                onSelectDiagnostic={handleDiagnosticSelect}
                onValidate={() =>
                  validateMutation.mutate({
                    templateId: selectedTemplateId,
                    versionId: selectedVersionId,
                  })
                }
                isLoading={validateMutation.isPending}
                disabled={!canValidate}
              />
            </TabsContent>
            <TabsContent value="preview" className="min-h-0">
              <Suspense fallback={<DesignerPanelSkeleton />}>
                <TemplatePreviewPanel />
              </Suspense>
            </TabsContent>
          </Tabs>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-2 border-t px-3 py-2">
          <div className="text-xs text-muted-foreground">
            Draft changes are explicit. Segment, element, and script edits are not sent until Save
            Draft is clicked.
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
      </main>

      <aside className="flex min-h-0 flex-col rounded-md border bg-background max-xl:hidden">
        <PanelHeader icon={<AlertTriangleIcon />} title="Diagnostics" />
        <DiagnosticsList diagnostics={diagnostics} onSelect={handleDiagnosticSelect} />
      </aside>
    </div>
  );
}
