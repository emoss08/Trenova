import { darkTheme, lightTheme } from "@/components/formula-editor/editor-theme";
import { useTheme } from "@/components/theme-provider";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import type {
  EDIDiagnostic,
  EDIDocumentPreview,
  EDIMessage,
  EDIPartnerDocumentProfile,
  EDIPartnerSettingField,
  EDISourceContextField,
  EDITemplate,
  EDITemplateElement,
  EDITemplateElementBaseSource,
  EDITemplateScriptLibrary,
  EDITemplateSegment,
  EDITemplateTransformStep,
  EDITemplateVersion,
  EDIX12EnvelopeSettings,
  UpsertEDIPartnerDocumentProfileRequest,
} from "@/types/edi";
import { json } from "@codemirror/lang-json";
import { EditorView } from "@codemirror/view";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import CodeMirror from "@uiw/react-codemirror";
import {
  AlertTriangleIcon,
  ArchiveIcon,
  CheckCircle2Icon,
  ClipboardCheckIcon,
  CopyPlusIcon,
  DatabaseIcon,
  FileCode2Icon,
  FilterIcon,
  Layers3Icon,
  ListChecksIcon,
  PlayIcon,
  PlusIcon,
  RefreshCwIcon,
  SaveIcon,
  SearchIcon,
  ShieldCheckIcon,
  ShuffleIcon,
  Trash2Icon,
} from "lucide-react";
import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type Dispatch,
  type ReactNode,
  type SetStateAction,
} from "react";
import { toast } from "sonner";
import {
  buildConditionString,
  cloneSegments,
  createTransformStep,
  diagnosticKey,
  diagnosticsForElement,
  diagnosticsForSegment,
  getReadOnlyReason,
  getTransformOperationDefinition,
  insertPathReference,
  isTemplateVersionEditable,
  parseConditionString,
  transformOperationDefinitions,
  type ConditionDraft,
} from "./edi-designer-utils";
import { formatUnix } from "./edi-display-utils";
import {
  getEDIScriptPresetsByCategory,
  insertScriptPresetCode,
  type EDIScriptPreset,
} from "./edi-script-presets";

const defaultEnvelope: EDIX12EnvelopeSettings = {
  interchangeSenderId: "TRENOVA",
  interchangeReceiverId: "PARTNER",
  applicationSenderCode: "TRENOVA",
  applicationReceiverCode: "PARTNER",
  interchangeUsageIndicator: "T",
  elementSeparator: "*",
  segmentTerminator: "~",
  componentSeparator: ">",
  repetitionSeparator: "^",
};

const defaultProfileDraft: UpsertEDIPartnerDocumentProfileRequest = {
  ediPartnerId: "",
  name: "Outbound X12 204",
  status: "Active",
  functionalGroupId: "SM",
  envelope: defaultEnvelope,
  acknowledgment: {
    expected: false,
    type: "None",
    slaInMinutes: 0,
    missingAckSeverity: "Warning",
  },
  validationMode: "Strict",
  partnerSettings: {},
};

export function DesignerWorkspace() {
  return (
    <Tabs defaultValue="templates" className="min-h-[calc(100vh-11rem)] gap-3">
      <TabsList className="grid w-fit grid-cols-2">
        <TabsTrigger value="templates">
          <Layers3Icon data-icon="inline-start" />
          Templates
        </TabsTrigger>
        <TabsTrigger value="documents">
          <ArchiveIcon data-icon="inline-start" />
          Document Preview & Archive
        </TabsTrigger>
      </TabsList>
      <TabsContent value="templates" className="min-h-0">
        <TemplateDesignerTab />
      </TabsContent>
      <TabsContent value="documents" className="min-h-0">
        <DocumentPreviewArchiveTab />
      </TabsContent>
    </Tabs>
  );
}

