import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import type {
  EDIDiagnostic,
  EDITemplate,
  EDITemplateElement,
  EDITemplateScriptLibrary,
  EDITemplateSegment,
  EDITemplateVersion,
} from "@/types/edi";
import {
  AlertTriangleIcon,
  ArchiveIcon,
  CheckCircle2Icon,
  ClipboardCheckIcon,
  CopyPlusIcon,
  FileCode2Icon,
  ListChecksIcon,
  PlusIcon,
  SaveIcon,
  SearchIcon,
} from "lucide-react";
import {
  lazy,
  Suspense,
  useCallback,
  useEffect,
  useMemo,
  useState,
  type Dispatch,
  type SetStateAction,
} from "react";
import { toast } from "sonner";
import {
  cloneSegments,
  diagnosticsForSegment,
  getReadOnlyReason,
  isTemplateVersionEditable,
} from "../utils/edi-designer-utils";
import { useTemplateDesignerUrlState } from "../hooks/use-edi-designer-url-state";
import { useEDITemplateQueries } from "../hooks/use-edi-template-queries";
import {
  useActivateEDITemplateMutation,
  useArchiveEDITemplateMutation,
  useCreateEDITemplateDraftMutation,
  useCreateEDITemplateMutation,
  useInvalidateEDITemplateQueries,
  useSaveEDITemplateMetadataMutation,
  useSaveEDITemplateScriptsMutation,
  useSaveEDITemplateSegmentsMutation,
  useValidateAndCertifyEDITemplateMutation,
  useValidateEDITemplateMutation,
} from "../hooks/use-edi-template-mutations";
import { ElementDesigner } from "../element/element-designer";
import { DesignerPanelSkeleton } from "./designer-workspace-skeleton";
import { ValidationPanel } from "./validation-panel";
import {
  DiagnosticsList,
  InputBlock,
  PanelHeader,
  ReadOnlyBanner,
  SelectBlock,
  VersionStatusBadge,
} from "./designer-shared";

const ScriptLibraryEditor = lazy(() => import("../scripts/script-library-editor"));
const TemplatePreviewPanel = lazy(() => import("./template-preview-panel"));

export default function TemplateDesignerTab() {
  const [templateUrlState, setTemplateUrlState] = useTemplateDesignerUrlState();
  const {
    templateSearch,
    templateStatus,
    templateId: selectedTemplateId,
    versionId: selectedVersionId,
    segmentId: selectedSegmentId,
    elementPosition: selectedElementPosition,
  } = templateUrlState;
  const setTemplateSearch = useCallback(
    (value: string) => void setTemplateUrlState({ templateSearch: value }),
    [setTemplateUrlState],
  );
  const setTemplateStatus = useCallback(
    (value: string) => void setTemplateUrlState({ templateStatus: value }),
    [setTemplateUrlState],
  );
  const setSelectedTemplateId = useCallback(
    (templateId: string) => void setTemplateUrlState({ templateId }),
    [setTemplateUrlState],
  );
  const setSelectedVersionId = useCallback(
    (versionId: string) => void setTemplateUrlState({ versionId }),
    [setTemplateUrlState],
  );
  const setSelectedSegmentId = useCallback(
    (segmentId: string) => void setTemplateUrlState({ segmentId }),
    [setTemplateUrlState],
  );
  const setSelectedElementPosition = useCallback(
    (elementPosition: number) => void setTemplateUrlState({ elementPosition }),
    [setTemplateUrlState],
  );
  const [segmentsDraft, setSegmentsDraft] = useState<EDITemplateSegment[]>([]);
  const [scriptDraft, setScriptDraft] = useState<EDITemplateScriptLibrary[]>([]);
  const [versionNotes, setVersionNotes] = useState("");
  const [x12Version, setX12Version] = useState("004010");
  const [functionalGroupId, setFunctionalGroupId] = useState("SM");
  const [segmentsDirty, setSegmentsDirty] = useState(false);
  const [scriptsDirty, setScriptsDirty] = useState(false);
  const [metadataDirty, setMetadataDirty] = useState(false);
  const [diagnostics, setDiagnostics] = useState<EDIDiagnostic[]>([]);
  const [newTemplate, setNewTemplate] = useState({
    documentTypeId: "",
    name: "",
    description: "",
    x12Version: "004010",
    functionalGroupId: "SM",
    notes: "",
  });

  const templatesQueryString = useMemo(() => {
    const params = new URLSearchParams({
      limit: "100",
      transactionSet: "204",
      direction: "Outbound",
    });
    if (templateSearch.trim()) params.set("search", templateSearch.trim());
    if (templateStatus) params.set("status", templateStatus);
    return `?${params.toString()}`;
  }, [templateSearch, templateStatus]);

  const {
    templatesQuery,
    documentTypesQuery,
    templateQuery,
    versionsQuery,
    versionQuery,
    sourceFieldsQuery,
    partnerFieldsQuery,
  } = useEDITemplateQueries({
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
    if (!selectedTemplateId && templates[0]) {
      setSelectedTemplateId(templates[0].id);
    }
  }, [selectedTemplateId, setSelectedTemplateId, templates]);

  useEffect(() => {
    if (!selectedVersionId && versions[0]) {
      setSelectedVersionId(versions[0].id);
    }
    if (
      selectedVersionId &&
      versions.length > 0 &&
      !versions.some((version) => version.id === selectedVersionId)
    ) {
      setSelectedVersionId(versions[0].id);
    }
  }, [selectedVersionId, setSelectedVersionId, versions]);

  useEffect(() => {
    if (!selectedVersion || segmentsDirty || scriptsDirty || metadataDirty) return;
    setSegmentsDraft(cloneSegments(selectedVersion.segments));
    setScriptDraft(selectedVersion.scriptLibraries.map((library) => ({ ...library })));
    setVersionNotes(selectedVersion.notes ?? "");
    setX12Version(selectedVersion.x12Version);
    setFunctionalGroupId(selectedVersion.functionalGroupId);
    setDiagnostics([]);
  }, [metadataDirty, scriptsDirty, segmentsDirty, selectedVersion]);

  useEffect(() => {
    if (
      segmentsDraft.length > 0 &&
      !segmentsDraft.some((segment) => segment.id === selectedSegmentId)
    ) {
      setSelectedSegmentId(segmentsDraft[0].id);
      setSelectedElementPosition(segmentsDraft[0].elements[0]?.position ?? 0);
    }
  }, [selectedSegmentId, segmentsDraft, setSelectedElementPosition, setSelectedSegmentId]);

  useEffect(() => {
    if (selectedSegment && selectedElementPosition === 0) {
      setSelectedElementPosition(selectedSegment.elements[0]?.position ?? 0);
    }
  }, [selectedElementPosition, selectedSegment, setSelectedElementPosition]);

  const invalidateTemplateQueries = useInvalidateEDITemplateQueries(
    selectedTemplateId,
    selectedVersionId,
  );

  const createTemplateMutation = useCreateEDITemplateMutation({
    onSuccess: async (template) => {
      toast.success("EDI template created");
      setSelectedTemplateId(template.id);
      setSelectedVersionId(template.versions[0]?.id ?? template.activeVersion?.id ?? "");
      setNewTemplate({
        documentTypeId: "",
        name: "",
        description: "",
        x12Version: "004010",
        functionalGroupId: "SM",
        notes: "",
      });
      await invalidateTemplateQueries();
    },
    onError: () => toast.error("Failed to create EDI template"),
  });

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
    updater: (segment: EDITemplateSegment) => EDITemplateSegment,
  ) => {
    if (!isEditable) return;
    setSegmentsDraft((current) =>
      current.map((segment) => (segment.id === segmentId ? updater(segment) : segment)),
    );
    setSegmentsDirty(true);
  };

  const updateElement = (
    position: number,
    updater: (element: EDITemplateElement) => EDITemplateElement,
  ) => {
    if (!selectedSegment || !isEditable) return;
    updateSegment(selectedSegment.id, (segment) => ({
      ...segment,
      elements: segment.elements.map((element) =>
        element.position === position ? updater(element) : element,
      ),
    }));
  };

  const clearDirtyState = () => {
    setSegmentsDirty(false);
    setScriptsDirty(false);
    setMetadataDirty(false);
  };

  return (
    <div className="grid min-h-[calc(100vh-14rem)] grid-cols-[310px_minmax(0,1fr)_380px] gap-3 max-xl:grid-cols-[280px_minmax(0,1fr)] max-lg:grid-cols-1">
      <aside className="flex min-h-0 flex-col rounded-md border bg-background">
        <PanelHeader icon={<FileCode2Icon />} title="Templates" />
        <div className="space-y-3 border-b p-3">
          <div className="flex items-center gap-2">
            <SearchIcon className="size-4 text-muted-foreground" />
            <Input
              value={templateSearch}
              onChange={(event) => setTemplateSearch(event.target.value)}
              placeholder="Search templates"
              className="h-8"
            />
          </div>
          <SelectBlock
            label="Status"
            value={templateStatus}
            onValueChange={setTemplateStatus}
            options={[
              { value: "Draft", label: "Draft" },
              { value: "Certified", label: "Certified" },
              { value: "Active", label: "Active" },
              { value: "Deprecated", label: "Deprecated" },
              { value: "Superseded", label: "Superseded" },
              { value: "Archived", label: "Archived" },
            ]}
            placeholder="All statuses"
          />
        </div>
        <TemplateList
          templates={templates}
          selectedTemplateId={selectedTemplateId}
          onSelect={(templateId) => {
            clearDirtyState();
            setSelectedTemplateId(templateId);
            setSelectedVersionId("");
          }}
        />
        <CreateTemplateForm
          documentTypes={documentTypesQuery.data ?? []}
          draft={newTemplate}
          onChange={setNewTemplate}
          onCreate={() =>
            createTemplateMutation.mutate({
              documentTypeId: newTemplate.documentTypeId,
              name: newTemplate.name,
              description: newTemplate.description,
              direction: "Outbound",
              standard: "X12",
              transactionSet: "204",
              x12Version: newTemplate.x12Version,
              functionalGroupId: newTemplate.functionalGroupId,
              notes: newTemplate.notes,
            })
          }
          isLoading={createTemplateMutation.isPending}
        />
      </aside>

      <main className="flex min-h-0 flex-col rounded-md border bg-background">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b px-3 py-2">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <span className="truncate text-sm font-semibold">
                {selectedTemplate?.name ?? "No template selected"}
              </span>
              {selectedVersion ? <VersionStatusBadge version={selectedVersion} /> : null}
              {hasUnsavedChanges && <Badge variant="warning">Unsaved</Badge>}
            </div>
            <div className="text-xs text-muted-foreground">
              {selectedVersion
                ? `Version ${selectedVersion.versionNumber} / ${selectedVersion.x12Version} / ${segmentsDraft.length} segments`
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
              setSelectedVersionId(versionId);
            }}
            segments={segmentsDraft}
            diagnostics={diagnostics}
            selectedSegmentId={selectedSegment?.id ?? ""}
            onSegmentSelect={(segment) => {
              setSelectedSegmentId(segment.id);
              setSelectedElementPosition(segment.elements[0]?.position ?? 0);
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
                sourceFields={sourceFieldsQuery.data?.results ?? []}
                partnerFields={partnerFieldsQuery.data?.results ?? []}
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
                onSelectDiagnostic={(diagnostic) => {
                  const segment = segmentsDraft.find(
                    (item) => item.segmentId === diagnostic.segmentId,
                  );
                  if (!segment) return;
                  setSelectedSegmentId(segment.id);
                  if (diagnostic.elementPosition > 0)
                    setSelectedElementPosition(diagnostic.elementPosition);
                }}
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
        <DiagnosticsList
          diagnostics={diagnostics}
          onSelect={(diagnostic) => {
            const segment = segmentsDraft.find((item) => item.segmentId === diagnostic.segmentId);
            if (!segment) return;
            setSelectedSegmentId(segment.id);
            if (diagnostic.elementPosition > 0)
              setSelectedElementPosition(diagnostic.elementPosition);
          }}
        />
      </aside>
    </div>
  );
}