function TemplateDesignerTab() {
  const queryClient = useQueryClient();
  const [templateSearch, setTemplateSearch] = useState("");
  const [templateStatus, setTemplateStatus] = useState("");
  const [selectedTemplateId, setSelectedTemplateId] = useState("");
  const [selectedVersionId, setSelectedVersionId] = useState("");
  const [selectedSegmentId, setSelectedSegmentId] = useState("");
  const [selectedElementPosition, setSelectedElementPosition] = useState(0);
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

  const templatesQuery = useQuery(queries.edi.templates(templatesQueryString));
  const documentTypesQuery = useQuery(queries.edi.documentTypes());
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
  const sourceFieldsQuery = useQuery(
    queries.edi.sourceContextFields(
      "?limit=100&status=Active&transactionSet=204&direction=Outbound",
    ),
  );
  const partnerFieldsQuery = useQuery(
    queries.edi.partnerSettingFields(
      "?limit=100&status=Active&transactionSet=204&direction=Outbound",
    ),
  );

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
  }, [selectedTemplateId, templates]);

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
  }, [selectedVersionId, versions]);

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
  }, [selectedSegmentId, segmentsDraft]);

  useEffect(() => {
    if (selectedSegment && selectedElementPosition === 0) {
      setSelectedElementPosition(selectedSegment.elements[0]?.position ?? 0);
    }
  }, [selectedElementPosition, selectedSegment]);

  const invalidateTemplateQueries = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: queries.edi.templates._def });
    if (selectedTemplateId) {
      await queryClient.invalidateQueries({
        queryKey: queries.edi.template(selectedTemplateId).queryKey,
      });
      await queryClient.invalidateQueries({
        queryKey: queries.edi.templateVersions(selectedTemplateId).queryKey,
      });
    }
    if (selectedTemplateId && selectedVersionId) {
      await queryClient.invalidateQueries({
        queryKey: queries.edi.templateVersion(selectedTemplateId, selectedVersionId).queryKey,
      });
    }
  }, [queryClient, selectedTemplateId, selectedVersionId]);

  const createTemplateMutation = useApiMutation({
    mutationFn: async () =>
      apiService.ediService.createTemplate({
        documentTypeId: newTemplate.documentTypeId,
        name: newTemplate.name,
        description: newTemplate.description,
        direction: "Outbound",
        standard: "X12",
        transactionSet: "204",
        x12Version: newTemplate.x12Version,
        functionalGroupId: newTemplate.functionalGroupId,
        notes: newTemplate.notes,
      }),
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

  const createDraftMutation = useApiMutation({
    mutationFn: async () =>
      apiService.ediService.createTemplateDraft(selectedTemplateId, {
        sourceVersionId: selectedVersion?.id,
        notes: "Draft cloned for template design changes",
      }),
    onSuccess: async (version) => {
      toast.success("Draft version created");
      clearDirtyState();
      setSelectedVersionId(version.id);
      await invalidateTemplateQueries();
    },
    onError: () => toast.error("Failed to create draft version"),
  });

  const saveMetadataMutation = useApiMutation({
    mutationFn: async () =>
      apiService.ediService.updateTemplateVersion(selectedTemplateId, selectedVersionId, {
        x12Version,
        functionalGroupId,
        notes: versionNotes,
        version: selectedVersion?.version,
      }),
    onSuccess: async () => {
      toast.success("Version metadata saved");
      setMetadataDirty(false);
      await invalidateTemplateQueries();
    },
    onError: () => toast.error("Failed to save version metadata"),
  });

  const saveSegmentsMutation = useApiMutation({
    mutationFn: async () =>
      apiService.ediService.replaceTemplateSegments(selectedTemplateId, selectedVersionId, {
        segments: segmentsDraft,
        version: selectedVersion?.version,
      }),
    onSuccess: async () => {
      toast.success("Draft segments saved");
      setSegmentsDirty(false);
      await invalidateTemplateQueries();
    },
    onError: () => toast.error("Failed to save draft segments"),
  });

  const saveScriptsMutation = useApiMutation({
    mutationFn: async () =>
      apiService.ediService.replaceTemplateScriptLibraries(selectedTemplateId, selectedVersionId, {
        scriptLibraries: scriptDraft,
        version: selectedVersion?.version,
      }),
    onSuccess: async () => {
      toast.success("Script libraries saved");
      setScriptsDirty(false);
      await invalidateTemplateQueries();
    },
    onError: () => toast.error("Failed to save script libraries"),
  });

  const validateMutation = useApiMutation({
    mutationFn: async () =>
      apiService.ediService.validateTemplateVersion(selectedTemplateId, selectedVersionId),
    onSuccess: (response) => {
      setDiagnostics(response.diagnostics);
      toast.success("Template validation complete");
    },
    onError: () => toast.error("Template validation failed"),
  });

  const certifyMutation = useApiMutation({
    mutationFn: async () => {
      const response = await apiService.ediService.validateTemplateVersion(
        selectedTemplateId,
        selectedVersionId,
      );
      setDiagnostics(response.diagnostics);
      if (response.diagnostics.some((diagnostic) => diagnostic.severity === "Error")) {
        throw new Error("Template has validation errors");
      }
      return apiService.ediService.certifyTemplateVersion(selectedTemplateId, selectedVersionId, {
        notes: "Certified from EDI designer",
      });
    },
    onSuccess: async () => {
      toast.success("Template version certified");
      clearDirtyState();
      await invalidateTemplateQueries();
    },
    onError: () => toast.error("Certification requires zero validation errors"),
  });

  const activateMutation = useApiMutation({
    mutationFn: async () =>
      apiService.ediService.activateTemplateVersion(selectedTemplateId, selectedVersionId, {
        notes: "Activated from EDI designer",
      }),
    onSuccess: async () => {
      toast.success("Template version activated");
      clearDirtyState();
      await invalidateTemplateQueries();
    },
    onError: () => toast.error("Only certified versions can be activated"),
  });

  const archiveMutation = useApiMutation({
    mutationFn: async () =>
      apiService.ediService.archiveTemplateVersion(selectedTemplateId, selectedVersionId, {
        notes: "Archived from EDI designer",
      }),
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
    <div className="grid min-h-[calc(100vh-14rem)] grid-cols-[310px_minmax(0,1fr)_380px] gap-3">
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
          onCreate={() => createTemplateMutation.mutate(undefined)}
          isLoading={createTemplateMutation.isPending}
        />
      </aside>

      <main className="flex min-h-0 flex-col rounded-md border bg-background">
        <div className="flex items-center justify-between gap-3 border-b px-3 py-2">
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
          <div className="flex shrink-0 items-center gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => createDraftMutation.mutate(undefined)}
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
              onClick={() => validateMutation.mutate(undefined)}
              isLoading={validateMutation.isPending}
              disabled={!canValidate}
            >
              <ListChecksIcon className="size-4" />
              Validate
            </Button>
            <Button
              type="button"
              variant="outline"
              onClick={() => certifyMutation.mutate(undefined)}
              isLoading={certifyMutation.isPending}
              disabled={!isEditable || hasUnsavedChanges}
            >
              <ClipboardCheckIcon className="size-4" />
              Certify
            </Button>
            <Button
              type="button"
              onClick={() => activateMutation.mutate(undefined)}
              isLoading={activateMutation.isPending}
              disabled={selectedVersion?.status !== "Certified"}
            >
              <CheckCircle2Icon className="size-4" />
              Activate
            </Button>
          </div>
        </div>
        {!isEditable && selectedVersion ? <ReadOnlyBanner reason={readOnlyReason} /> : null}
        <div className="grid min-h-0 flex-1 grid-cols-[280px_minmax(0,1fr)]">
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
            <div className="flex items-center justify-between border-b px-1 bg-sidebar">
              <TabsList variant="underline">
                <TabsTrigger value="elements">Elements</TabsTrigger>
                <TabsTrigger value="scripts">Scripts</TabsTrigger>
                <TabsTrigger value="validation">Validation</TabsTrigger>
                <TabsTrigger value="preview">Preview</TabsTrigger>
              </TabsList>
              <div className="flex items-center gap-2">
                <Button
                  type="button"
                  variant="outline"
                  size="xs"
                  onClick={() => saveMetadataMutation.mutate(undefined)}
                  isLoading={saveMetadataMutation.isPending}
                  disabled={!isEditable || !metadataDirty}
                >
                  <SaveIcon className="size-4" />
                  Save Metadata
                </Button>
                <Button
                  type="button"
                  size="xs"
                  onClick={() => saveSegmentsMutation.mutate(undefined)}
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
              <ScriptLibraryEditor
                libraries={scriptDraft}
                isEditable={isEditable}
                onChange={(libraries) => {
                  setScriptDraft(libraries);
                  setScriptsDirty(true);
                }}
                onSave={() => saveScriptsMutation.mutate(undefined)}
                isSaving={saveScriptsMutation.isPending}
              />
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
                onValidate={() => validateMutation.mutate(undefined)}
                isLoading={validateMutation.isPending}
                disabled={!canValidate}
              />
            </TabsContent>
            <TabsContent value="preview" className="min-h-0">
              <TemplatePreviewPanel />
            </TabsContent>
          </Tabs>
        </div>
        <div className="flex items-center justify-between border-t px-3 py-2">
          <div className="text-xs text-muted-foreground">
            Draft changes are explicit. Segment, element, and script edits are not sent until Save
            Draft is clicked.
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => archiveMutation.mutate(undefined)}
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

      <aside className="flex min-h-0 flex-col rounded-md border bg-background">
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
        <div className="sticky top-0 border-b bg-sidebar px-3 py-2.5 text-sm font-semibold min-h-10.25 text-left justify-center">
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

function ElementDesigner({
  version,
  x12Version,
  functionalGroupId,
  notes,
  onMetadataChange,
  segment,
  element,
  diagnostics,
  isEditable,
  sourceFields,
  partnerFields,
  onSegmentChange,
  onElementSelect,
  onElementChange,
}: {
  version?: EDITemplateVersion;
  x12Version: string;
  functionalGroupId: string;
  notes: string;
  onMetadataChange: (patch: {
    x12Version?: string;
    functionalGroupId?: string;
    notes?: string;
  }) => void;
  segment?: EDITemplateSegment;
  element?: EDITemplateElement;
  diagnostics: EDIDiagnostic[];
  isEditable: boolean;
  sourceFields: EDISourceContextField[];
  partnerFields: EDIPartnerSettingField[];
  onSegmentChange: (
    segmentId: string,
    updater: (segment: EDITemplateSegment) => EDITemplateSegment,
  ) => void;
  onElementSelect: (position: number) => void;
  onElementChange: (
    position: number,
    updater: (element: EDITemplateElement) => EDITemplateElement,
  ) => void;
}) {
  if (!version || !segment) {
    return (
      <div className="p-4 text-sm text-muted-foreground">Select a template version to edit.</div>
    );
  }

  return (
    <div className="grid min-h-0 grid-rows-[auto_minmax(0,1fr)]">
      <div className="grid grid-cols-4 gap-2 border-b p-3">
        <InputBlock
          label="X12 Version"
          value={x12Version}
          onChange={(value) => onMetadataChange({ x12Version: value })}
          disabled={!isEditable}
        />
        <InputBlock
          label="Functional Group"
          value={functionalGroupId}
          onChange={(value) => onMetadataChange({ functionalGroupId: value })}
          disabled={!isEditable}
        />
        <InputBlock
          label="Notes"
          value={notes}
          onChange={(value) => onMetadataChange({ notes: value })}
          disabled={!isEditable}
        />
        <InputBlock
          label="Segment Condition"
          value={segment.condition ?? ""}
          onChange={(condition) =>
            onSegmentChange(segment.id, (current) => ({ ...current, condition }))
          }
          disabled={!isEditable}
        />
      </div>
      <div className="grid min-h-0 grid-cols-[minmax(0,1fr)_360px]">
        <div className="min-h-0 overflow-auto p-3">
          <div className="mb-3 flex items-center gap-2">
            <Badge variant={segment.required ? "active" : "outline"}>{segment.segmentId}</Badge>
            <div>
              <div className="text-sm font-semibold">{segment.name}</div>
              <div className="text-xs text-muted-foreground">
                Sequence {segment.sequence}
                {segment.repeatPath ? ` / repeats ${segment.repeatPath}` : ""}
              </div>
            </div>
          </div>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-14">Pos</TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Source</TableHead>
                <TableHead>Path / Value</TableHead>
                <TableHead className="w-20">Issues</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {segment.elements.map((item) => {
                const itemDiagnostics = diagnosticsForElement(diagnostics, segment, item);
                return (
                  <TableRow
                    key={`${segment.id}-${item.position}`}
                    onClick={() => onElementSelect(item.position)}
                    className={cn(
                      "cursor-pointer",
                      element?.position === item.position && "bg-muted",
                    )}
                  >
                    <TableCell className="font-mono">{item.position}</TableCell>
                    <TableCell>{item.name}</TableCell>
                    <TableCell>
                      <Badge variant={item.validation.required ? "warning" : "outline"}>
                        {item.source}
                      </Badge>
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {templateElementSourceLabel(item)}
                    </TableCell>
                    <TableCell>
                      {itemDiagnostics.length > 0 ? (
                        <Badge
                          variant={
                            itemDiagnostics.some((diagnostic) => diagnostic.severity === "Error")
                              ? "inactive"
                              : "warning"
                          }
                        >
                          {itemDiagnostics.length}
                        </Badge>
                      ) : null}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
        <ElementInspector
          segment={segment}
          element={element}
          isEditable={isEditable}
          sourceFields={sourceFields}
          partnerFields={partnerFields}
          onChange={onElementChange}
        />
      </div>
    </div>
  );
}

function ElementInspector({
  segment,
  element,
  isEditable,
  sourceFields,
  partnerFields,
  onChange,
}: {
  segment: EDITemplateSegment;
  element?: EDITemplateElement;
  isEditable: boolean;
  sourceFields: EDISourceContextField[];
  partnerFields: EDIPartnerSettingField[];
  onChange: (
    position: number,
    updater: (element: EDITemplateElement) => EDITemplateElement,
  ) => void;
}) {
  if (!element) {
    return <div className="border-l p-4 text-sm text-muted-foreground">Select an element.</div>;
  }
  const update = (patch: Partial<EDITemplateElement>) =>
    onChange(element.position, (current) => ({ ...current, ...patch }));

  return (
    <div className="min-h-0 space-y-3 overflow-auto border-l p-3">
      <div>
        <div className="text-sm font-semibold">
          {segment.segmentId}
          {element.position.toString().padStart(2, "0")} {element.name}
        </div>
        <div className="text-xs text-muted-foreground">Element source and validation rules</div>
      </div>
      <SelectBlock
        label="Source"
        value={element.source}
        onValueChange={(source) => update({ source: source as EDITemplateElement["source"] })}
        disabled={!isEditable}
        options={[
          { value: "constant", label: "Constant" },
          { value: "fieldPath", label: "Field Path" },
          { value: "partnerSetting", label: "Partner Setting" },
          { value: "runtime", label: "Runtime" },
          { value: "repeat", label: "Repeat" },
          { value: "mapping", label: "Mapping" },
          { value: "transform", label: "Transform" },
          { value: "starlark", label: "Starlark" },
        ]}
      />
      <SourceEditor
        element={element}
        isEditable={isEditable}
        sourceFields={sourceFields}
        partnerFields={partnerFields}
        onChange={update}
      />
      <ConditionEditor
        condition={element.condition ?? ""}
        disabled={!isEditable}
        onChange={(condition) => update({ condition })}
      />
      <div className="grid grid-cols-2 gap-2">
        <InputBlock
          label="Default"
          value={element.default ?? ""}
          onChange={(value) => update({ default: value })}
          disabled={!isEditable}
        />
        <InputBlock
          label="Max Length"
          value={String(element.validation.maxLength || "")}
          onChange={(value) =>
            update({
              validation: { ...element.validation, maxLength: Number(value) || 0 },
            })
          }
          disabled={!isEditable}
        />
      </div>
      <div className="flex items-center justify-between rounded-md border p-2">
        <div>
          <div className="text-xs font-medium">Required</div>
          <div className="text-xs text-muted-foreground">Backend validation rule</div>
        </div>
        <Switch
          checked={element.validation.required}
          disabled={!isEditable}
          onCheckedChange={(required) =>
            update({ validation: { ...element.validation, required } })
          }
        />
      </div>
      <TextareaBlock
        label="Implementation Guide Note"
        value={element.implementationGuideNote ?? ""}
        onChange={(value) => update({ implementationGuideNote: value })}
        disabled={!isEditable}
      />
    </div>
  );
}

function SourceEditor({
  element,
  isEditable,
  sourceFields,
  partnerFields,
  onChange,
}: {
  element: EDITemplateElement;
  isEditable: boolean;
  sourceFields: EDISourceContextField[];
  partnerFields: EDIPartnerSettingField[];
  onChange: (patch: Partial<EDITemplateElement>) => void;
}) {
  if (element.source === "constant") {
    return (
      <InputBlock
        label="Value"
        value={element.value ?? ""}
        onChange={(value) => onChange({ value })}
        disabled={!isEditable}
      />
    );
  }
  if (element.source === "fieldPath") {
    return (
      <PathField
        label="Field Path"
        value={element.fieldPath ?? ""}
        onChange={(fieldPath) => onChange({ fieldPath })}
        fields={sourceFields}
        disabled={!isEditable}
      />
    );
  }
  if (element.source === "partnerSetting") {
    return (
      <PartnerPathField
        label="Partner Setting"
        value={element.partnerSettingPath ?? ""}
        onChange={(partnerSettingPath) => onChange({ partnerSettingPath })}
        fields={partnerFields}
        disabled={!isEditable}
      />
    );
  }
  if (element.source === "runtime") {
    return (
      <InputBlock
        label="Runtime Key"
        value={element.runtimeKey ?? ""}
        onChange={(runtimeKey) => onChange({ runtimeKey })}
        disabled={!isEditable}
      />
    );
  }
  if (element.source === "repeat") {
    return (
      <PathField
        label="Repeat Path"
        value={element.repeatPath ?? ""}
        onChange={(repeatPath) => onChange({ repeatPath })}
        fields={sourceFields.filter((field) => field.repeated || field.sourceKind === "repeat")}
        disabled={!isEditable}
      />
    );
  }
  if (element.source === "mapping") {
    return (
      <div className="space-y-2">
        <SelectBlock
          label="Mapping Entity"
          value={element.mappingEntityType ?? ""}
          onValueChange={(mappingEntityType) =>
            onChange({
              mappingEntityType: mappingEntityType as EDITemplateElement["mappingEntityType"],
            })
          }
          disabled={!isEditable}
          options={[
            { value: "Customer", label: "Customer" },
            { value: "ServiceType", label: "Service Type" },
            { value: "ShipmentType", label: "Shipment Type" },
            { value: "FormulaTemplate", label: "Formula Template" },
            { value: "Location", label: "Location" },
            { value: "Commodity", label: "Commodity" },
            { value: "AccessorialCharge", label: "Accessorial Charge" },
          ]}
        />
        <PathField
          label="Mapping Source Path"
          value={element.mappingSourcePath ?? ""}
          onChange={(mappingSourcePath) => onChange({ mappingSourcePath })}
          fields={sourceFields}
          disabled={!isEditable}
        />
      </div>
    );
  }
  if (element.source === "transform") {
    return (
      <TransformPipelineEditor
        element={element}
        disabled={!isEditable}
        sourceFields={sourceFields}
        partnerFields={partnerFields}
        onChange={onChange}
      />
    );
  }
  const starlarkPresets = [
    ...getEDIScriptPresetsByCategory("elementValue"),
    ...getEDIScriptPresetsByCategory("repeatItem"),
  ];
  const applyStarlarkPreset = (preset: EDIScriptPreset) => {
    const patch: Partial<EDITemplateElement> = {
      starlarkScript: insertScriptPresetCode(element.starlarkScript ?? "", preset),
    };
    if (preset.recommendedFunctionName && !element.starlarkFunction?.trim()) {
      patch.starlarkFunction = preset.recommendedFunctionName;
    }
    onChange(patch);
  };

  return (
    <div className="space-y-2">
      <InputBlock
        label="Function Name"
        value={element.starlarkFunction ?? ""}
        onChange={(starlarkFunction) => onChange({ starlarkFunction })}
        disabled={!isEditable}
      />
      <TextareaBlock
        label="Inline Script"
        value={element.starlarkScript ?? ""}
        onChange={(starlarkScript) => onChange({ starlarkScript })}
        disabled={!isEditable}
      />
      <ScriptPresetPicker
        title="Presets"
        presets={starlarkPresets}
        disabled={!isEditable}
        onApply={applyStarlarkPreset}
      />
    </div>
  );
}

function TransformPipelineEditor({
  element,
  disabled,
  sourceFields,
  partnerFields,
  onChange,
}: {
  element: EDITemplateElement;
  disabled: boolean;
  sourceFields: EDISourceContextField[];
  partnerFields: EDIPartnerSettingField[];
  onChange: (patch: Partial<EDITemplateElement>) => void;
}) {
  const baseSource = element.baseSource ?? { source: "fieldPath" as const, fieldPath: "" };
  const updateBase = (patch: Partial<EDITemplateElementBaseSource>) =>
    onChange({ baseSource: { ...baseSource, ...patch } });
  const updatePipeline = (transformPipeline: EDITemplateTransformStep[]) =>
    onChange({ transformPipeline });
  return (
    <div className="space-y-3 rounded-md border p-2">
      <div className="flex items-center gap-2 text-xs font-semibold">
        <ShuffleIcon className="size-4" />
        Transform Pipeline
      </div>
      <SelectBlock
        label="Base Source"
        value={baseSource.source}
        onValueChange={(source) =>
          updateBase({ source: source as EDITemplateElementBaseSource["source"] })
        }
        disabled={disabled}
        options={[
          { value: "constant", label: "Constant" },
          { value: "fieldPath", label: "Field Path" },
          { value: "partnerSetting", label: "Partner Setting" },
          { value: "runtime", label: "Runtime" },
          { value: "repeat", label: "Repeat" },
          { value: "mapping", label: "Mapping" },
        ]}
      />
      <BaseSourceValueEditor
        source={baseSource}
        disabled={disabled}
        sourceFields={sourceFields}
        partnerFields={partnerFields}
        onChange={updateBase}
      />
      <div className="space-y-2">
        {element.transformPipeline.map((step, index) => (
          <TransformStepEditor
            key={`${step.operation}-${index}`}
            step={step}
            index={index}
            disabled={disabled}
            sourceFields={sourceFields}
            partnerFields={partnerFields}
            onMove={(direction) => {
              const next = [...element.transformPipeline];
              const target = index + direction;
              if (target < 0 || target >= next.length) return;
              [next[index], next[target]] = [next[target], next[index]];
              updatePipeline(next);
            }}
            onRemove={() =>
              updatePipeline(
                element.transformPipeline.filter((_, itemIndex) => itemIndex !== index),
              )
            }
            onChange={(updated) =>
              updatePipeline(
                element.transformPipeline.map((item, itemIndex) =>
                  itemIndex === index ? updated : item,
                ),
              )
            }
          />
        ))}
      </div>
      <SelectBlock
        label="Add Operation"
        value=""
        onValueChange={(operation) => {
          if (!operation) return;
          updatePipeline([...element.transformPipeline, createTransformStep(operation)]);
        }}
        disabled={disabled}
        placeholder="Select operation"
        options={transformOperationDefinitions.map((definition) => ({
          value: definition.operation,
          label: definition.label,
        }))}
      />
    </div>
  );
}

function BaseSourceValueEditor({
  source,
  disabled,
  sourceFields,
  partnerFields,
  onChange,
}: {
  source: EDITemplateElementBaseSource;
  disabled: boolean;
  sourceFields: EDISourceContextField[];
  partnerFields: EDIPartnerSettingField[];
  onChange: (patch: Partial<EDITemplateElementBaseSource>) => void;
}) {
  if (source.source === "partnerSetting") {
    return (
      <PartnerPathField
        label="Base Partner Setting"
        value={source.partnerSettingPath ?? ""}
        onChange={(partnerSettingPath) => onChange({ partnerSettingPath })}
        fields={partnerFields}
        disabled={disabled}
      />
    );
  }
  if (source.source === "fieldPath" || source.source === "repeat" || source.source === "mapping") {
    return (
      <PathField
        label="Base Path"
        value={source.fieldPath ?? source.repeatPath ?? source.mappingSourcePath ?? ""}
        onChange={(value) => {
          if (source.source === "repeat") onChange({ repeatPath: value });
          else if (source.source === "mapping") onChange({ mappingSourcePath: value });
          else onChange({ fieldPath: value });
        }}
        fields={sourceFields}
        disabled={disabled}
      />
    );
  }
  if (source.source === "runtime") {
    return (
      <InputBlock
        label="Base Runtime Key"
        value={source.runtimeKey ?? ""}
        onChange={(runtimeKey) => onChange({ runtimeKey })}
        disabled={disabled}
      />
    );
  }
  return (
    <InputBlock
      label="Base Value"
      value={source.value ?? ""}
      onChange={(value) => onChange({ value })}
      disabled={disabled}
    />
  );
}

function TransformStepEditor({
  step,
  index,
  disabled,
  sourceFields,
  partnerFields,
  onChange,
  onMove,
  onRemove,
}: {
  step: EDITemplateTransformStep;
  index: number;
  disabled: boolean;
  sourceFields: EDISourceContextField[];
  partnerFields: EDIPartnerSettingField[];
  onChange: (step: EDITemplateTransformStep) => void;
  onMove: (direction: -1 | 1) => void;
  onRemove: () => void;
}) {
  const definition = getTransformOperationDefinition(step.operation);
  const setArg = (key: string, value: unknown) =>
    onChange({ ...step, arguments: { ...step.arguments, [key]: value } });
  return (
    <div className="space-y-2 rounded-md border bg-muted/20 p-2">
      <div className="flex items-center justify-between gap-2">
        <div>
          <div className="text-xs font-semibold">
            {index + 1}. {definition?.label ?? step.operation}
          </div>
          <div className="text-xs text-muted-foreground">{definition?.description}</div>
        </div>
        <div className="flex items-center gap-1">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            disabled={disabled}
            onClick={() => onMove(-1)}
          >
            Up
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            disabled={disabled}
            onClick={() => onMove(1)}
          >
            Down
          </Button>
          <Button type="button" variant="ghost" size="icon" disabled={disabled} onClick={onRemove}>
            <Trash2Icon className="size-4" />
          </Button>
        </div>
      </div>
      {(definition?.arguments ?? []).map((argument) => {
        const raw = step.arguments[argument.key];
        const value =
          argument.kind === "json"
            ? JSON.stringify(raw ?? {}, null, 2)
            : Array.isArray(raw)
              ? raw.join(", ")
              : formatArgumentValue(raw);
        const onValueChange = (nextValue: string) => {
          if (argument.kind === "number") setArg(argument.key, Number(nextValue) || 0);
          else if (argument.kind === "boolean") setArg(argument.key, nextValue === "true");
          else if (argument.kind === "json") {
            try {
              setArg(argument.key, JSON.parse(nextValue) as unknown);
            } catch {
              setArg(argument.key, nextValue);
            }
          } else if (argument.kind === "path-list") {
            setArg(
              argument.key,
              nextValue
                .split(",")
                .map((item) => item.trim())
                .filter(Boolean),
            );
          } else {
            setArg(argument.key, nextValue);
          }
        };
        return (
          <div key={argument.key} className="space-y-1">
            {argument.kind === "path-list" ? (
              <PathInsertField
                label={argument.label}
                value={value}
                placeholder={argument.placeholder}
                disabled={disabled}
                sourceFields={sourceFields}
                partnerFields={partnerFields}
                onChange={onValueChange}
              />
            ) : argument.kind === "json" ? (
              <TextareaBlock
                label={argument.label}
                value={value}
                onChange={onValueChange}
                disabled={disabled}
              />
            ) : (
              <InputBlock
                label={argument.label}
                value={value}
                onChange={onValueChange}
                disabled={disabled}
                placeholder={argument.placeholder}
              />
            )}
          </div>
        );
      })}
    </div>
  );
}

function ConditionEditor({
  condition,
  disabled,
  onChange,
}: {
  condition: string;
  disabled: boolean;
  onChange: (condition: string) => void;
}) {
  const [draft, setDraft] = useState<ConditionDraft>(() => parseConditionString(condition));

  useEffect(() => {
    setDraft(parseConditionString(condition));
  }, [condition]);

  const apply = (next: ConditionDraft) => {
    setDraft(next);
    onChange(buildConditionString(next));
  };
  const applyPreset = (preset: EDIScriptPreset) => {
    const next = parseConditionString(preset.code);
    if (draft.mode === "inlineStarlark" && next.mode === "inlineStarlark") {
      apply({
        mode: "inlineStarlark",
        script: insertScriptPresetCode(draft.script, { code: next.script }),
      });
      return;
    }
    apply(next);
  };

  return (
    <div className="space-y-2 rounded-md border p-2">
      <div className="text-xs font-semibold">Condition</div>
      <ScriptPresetPicker
        title="Presets"
        presets={getEDIScriptPresetsByCategory("condition")}
        disabled={disabled}
        onApply={applyPreset}
      />
      <SelectBlock
        label="Mode"
        value={draft.mode}
        disabled={disabled}
        onValueChange={(mode) => {
          if (mode === "none") apply({ mode: "none" });
          if (mode === "truthy") apply({ mode: "truthy", path: "" });
          if (mode === "falsey") apply({ mode: "falsey", path: "" });
          if (mode === "comparison")
            apply({ mode: "comparison", path: "", operator: "==", value: "" });
          if (mode === "starlarkFunction") apply({ mode: "starlarkFunction", functionName: "" });
          if (mode === "inlineStarlark") apply({ mode: "inlineStarlark", script: "" });
        }}
        options={[
          { value: "none", label: "None" },
          { value: "truthy", label: "Path Truthy" },
          { value: "falsey", label: "Path Falsey" },
          { value: "comparison", label: "Comparison" },
          { value: "starlarkFunction", label: "Starlark Function" },
          { value: "inlineStarlark", label: "Inline Starlark" },
        ]}
      />
      {draft.mode === "truthy" || draft.mode === "falsey" ? (
        <InputBlock
          label="Path"
          value={draft.path}
          disabled={disabled}
          onChange={(path) => apply({ ...draft, path })}
        />
      ) : null}
      {draft.mode === "comparison" ? (
        <div className="grid grid-cols-[1fr_76px_1fr] gap-2">
          <InputBlock
            label="Path"
            value={draft.path}
            disabled={disabled}
            onChange={(path) => apply({ ...draft, path })}
          />
          <SelectBlock
            label="Op"
            value={draft.operator}
            disabled={disabled}
            onValueChange={(operator) => apply({ ...draft, operator: operator as "==" | "!=" })}
            options={[
              { value: "==", label: "==" },
              { value: "!=", label: "!=" },
            ]}
          />
          <InputBlock
            label="Value"
            value={draft.value}
            disabled={disabled}
            onChange={(value) => apply({ ...draft, value })}
          />
        </div>
      ) : null}
      {draft.mode === "starlarkFunction" ? (
        <InputBlock
          label="Function"
          value={draft.functionName}
          disabled={disabled}
          onChange={(functionName) => apply({ ...draft, functionName })}
        />
      ) : null}
      {draft.mode === "inlineStarlark" ? (
        <TextareaBlock
          label="Script"
          value={draft.script}
          disabled={disabled}
          onChange={(script) => apply({ ...draft, script })}
        />
      ) : null}
    </div>
  );
}

function ScriptLibraryEditor({
  libraries,
  isEditable,
  onChange,
  onSave,
  isSaving,
}: {
  libraries: EDITemplateScriptLibrary[];
  isEditable: boolean;
  onChange: (libraries: EDITemplateScriptLibrary[]) => void;
  onSave: () => void;
  isSaving: boolean;
}) {
  const { theme } = useTheme();
  const editorTheme = theme === "dark" ? darkTheme : lightTheme;
  const [selectedId, setSelectedId] = useState("");
  const selected = libraries.find((library) => library.id === selectedId) ?? libraries[0];

  useEffect(() => {
    if (!selectedId && libraries[0]) setSelectedId(libraries[0].id);
  }, [libraries, selectedId]);

  const updateSelected = (patch: Partial<EDITemplateScriptLibrary>) => {
    if (!selected) return;
    onChange(
      libraries.map((library) => (library.id === selected.id ? { ...library, ...patch } : library)),
    );
  };
  const applyPreset = (preset: EDIScriptPreset) => {
    if (!selected) return;
    updateSelected({ script: insertScriptPresetCode(selected.script, preset) });
  };

  return (
    <div className="grid h-full grid-cols-[260px_minmax(0,1fr)]">
      <div className="min-h-0 overflow-auto border-r">
        <div className="flex items-center justify-between border-b px-3 py-2">
          <span className="text-xs font-semibold">Libraries</span>
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={!isEditable}
            onClick={() => {
              const id = `draft-${Date.now()}`;
              onChange([
                ...libraries,
                {
                  id,
                  templateVersionId: "",
                  name: "new_library",
                  description: "",
                  language: "Starlark",
                  script: "def normalize(value):\n    return value\n",
                  status: "Draft",
                  version: 0,
                  functionNames: ["normalize"],
                },
              ]);
              setSelectedId(id);
            }}
          >
            <PlusIcon className="size-4" />
          </Button>
        </div>
        {libraries.map((library) => (
          <button
            key={library.id}
            type="button"
            onClick={() => setSelectedId(library.id)}
            className={cn(
              "block w-full border-b px-3 py-2 text-left hover:bg-muted",
              selected?.id === library.id && "bg-muted",
            )}
          >
            <div className="truncate text-sm font-medium">{library.name}</div>
            <div className="truncate text-xs text-muted-foreground">
              {library.functionNames.length > 0
                ? library.functionNames.join(", ")
                : "No functions discovered"}
            </div>
          </button>
        ))}
      </div>
      <div className="grid min-h-0 grid-rows-[auto_auto_minmax(0,1fr)]">
        <div className="flex items-end justify-between gap-3 border-b p-3">
          <div className="grid flex-1 grid-cols-2 gap-2">
            <InputBlock
              label="Name"
              value={selected?.name ?? ""}
              disabled={!isEditable || !selected}
              onChange={(name) => updateSelected({ name })}
            />
            <InputBlock
              label="Description"
              value={selected?.description ?? ""}
              disabled={!isEditable || !selected}
              onChange={(description) => updateSelected({ description })}
            />
          </div>
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="outline"
              disabled={!isEditable || !selected}
              onClick={() =>
                selected && onChange(libraries.filter((library) => library.id !== selected.id))
              }
            >
              <Trash2Icon className="size-4" />
              Remove
            </Button>
            <Button type="button" disabled={!isEditable} isLoading={isSaving} onClick={onSave}>
              <SaveIcon className="size-4" />
              Save Scripts
            </Button>
          </div>
        </div>
        <div className="border-b p-3">
          <ScriptPresetPicker
            title="Script Presets"
            presets={getEDIScriptPresetsByCategory("scriptLibrary")}
            disabled={!isEditable || !selected}
            onApply={applyPreset}
          />
        </div>
        {selected ? (
          <div className="min-h-0 overflow-hidden">
            <CodeMirror
              value={selected.script}
              editable={isEditable}
              height="100%"
              extensions={[EditorView.lineWrapping, json()]}
              theme={editorTheme}
              basicSetup={{ lineNumbers: true, foldGutter: true, autocompletion: true }}
              onChange={(script) => updateSelected({ script })}
            />
          </div>
        ) : (
          <div className="p-4 text-sm text-muted-foreground">No script libraries.</div>
        )}
      </div>
    </div>
  );
}

function ValidationPanel({
  diagnostics,
  onSelectDiagnostic,
  onValidate,
  isLoading,
  disabled,
}: {
  diagnostics: EDIDiagnostic[];
  onSelectDiagnostic: (diagnostic: EDIDiagnostic) => void;
  onValidate: () => void;
  isLoading: boolean;
  disabled: boolean;
}) {
  return (
    <div className="min-h-0 overflow-auto p-3">
      <div className="mb-3 flex items-center justify-between">
        <div>
          <div className="text-sm font-semibold">Validation Diagnostics</div>
          <div className="text-xs text-muted-foreground">
            {diagnostics.length} diagnostics returned by backend validation
          </div>
        </div>
        <Button
          type="button"
          variant="outline"
          onClick={onValidate}
          isLoading={isLoading}
          disabled={disabled}
        >
          <ListChecksIcon className="size-4" />
          Run
        </Button>
      </div>
      <DiagnosticsList diagnostics={diagnostics} onSelect={onSelectDiagnostic} />
    </div>
  );
}

function TemplatePreviewPanel() {
  const [profileId, setProfileId] = useState("");
  const [shipmentId, setShipmentId] = useState("");
  const [transferId, setTransferId] = useState("");
  const [payloadJson, setPayloadJson] = useState("");
  const profilesQuery = useQuery(
    queries.edi.documentProfiles("?limit=100&transactionSet=204&direction=Outbound"),
  );
  const previewMutation = useApiMutation({
    mutationFn: async () =>
      apiService.ediService.previewDocument({
        partnerDocumentProfileId: profileId || undefined,
        shipmentId: shipmentId || undefined,
        transferId: transferId || undefined,
        payload: parsePayload(payloadJson),
      }),
    onError: () => toast.error("Failed to preview EDI document"),
  });
  const canPreview = !!profileId && (!!shipmentId || !!transferId || !!payloadJson.trim());

  return (
    <div className="grid min-h-0 grid-cols-[360px_minmax(0,1fr)]">
      <div className="space-y-3 border-r p-3">
        <SelectBlock
          label="Document Profile"
          value={profileId}
          onValueChange={setProfileId}
          options={(profilesQuery.data?.results ?? []).map((profile) => ({
            value: profile.id,
            label: profile.name,
          }))}
        />
        <InputBlock label="Shipment ID" value={shipmentId} onChange={setShipmentId} />
        <InputBlock label="Transfer ID" value={transferId} onChange={setTransferId} />
        <TextareaBlock label="Payload JSON" value={payloadJson} onChange={setPayloadJson} />
        <Button
          type="button"
          onClick={() => previewMutation.mutate(undefined)}
          isLoading={previewMutation.isPending}
          disabled={!canPreview}
        >
          <RefreshCwIcon className="size-4" />
          Preview
        </Button>
      </div>
      <PreviewPane preview={previewMutation.data} isLoading={previewMutation.isPending} />
    </div>
  );
}

function DocumentPreviewArchiveTab() {
  const queryClient = useQueryClient();
  const [partnerId, setPartnerId] = useState("");
  const [profileId, setProfileId] = useState("");
  const [shipmentId, setShipmentId] = useState("");
  const [rawPartnerSettings, setRawPartnerSettings] = useState("{}");
  const [profileDraft, setProfileDraft] =
    useState<UpsertEDIPartnerDocumentProfileRequest>(defaultProfileDraft);

  const partnersQuery = useQuery(queries.edi.partnerOptions());
  const profilesQuery = useQuery(
    queries.edi.documentProfiles("?limit=100&transactionSet=204&direction=Outbound"),
  );
  const templatesQuery = useQuery(
    queries.edi.templates("?limit=100&transactionSet=204&direction=Outbound"),
  );
  const messagesQuery = useQuery(
    queries.edi.messages("?limit=25&transactionSet=204&direction=Outbound"),
  );
  const selectedProfile = profilesQuery.data?.results.find((profile) => profile.id === profileId);
  const activeTemplate =
    templatesQuery.data?.results.find((template) => template.id === profileDraft.templateId) ??
    templatesQuery.data?.results[0];

  useEffect(() => {
    if (selectedProfile) {
      setPartnerId(selectedProfile.ediPartnerId);
      setProfileDraft(profileToDraft(selectedProfile));
      setRawPartnerSettings(JSON.stringify(selectedProfile.partnerSettings ?? {}, null, 2));
      return;
    }
    setProfileDraft((current) => ({
      ...current,
      ediPartnerId: partnerId,
      templateId: activeTemplate?.id ?? current.templateId,
    }));
  }, [activeTemplate?.id, partnerId, selectedProfile]);

  const saveProfileMutation = useApiMutation({
    mutationFn: async () => {
      const request = {
        ...profileDraft,
        ediPartnerId: partnerId,
        templateId: activeTemplate?.id ?? profileDraft.templateId,
        partnerSettings: parseSettings(rawPartnerSettings),
      };
      if (profileId) return apiService.ediService.updatePartnerDocumentProfile(profileId, request);
      return apiService.ediService.createPartnerDocumentProfile(request);
    },
    onSuccess: async (profile) => {
      toast.success("204 document profile saved");
      setProfileId(profile.id);
      await queryClient.invalidateQueries({ queryKey: queries.edi.documentProfiles._def });
    },
    onError: () => toast.error("Failed to save document profile"),
  });

  const previewMutation = useApiMutation({
    mutationFn: () =>
      apiService.ediService.previewDocument({
        partnerDocumentProfileId: profileId || undefined,
        ediPartnerId: partnerId || undefined,
        shipmentId: shipmentId || undefined,
      }),
    onError: () => toast.error("Failed to preview 204 document"),
  });

  const generateMutation = useApiMutation({
    mutationFn: () =>
      apiService.ediService.generateDocument({
        partnerDocumentProfileId: profileId || undefined,
        ediPartnerId: partnerId || undefined,
        shipmentId: shipmentId || undefined,
      }),
    onSuccess: async () => {
      toast.success("204 message generated and archived");
      await queryClient.invalidateQueries({ queryKey: queries.edi.messages._def });
    },
    onError: () => toast.error("Failed to generate 204 message"),
  });

  return (
    <div className="grid min-h-[calc(100vh-14rem)] grid-cols-[360px_minmax(0,1fr)_360px] gap-3">
      <aside className="flex min-h-0 flex-col rounded-md border bg-background">
        <PanelHeader icon={<ShieldCheckIcon />} title="204 Profile" />
        <div className="flex min-h-0 flex-col gap-3 overflow-auto p-3">
          <SelectBlock
            label="Partner"
            value={partnerId}
            onValueChange={setPartnerId}
            options={(partnersQuery.data?.results ?? [])
              .filter((partner) => partner.kind === "External")
              .map((partner) => ({
                value: partner.id,
                label: `${partner.code} - ${partner.name}`,
              }))}
          />
          <SelectBlock
            label="Document Profile"
            value={profileId}
            onValueChange={setProfileId}
            options={(profilesQuery.data?.results ?? []).map((profile) => ({
              value: profile.id,
              label: profile.name,
            }))}
          />
          <InputBlock
            label="Profile Name"
            value={profileDraft.name}
            onChange={(name) => setProfileDraft((current) => ({ ...current, name }))}
          />
          <SelectBlock
            label="Template"
            value={activeTemplate?.id ?? ""}
            onValueChange={(templateId) =>
              setProfileDraft((current) => ({
                ...current,
                templateId,
                templateVersionId: undefined,
              }))
            }
            options={(templatesQuery.data?.results ?? []).map((template) => ({
              value: template.id,
              label: template.name,
            }))}
          />
          <div className="grid grid-cols-2 gap-2">
            <InputBlock
              label="Version Override"
              value={profileDraft.x12VersionOverride ?? ""}
              onChange={(x12VersionOverride) =>
                setProfileDraft((current) => ({ ...current, x12VersionOverride }))
              }
            />
            <InputBlock
              label="Group"
              value={profileDraft.functionalGroupId}
              onChange={(functionalGroupId) =>
                setProfileDraft((current) => ({ ...current, functionalGroupId }))
              }
            />
          </div>
          <SelectBlock
            label="Validation"
            value={profileDraft.validationMode}
            onValueChange={(validationMode) =>
              setProfileDraft((current) => ({
                ...current,
                validationMode:
                  validationMode as UpsertEDIPartnerDocumentProfileRequest["validationMode"],
              }))
            }
            options={[
              { value: "Strict", label: "Strict" },
              { value: "WarnOnly", label: "Warn Only" },
              { value: "Disabled", label: "Disabled" },
            ]}
          />
          <EnvelopeEditor
            envelope={profileDraft.envelope}
            onChange={(envelope) => setProfileDraft((current) => ({ ...current, envelope }))}
          />
          <AckEditor profile={profileDraft} onChange={setProfileDraft} />
          <TextareaBlock
            label="Partner Settings"
            value={rawPartnerSettings}
            onChange={setRawPartnerSettings}
          />
          <Button
            type="button"
            onClick={() => saveProfileMutation.mutate(undefined)}
            isLoading={saveProfileMutation.isPending}
            disabled={!partnerId}
          >
            <ShieldCheckIcon className="size-4" />
            Save Profile
          </Button>
        </div>
      </aside>
      <main className="flex min-h-0 flex-col rounded-md border bg-background">
        <div className="flex items-center justify-between gap-3 border-b px-3 py-2">
          <div>
            <div className="text-sm font-semibold">Document Preview</div>
            <div className="text-xs text-muted-foreground">
              Uses the existing outbound document preview endpoint.
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Input
              value={shipmentId}
              onChange={(event) => setShipmentId(event.target.value)}
              placeholder="Shipment ID"
              className="w-56"
            />
            <Button
              type="button"
              variant="outline"
              onClick={() => previewMutation.mutate(undefined)}
              isLoading={previewMutation.isPending}
              disabled={!profileId && !partnerId}
            >
              <RefreshCwIcon className="size-4" />
              Preview
            </Button>
            <Button
              type="button"
              onClick={() => generateMutation.mutate(undefined)}
              isLoading={generateMutation.isPending}
              disabled={!profileId || !shipmentId}
            >
              <PlayIcon className="size-4" />
              Generate
            </Button>
          </div>
        </div>
        <PreviewPane preview={previewMutation.data} isLoading={previewMutation.isPending} />
      </main>
      <aside className="flex min-h-0 flex-col rounded-md border bg-background">
        <PanelHeader icon={<DatabaseIcon />} title="204 Archive" />
        <MessageArchive messages={messagesQuery.data?.results ?? []} />
      </aside>
    </div>
  );
}

function ScriptPresetPicker({
  title,
  presets,
  disabled,
  onApply,
}: {
  title: string;
  presets: EDIScriptPreset[];
  disabled?: boolean;
  onApply: (preset: EDIScriptPreset) => void;
}) {
  if (presets.length === 0) return null;
  return (
    <div className="space-y-2 rounded-md border bg-muted/20 p-2">
      <div className="text-xs font-semibold">{title}</div>
      <div className="space-y-1">
        {presets.map((preset) => (
          <button
            key={preset.id}
            type="button"
            disabled={disabled}
            onClick={() => onApply(preset)}
            className="flex w-full items-start gap-2 rounded-sm px-2 py-1.5 text-left hover:bg-background disabled:cursor-not-allowed disabled:opacity-50"
          >
            <CopyPlusIcon className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" />
            <span className="min-w-0">
              <span className="block text-xs font-medium">{preset.label}</span>
              <span className="block text-xs leading-snug text-muted-foreground">
                {preset.description}
              </span>
            </span>
          </button>
        ))}
      </div>
    </div>
  );
}

function PathField({
  label,
  value,
  onChange,
  fields,
  disabled,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  fields: EDISourceContextField[];
  disabled?: boolean;
}) {
  return (
    <div className="space-y-1">
      <InputBlock label={label} value={value} onChange={onChange} disabled={disabled} />
      <FieldPicker
        fields={fields}
        disabled={disabled}
        getPath={(field) => field.path}
        getLabel={(field) => `${field.path} (${field.dataType})`}
        onPick={onChange}
      />
    </div>
  );
}

function PartnerPathField({
  label,
  value,
  onChange,
  fields,
  disabled,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  fields: EDIPartnerSettingField[];
  disabled?: boolean;
}) {
  return (
    <div className="space-y-1">
      <InputBlock label={label} value={value} onChange={onChange} disabled={disabled} />
      <FieldPicker
        fields={fields}
        disabled={disabled}
        getPath={(field) => field.path}
        getLabel={(field) => `${field.path} (${field.dataType})`}
        onPick={onChange}
      />
    </div>
  );
}

function PathInsertField({
  label,
  value,
  placeholder,
  disabled,
  sourceFields,
  partnerFields,
  onChange,
}: {
  label: string;
  value: string;
  placeholder?: string;
  disabled: boolean;
  sourceFields: EDISourceContextField[];
  partnerFields: EDIPartnerSettingField[];
  onChange: (value: string) => void;
}) {
  return (
    <div className="space-y-1">
      <InputBlock
        label={label}
        value={value}
        onChange={onChange}
        disabled={disabled}
        placeholder={placeholder}
      />
      <div className="grid grid-cols-2 gap-1">
        <FieldPicker
          fields={sourceFields}
          disabled={disabled}
          getPath={(field) => field.path}
          getLabel={(field) => field.path}
          onPick={(path) => onChange(insertPathReference(value, path))}
        />
        <FieldPicker
          fields={partnerFields}
          disabled={disabled}
          getPath={(field) => `partner.${field.path}`}
          getLabel={(field) => field.path}
          onPick={(path) => onChange(insertPathReference(value, path))}
        />
      </div>
    </div>
  );
}

function FieldPicker<T>({
  fields,
  disabled,
  getPath,
  getLabel,
  onPick,
}: {
  fields: T[];
  disabled?: boolean;
  getPath: (field: T) => string;
  getLabel: (field: T) => string;
  onPick: (path: string) => void;
}) {
  const [filter, setFilter] = useState("");
  const visible = fields
    .filter((field) => getLabel(field).toLowerCase().includes(filter.toLowerCase()))
    .slice(0, 8);
  return (
    <div className="rounded-md border bg-muted/20 p-2">
      <div className="mb-1 flex items-center gap-1">
        <FilterIcon className="size-3 text-muted-foreground" />
        <Input
          value={filter}
          disabled={disabled}
          onChange={(event) => setFilter(event.target.value)}
          placeholder="Find path"
          className="h-7 text-xs"
        />
      </div>
      <div className="max-h-32 space-y-1 overflow-auto">
        {visible.map((field) => (
          <button
            key={getPath(field)}
            type="button"
            disabled={disabled}
            onClick={() => onPick(getPath(field))}
            className="block w-full truncate rounded-sm px-1.5 py-1 text-left font-mono text-xs hover:bg-background disabled:opacity-50"
          >
            {getLabel(field)}
          </button>
        ))}
      </div>
    </div>
  );
}

function PreviewPane({ preview, isLoading }: { preview?: EDIDocumentPreview; isLoading: boolean }) {
  return (
    <div className="grid min-h-0 flex-1 grid-cols-[minmax(0,1fr)_300px]">
      <pre className="min-h-0 overflow-auto bg-zinc-950 p-3 font-mono text-xs text-zinc-100">
        {isLoading ? "Rendering preview..." : (preview?.rawX12 ?? "Preview output appears here.")}
      </pre>
      <DiagnosticsList diagnostics={preview?.diagnostics ?? []} />
    </div>
  );
}

function DiagnosticsList({
  diagnostics,
  onSelect,
}: {
  diagnostics: EDIDiagnostic[];
  onSelect?: (diagnostic: EDIDiagnostic) => void;
}) {
  const grouped = {
    Error: diagnostics.filter((diagnostic) => diagnostic.severity === "Error"),
    Warning: diagnostics.filter((diagnostic) => diagnostic.severity === "Warning"),
    Info: diagnostics.filter((diagnostic) => diagnostic.severity === "Info"),
  };
  return (
    <div className="min-h-0 overflow-auto p-3">
      {diagnostics.length === 0 ? (
        <div className="text-sm text-muted-foreground">No diagnostics.</div>
      ) : (
        (Object.keys(grouped) as Array<keyof typeof grouped>).map((severity) =>
          grouped[severity].length > 0 ? (
            <div key={severity} className="mb-3 space-y-2">
              <div className="text-xs font-semibold">{severity}</div>
              {grouped[severity].map((diagnostic) => (
                <button
                  key={diagnosticKey(diagnostic)}
                  type="button"
                  onClick={() => onSelect?.(diagnostic)}
                  className="block w-full rounded-md border p-2 text-left hover:bg-muted"
                >
                  <div className="flex items-center gap-2">
                    <Badge variant={diagnostic.severity === "Error" ? "inactive" : "warning"}>
                      {diagnostic.severity}
                    </Badge>
                    <span className="font-mono text-xs">
                      {diagnostic.segmentId ?? diagnostic.path}
                    </span>
                  </div>
                  <div className="mt-1 text-xs">{diagnostic.message}</div>
                  {diagnostic.suggestedFix ? (
                    <div className="mt-1 text-xs text-muted-foreground">
                      {diagnostic.suggestedFix}
                    </div>
                  ) : null}
                </button>
              ))}
            </div>
          ) : null,
        )
      )}
    </div>
  );
}

function MessageArchive({ messages }: { messages: EDIMessage[] }) {
  const [selectedId, setSelectedId] = useState("");
  const selected = messages.find((message) => message.id === selectedId) ?? messages[0];
  return (
    <div className="grid min-h-0 flex-1 grid-rows-[220px_minmax(0,1fr)]">
      <div className="min-h-0 overflow-auto border-b">
        {messages.map((message) => (
          <button
            key={message.id}
            type="button"
            onClick={() => setSelectedId(message.id)}
            className={cn(
              "block w-full border-b px-3 py-2 text-left hover:bg-muted",
              selected?.id === message.id && "bg-muted",
            )}
          >
            <div className="flex items-center justify-between gap-2">
              <span className="font-mono text-xs">{message.transactionControlNumber}</span>
              <Badge variant="info">{message.x12Version}</Badge>
            </div>
            <div className="mt-1 text-xs text-muted-foreground">
              ISA {message.interchangeControlNumber} / GS {message.groupControlNumber}
            </div>
            <div className="text-xs text-muted-foreground">{formatUnix(message.generatedAt)}</div>
          </button>
        ))}
      </div>
      <pre className="min-h-0 overflow-auto bg-zinc-950 p-3 font-mono text-xs text-zinc-100">
        {selected?.rawX12 ?? "Generated 204 messages appear here."}
      </pre>
    </div>
  );
}

function EnvelopeEditor({
  envelope,
  onChange,
}: {
  envelope: EDIX12EnvelopeSettings;
  onChange: (envelope: EDIX12EnvelopeSettings) => void;
}) {
  const update = (key: keyof EDIX12EnvelopeSettings, value: string) => {
    onChange({ ...envelope, [key]: value });
  };
  return (
    <div className="space-y-2 rounded-md border bg-muted/30 p-2">
      <div className="text-xs font-medium">X12 Envelope</div>
      <div className="grid grid-cols-2 gap-2">
        <InputBlock
          label="ISA Sender"
          value={envelope.interchangeSenderId}
          onChange={(value) => update("interchangeSenderId", value)}
        />
        <InputBlock
          label="ISA Receiver"
          value={envelope.interchangeReceiverId}
          onChange={(value) => update("interchangeReceiverId", value)}
        />
        <InputBlock
          label="GS Sender"
          value={envelope.applicationSenderCode}
          onChange={(value) => update("applicationSenderCode", value)}
        />
        <InputBlock
          label="GS Receiver"
          value={envelope.applicationReceiverCode}
          onChange={(value) => update("applicationReceiverCode", value)}
        />
      </div>
    </div>
  );
}

function AckEditor({
  profile,
  onChange,
}: {
  profile: UpsertEDIPartnerDocumentProfileRequest;
  onChange: Dispatch<SetStateAction<UpsertEDIPartnerDocumentProfileRequest>>;
}) {
  return (
    <div className="space-y-2 rounded-md border bg-muted/30 p-2">
      <div className="flex items-center justify-between">
        <div className="text-xs font-medium">Acknowledgment</div>
        <Switch
          checked={profile.acknowledgment.expected}
          onCheckedChange={(expected) =>
            onChange((current) => ({
              ...current,
              acknowledgment: { ...current.acknowledgment, expected },
            }))
          }
        />
      </div>
      <div className="grid grid-cols-2 gap-2">
        <SelectBlock
          label="Type"
          value={profile.acknowledgment.type}
          onValueChange={(type) =>
            onChange((current) => ({
              ...current,
              acknowledgment: { ...current.acknowledgment, type },
            }))
          }
          options={[
            { value: "None", label: "None" },
            { value: "997", label: "997" },
            { value: "999", label: "999" },
          ]}
        />
        <InputBlock
          label="SLA Minutes"
          value={String(profile.acknowledgment.slaInMinutes)}
          onChange={(slaInMinutes) =>
            onChange((current) => ({
              ...current,
              acknowledgment: {
                ...current.acknowledgment,
                slaInMinutes: Number(slaInMinutes) || 0,
              },
            }))
          }
        />
      </div>
    </div>
  );
}

function PanelHeader({ icon, title }: { icon: ReactNode; title: string }) {
  return (
    <div className="flex h-11 items-center gap-2 border-b px-3">
      <span className="text-muted-foreground [&_svg]:size-4">{icon}</span>
      <span className="text-sm font-semibold">{title}</span>
    </div>
  );
}

function ReadOnlyBanner({ reason }: { reason: string }) {
  return (
    <div className="flex items-center gap-2 border-b bg-muted/50 px-3 py-2 text-xs text-muted-foreground">
      <AlertTriangleIcon className="size-4" />
      {reason}
    </div>
  );
}

function VersionStatusBadge({ version }: { version: EDITemplateVersion }) {
  const variant =
    version.status === "Active" ? "active" : version.status === "Draft" ? "warning" : "outline";
  return <Badge variant={variant}>{version.isActive ? "Active" : version.status}</Badge>;
}

function InputBlock({
  label,
  value,
  onChange,
  disabled,
  placeholder,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
  placeholder?: string;
}) {
  return (
    <div className="space-y-1">
      <Label className="text-xs text-muted-foreground">{label}</Label>
      <Input
        value={value}
        disabled={disabled}
        placeholder={placeholder}
        onChange={(event) => onChange(event.target.value)}
      />
    </div>
  );
}

function TextareaBlock({
  label,
  value,
  onChange,
  disabled,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
}) {
  return (
    <div className="space-y-1">
      <Label className="text-xs text-muted-foreground">{label}</Label>
      <Textarea
        value={value}
        disabled={disabled}
        onChange={(event) => onChange(event.target.value)}
        className="min-h-24 font-mono text-xs"
      />
    </div>
  );
}

function SelectBlock({
  label,
  value,
  options,
  onValueChange,
  placeholder = "Select",
  disabled,
}: {
  label: string;
  value: string;
  options: { value: string; label: string }[];
  onValueChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
}) {
  return (
    <div className="space-y-1">
      <Label className="text-xs text-muted-foreground">{label}</Label>
      <select
        value={value}
        disabled={disabled}
        onChange={(event) => onValueChange(event.target.value)}
        className="h-8 w-full rounded-md border border-input bg-muted px-2 text-sm outline-none focus-visible:border-brand focus-visible:ring-4 focus-visible:ring-brand/30 disabled:opacity-50"
      >
        <option value="">{placeholder}</option>
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </div>
  );
}

function templateElementSourceLabel(element: EDITemplateElement) {
  if (element.source === "transform") {
    const base = element.baseSource
      ? templateBaseSourceLabel(element.baseSource)
      : "No base source";
    const steps =
      element.transformPipeline.length > 0
        ? element.transformPipeline.map((step) => step.operation).join(" -> ")
        : "No transforms";
    return `${base} / ${steps}`;
  }
  if (element.source === "starlark")
    return element.starlarkFunction ?? element.starlarkScript ?? "Starlark script";
  return (
    element.fieldPath ??
    element.runtimeKey ??
    element.mappingSourcePath ??
    element.partnerSettingPath ??
    element.repeatPath ??
    element.value ??
    element.default ??
    ""
  );
}

function templateBaseSourceLabel(source: EDITemplateElementBaseSource) {
  return (
    source.fieldPath ??
    source.runtimeKey ??
    source.mappingSourcePath ??
    source.partnerSettingPath ??
    source.repeatPath ??
    source.value ??
    source.default ??
    source.source
  );
}

function profileToDraft(
  profile: EDIPartnerDocumentProfile,
): UpsertEDIPartnerDocumentProfileRequest {
  return {
    ediPartnerId: profile.ediPartnerId,
    templateId: profile.templateId,
    templateVersionId: profile.templateVersionId ?? undefined,
    name: profile.name,
    status: profile.status,
    x12VersionOverride: profile.x12VersionOverride ?? undefined,
    functionalGroupId: profile.functionalGroupId,
    envelope: profile.envelope,
    acknowledgment: profile.acknowledgment,
    validationMode: profile.validationMode,
    partnerSettings: profile.partnerSettings,
    version: profile.version,
  };
}

function parseSettings(value: string): Record<string, unknown> {
  try {
    const parsed = JSON.parse(value) as unknown;
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
  } catch {
    toast.error("Partner settings must be valid JSON");
  }
  return {};
}

function parsePayload(value: string) {
  if (!value.trim()) return undefined;
  try {
    return JSON.parse(value) as never;
  } catch {
    toast.error("Payload must be valid JSON");
    return undefined;
  }
}

function formatArgumentValue(value: unknown) {
  if (value == null) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return JSON.stringify(value);
}