function TemplateList({
  templates,
  selectedTemplateId,
  onSelect,
}: {
  templates: EDITemplate[];
  selectedTemplateId: string;
  onSelect: (templateId: string) => void;
}) {
  return (
    <div className="min-h-0 flex-1 overflow-auto">
      {templates.map((template) => (
        <button
          key={template.id}
          type="button"
          onClick={() => onSelect(template.id)}
          className={cn(
            "block w-full border-b px-3 py-2 text-left hover:bg-muted",
            selectedTemplateId === template.id && "bg-muted",
          )}
        >
          <div className="flex items-center justify-between gap-2">
            <span className="truncate text-sm font-medium">{template.name}</span>
            <Badge variant={template.status === "Active" ? "active" : "outline"}>
              {template.status}
            </Badge>
          </div>
          <div className="mt-1 text-xs text-muted-foreground">
            {template.transactionSet} {template.direction} / {template.versions.length} versions
          </div>
        </button>
      ))}
      {templates.length === 0 ? (
        <div className="p-3 text-sm text-muted-foreground">No matching templates.</div>
      ) : null}
    </div>
  );
}

function CreateTemplateForm({
  documentTypes,
  draft,
  onChange,
  onCreate,
  isLoading,
}: {
  documentTypes: { id: string; code: string; name: string; defaultVersion: string }[];
  draft: {
    documentTypeId: string;
    name: string;
    description: string;
    x12Version: string;
    functionalGroupId: string;
    notes: string;
  };
  onChange: Dispatch<
    SetStateAction<{
      documentTypeId: string;
      name: string;
      description: string;
      x12Version: string;
      functionalGroupId: string;
      notes: string;
    }>
  >;
  onCreate: () => void;
  isLoading: boolean;
}) {
  return (
    <div className="space-y-2 border-t p-3">
      <div className="flex items-center gap-2 text-xs font-semibold">
        <PlusIcon className="size-4" />
        New Template
      </div>
      <SelectBlock
        label="Document Type"
        value={draft.documentTypeId}
        onValueChange={(documentTypeId) => {
          const documentType = documentTypes.find((item) => item.id === documentTypeId);
          onChange((current) => ({
            ...current,
            documentTypeId,
            x12Version: documentType?.defaultVersion ?? current.x12Version,
          }));
        }}
        options={documentTypes.map((documentType) => ({
          value: documentType.id,
          label: `${documentType.code} - ${documentType.name}`,
        }))}
      />
      <InputBlock
        label="Name"
        value={draft.name}
        onChange={(name) => onChange((current) => ({ ...current, name }))}
      />
      <InputBlock
        label="Description"
        value={draft.description}
        onChange={(description) => onChange((current) => ({ ...current, description }))}
      />
      <div className="grid grid-cols-2 gap-2">
        <InputBlock
          label="X12 Version"
          value={draft.x12Version}
          onChange={(x12Version) => onChange((current) => ({ ...current, x12Version }))}
        />
        <InputBlock
          label="Group"
          value={draft.functionalGroupId}
          onChange={(functionalGroupId) =>
            onChange((current) => ({ ...current, functionalGroupId }))
          }
        />
      </div>
      <Button
        type="button"
        className="w-full"
        onClick={onCreate}
        isLoading={isLoading}
        disabled={!draft.documentTypeId || !draft.name.trim()}
      >
        <PlusIcon className="size-4" />
        Create Template
      </Button>
    </div>
  );
}

function VersionAndSegmentRail({
  versions,
  selectedVersionId,
  onVersionSelect,
  segments,
  diagnostics,
  selectedSegmentId,
  onSegmentSelect,
}: {
  versions: EDITemplateVersion[];
  selectedVersionId: string;
  onVersionSelect: (versionId: string) => void;
  segments: EDITemplateSegment[];
  diagnostics: EDIDiagnostic[];
  selectedSegmentId: string;
  onSegmentSelect: (segment: EDITemplateSegment) => void;
}) {
  return (
    <div className="grid min-h-0 grid-rows-[180px_minmax(0,1fr)] border-r">
      <div className="min-h-0 overflow-auto border-b">
        <div className="sticky top-0 min-h-10.25 justify-center border-b bg-sidebar px-3 py-2.5 text-left text-sm font-semibold">
          Versions
        </div>
        {versions.map((version) => (
          <button
            key={version.id}
            type="button"
            onClick={() => onVersionSelect(version.id)}
            className={cn(
              "flex w-full items-center justify-between gap-2 border-b px-3 py-2 text-left hover:bg-muted",
              selectedVersionId === version.id && "bg-muted",
            )}
          >
            <span className="font-mono text-xs">v{version.versionNumber}</span>
            <VersionStatusBadge version={version} />
          </button>
        ))}
      </div>
      <div className="min-h-0 overflow-auto">
        <div className="sticky top-0 border-b bg-background px-3 py-2 text-xs font-semibold">
          Outline
        </div>
        {segments.map((segment) => {
          const segmentDiagnostics = diagnosticsForSegment(diagnostics, segment);
          return (
            <button
              key={segment.id}
              type="button"
              onClick={() => onSegmentSelect(segment)}
              className={cn(
                "flex w-full items-center justify-between gap-2 border-b px-3 py-2 text-left hover:bg-muted",
                selectedSegmentId === segment.id && "bg-muted",
              )}
            >
              <span className="min-w-0">
                <span className="block font-mono text-sm font-medium">{segment.segmentId}</span>
                <span className="block truncate text-xs text-muted-foreground">{segment.name}</span>
              </span>
              {segmentDiagnostics.length > 0 ? (
                <Badge
                  variant={
                    segmentDiagnostics.some((item) => item.severity === "Error")
                      ? "inactive"
                      : "warning"
                  }
                >
                  {segmentDiagnostics.length}
                </Badge>
              ) : null}
            </button>
          );
        })}
      </div>
    </div>
  );
}
