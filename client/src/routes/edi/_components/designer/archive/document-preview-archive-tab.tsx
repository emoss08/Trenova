import { DocumentSourceControls } from "@/components/edi/document-source-controls";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
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
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import {
  buildEDIDocumentResolutionRequest,
  hasEDIDocumentSourceValue,
  pruneEDIDocumentSourceValues,
  resolveEDIDocumentSourceContext,
  type EDIDocumentSourceField,
  type EDIDocumentSourceValues,
} from "@/lib/edi/document-source";
import { downloadJsonFile, downloadTextFile } from "@/lib/utils";
import type {
  EDIMessage,
  EDIPartnerDocumentProfile,
  UpsertEDIPartnerDocumentProfileRequest,
} from "@/types/edi";
import { json } from "@codemirror/lang-json";
import { EditorView } from "@codemirror/view";
import CodeMirror from "@uiw/react-codemirror";
import {
  ClipboardCheckIcon,
  CopyIcon,
  DatabaseIcon,
  DownloadIcon,
  EyeIcon,
  FileCode2Icon,
  FileJsonIcon,
  InfoIcon,
  PlayIcon,
  RefreshCwIcon,
  ShieldCheckIcon,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import { formatUnix } from "../../edi-display-utils";
import {
  ControlledSelectField,
  EDIDocumentProfileAutocompleteField,
  EDIPartnerAutocompleteField,
  EDITemplateAutocompleteField,
} from "../components/designer-fields";
import {
  InputBlock,
  PanelHeader,
  PreviewPane,
  TextareaBlock,
  parsePayload,
  parseSettings,
  profileToDraft,
  useEditorTheme,
} from "../components/designer-shared";
import { useDocumentArchiveUrlState } from "../hooks/use-edi-designer-url-state";
import {
  useGenerateEDIDocumentMutation,
  useInvalidateEDIDocumentProfiles,
  useInvalidateEDIMessageArchive,
  usePreviewEDIDocumentMutation,
  useSaveEDIDocumentProfileMutation,
} from "../hooks/use-edi-document-mutations";
import {
  useEDIDocumentArchiveQueries,
  useEDIMessageDetailQuery,
} from "../hooks/use-edi-document-queries";
import { AckEditor } from "../profile/ack-editor";
import { EnvelopeEditor } from "../profile/envelope-editor";
import {
  documentDirectionOptions,
  documentStatusOptions,
  messageStatusOptions,
  transactionSetOptions,
  validationModeOptions,
} from "../utils/edi-designer-options";
import {
  buildEDIDocumentContextQuery,
  buildNewPartnerDocumentProfileDraft,
  diagnosticKey,
  resolveSelectedDocumentTemplateId,
} from "../utils/edi-designer-utils";
import {
  buildArchiveMessagesQueryString,
  buildMessageJsonFilename,
  buildX12Filename,
  formatRawX12Display,
  groupDiagnostics,
  parseX12Segments,
} from "../utils/edi-message-utils";

const defaultEnvelope = {
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
  name: "EDI Document Profile",
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

export function DocumentPreviewArchiveTab() {
  const [
    [
      {
        archivePartnerId,
        archiveTransactionSet,
        archiveDirection,
        archiveStatus,
        archiveGeneratedFrom,
        archiveGeneratedTo,
        archiveQuery,
      },
      setArchiveUrlState,
    ],
    [inspectorMessageId, setInspectorMessageId],
  ] = useDocumentArchiveUrlState();
  const [partnerId, setPartnerId] = useState("");
  const [profileId, setProfileId] = useState("");
  const [selectedProfile, setSelectedProfile] = useState<EDIPartnerDocumentProfile | null>(null);
  const [sourceValues, setSourceValues] = useState<EDIDocumentSourceValues>({});
  const [rawPartnerSettings, setRawPartnerSettings] = useState("{}");
  const [profileDraft, setProfileDraft] =
    useState<UpsertEDIPartnerDocumentProfileRequest>(defaultProfileDraft);

  const messagesQueryString = useMemo(
    () =>
      buildArchiveMessagesQueryString({
        partnerId: archivePartnerId,
        transactionSet: archiveTransactionSet,
        direction: archiveDirection,
        status: archiveStatus,
        generatedFrom: archiveGeneratedFrom,
        generatedTo: archiveGeneratedTo,
        query: archiveQuery,
      }),
    [
      archiveDirection,
      archiveGeneratedFrom,
      archiveGeneratedTo,
      archivePartnerId,
      archiveQuery,
      archiveStatus,
      archiveTransactionSet,
    ],
  );

  const profileContextQueryString = useMemo(
    () =>
      buildEDIDocumentContextQuery({
        limit: 100,
        partnerId: partnerId || undefined,
        transactionSet: archiveTransactionSet,
        direction: archiveDirection,
      }),
    [archiveDirection, archiveTransactionSet, partnerId],
  );
  const documentContextQueryString = useMemo(
    () =>
      buildEDIDocumentContextQuery({
        limit: 100,
        transactionSet: archiveTransactionSet,
        direction: archiveDirection,
      }),
    [archiveDirection, archiveTransactionSet],
  );
  const { profilesQuery, templatesQuery, messagesQuery } = useEDIDocumentArchiveQueries({
    messagesQueryString,
    profilesQueryString: profileContextQueryString,
    templatesQueryString: documentContextQueryString,
  });
  const queriedSelectedProfile =
    profilesQuery.data?.results.find((profile) => profile.id === profileId) ?? null;
  const firstTemplateId = templatesQuery.data?.results[0]?.id;
  const selectedTemplateId = resolveSelectedDocumentTemplateId(
    profileDraft.templateId,
    firstTemplateId,
  );
  const activeTemplate =
    templatesQuery.data?.results.find((template) => template.id === selectedTemplateId) ??
    templatesQuery.data?.results[0];
  const selectedDocumentProfile =
    selectedProfile?.id === profileId ? selectedProfile : queriedSelectedProfile;
  const sourceContext = resolveEDIDocumentSourceContext({
    profile: selectedDocumentProfile,
    template: activeTemplate,
    fallbackTransactionSet: archiveTransactionSet || activeTemplate?.transactionSet,
    fallbackDirection: archiveDirection || activeTemplate?.direction,
  });
  const sourceTransactionSet = sourceContext.transactionSet;
  const sourceDirection = sourceContext.direction;
  const hasSourceValue = hasEDIDocumentSourceValue(sourceValues, sourceTransactionSet);
  const matchingProfiles = profilesQuery.data?.results ?? [];
  const selectedPartnerHasNoProfiles =
    !!partnerId && !profileId && !profilesQuery.isLoading && matchingProfiles.length === 0;
  const isCreatingProfile = !!partnerId && !profileId;
  const invalidateDocumentProfiles = useInvalidateEDIDocumentProfiles();
  const invalidateMessageArchive = useInvalidateEDIMessageArchive();

  useEffect(() => {
    if (selectedDocumentProfile) {
      setPartnerId(selectedDocumentProfile.ediPartnerId);
      setProfileDraft(profileToDraft(selectedDocumentProfile));
      setRawPartnerSettings(JSON.stringify(selectedDocumentProfile.partnerSettings ?? {}, null, 2));
    }
  }, [selectedDocumentProfile]);

  useEffect(() => {
    setSourceValues((current) => pruneEDIDocumentSourceValues(current, sourceTransactionSet));
  }, [sourceTransactionSet]);

  const saveProfileMutation = useSaveEDIDocumentProfileMutation({
    onSuccess: async (profile) => {
      toast.success("Document profile saved");
      setProfileId(profile.id);
      setSelectedProfile(profile);
      await invalidateDocumentProfiles(profile);
    },
    onError: () => toast.error("Failed to save document profile"),
  });

  const previewMutation = usePreviewEDIDocumentMutation({
    onError: () => toast.error("Failed to preview EDI document"),
  });

  const generateMutation = useGenerateEDIDocumentMutation({
    onSuccess: async (message) => {
      toast.success("EDI message generated and archived");
      void setInspectorMessageId(message.id);
      await invalidateMessageArchive();
    },
    onError: () => toast.error("Failed to generate EDI message"),
  });

  const setSourceValue = (field: EDIDocumentSourceField, value: string) => {
    setSourceValues((current) => ({ ...current, [field]: value }));
  };

  return (
    <div className="grid min-h-[calc(100vh-14rem)] grid-cols-[360px_minmax(0,1fr)] gap-3">
      <aside className="flex min-h-0 flex-col rounded-md border bg-background">
        <PanelHeader icon={<ShieldCheckIcon />} title="Document Profile" />
        <div className="flex min-h-0 flex-col gap-3 overflow-auto p-3">
          <EDIPartnerAutocompleteField
            value={partnerId}
            onValueChange={(nextPartnerId) => {
              setPartnerId(nextPartnerId);
              setProfileId("");
              setSelectedProfile(null);
              setRawPartnerSettings("{}");
              setProfileDraft(
                buildNewPartnerDocumentProfileDraft({
                  defaultDraft: defaultProfileDraft,
                  partnerId: nextPartnerId,
                  templateId: firstTemplateId,
                  status: templatesQuery.data?.results[0]?.activeVersion ? "Active" : "Inactive",
                }),
              );
            }}
          />
          <EDIDocumentProfileAutocompleteField
            value={profileId}
            onValueChange={(nextProfileId) => {
              setProfileId(nextProfileId);
              setSelectedProfile((current) => (current?.id === nextProfileId ? current : null));
            }}
            onOptionChange={setSelectedProfile}
            partnerId={partnerId}
            transactionSet={archiveTransactionSet}
            direction={archiveDirection}
            disabled={!partnerId}
            placeholder={partnerId ? "Select document profile" : "Select a partner first."}
            description={!partnerId ? "Select a partner first." : undefined}
            noResultsMessage="No document profiles match this partner and document context."
          />
          {selectedPartnerHasNoProfiles && (
            <Alert variant="info" className="py-2 text-xs">
              <InfoIcon className="size-4" />
              <AlertDescription className="text-xs">
                No document profiles exist for this partner yet. Fill the profile details below and
                click Save Profile.
              </AlertDescription>
            </Alert>
          )}
          {isCreatingProfile && (
            <Alert variant="info" className="py-2 text-xs">
              <InfoIcon className="size-4" />
              <AlertDescription className="text-xs">
                New profile for selected partner. Save Profile will create and select it.
              </AlertDescription>
            </Alert>
          )}
          <InputBlock
            label="Profile Name"
            value={profileDraft.name}
            onChange={(name) => setProfileDraft((current) => ({ ...current, name }))}
          />
          <EDITemplateAutocompleteField
            value={selectedTemplateId}
            transactionSet={archiveTransactionSet}
            direction={archiveDirection}
            onValueChange={(templateId) => {
              const selectedTemplateHasProductionVersion = !!templatesQuery.data?.results.find(
                (template) => template.id === templateId,
              )?.activeVersion;
              setProfileDraft((current) => ({
                ...current,
                templateId,
                templateVersionId: undefined,
                status:
                  selectedTemplateHasProductionVersion || current.status === "Inactive"
                    ? current.status
                    : "Inactive",
              }));
            }}
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
          <ControlledSelectField
            label="Status"
            value={profileDraft.status}
            onValueChange={(status) =>
              setProfileDraft((current) => ({
                ...current,
                status: status as UpsertEDIPartnerDocumentProfileRequest["status"],
              }))
            }
            options={documentStatusOptions}
            clearable={false}
          />
          <ControlledSelectField
            label="Validation"
            value={profileDraft.validationMode}
            onValueChange={(validationMode) =>
              setProfileDraft((current) => ({
                ...current,
                validationMode:
                  validationMode as UpsertEDIPartnerDocumentProfileRequest["validationMode"],
              }))
            }
            options={validationModeOptions}
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
            onClick={() =>
              saveProfileMutation.mutate({
                profileId,
                request: {
                  ...profileDraft,
                  ediPartnerId: partnerId,
                  templateId: selectedTemplateId || undefined,
                  partnerSettings: parseSettings(rawPartnerSettings),
                },
              })
            }
            isLoading={saveProfileMutation.isPending}
            disabled={!partnerId}
          >
            <ShieldCheckIcon className="size-4" />
            Save Profile
          </Button>
        </div>
      </aside>
      <main className="min-h-0 rounded-md border bg-background">
        <Tabs
          defaultValue="preview"
          className="grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)] gap-0"
        >
          <div className="flex items-center justify-between gap-3 border-b px-3 py-2">
            <TabsList className="grid w-fit grid-cols-2">
              <TabsTrigger value="preview">
                <FileCode2Icon data-icon="inline-start" />
                Preview
              </TabsTrigger>
              <TabsTrigger value="archive">
                <DatabaseIcon data-icon="inline-start" />
                Archive
              </TabsTrigger>
            </TabsList>
            <div className="flex items-center gap-2">
              <DocumentSourceControls
                transactionSet={sourceTransactionSet}
                values={sourceValues}
                onChange={setSourceValue}
                layout="toolbar"
              />
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  const payloadResult = parsePayload(sourceValues.payload ?? "");
                  if (!payloadResult.ok) return;
                  previewMutation.mutate(
                    buildEDIDocumentResolutionRequest({
                      partnerDocumentProfileId: profileId || undefined,
                      ediPartnerId: partnerId || undefined,
                      sourceValues,
                      transactionSet: sourceTransactionSet,
                      direction: sourceDirection,
                      payload: payloadResult.payload,
                    }),
                  );
                }}
                isLoading={previewMutation.isPending}
                disabled={(!profileId && !partnerId) || !hasSourceValue}
              >
                <RefreshCwIcon className="size-4" />
                Preview
              </Button>
              <Button
                type="button"
                onClick={() => {
                  const payloadResult = parsePayload(sourceValues.payload ?? "");
                  if (!payloadResult.ok) return;
                  generateMutation.mutate(
                    buildEDIDocumentResolutionRequest({
                      partnerDocumentProfileId: profileId || undefined,
                      ediPartnerId: partnerId || undefined,
                      sourceValues,
                      transactionSet: sourceTransactionSet,
                      direction: sourceDirection,
                      payload: payloadResult.payload,
                    }),
                  );
                }}
                isLoading={generateMutation.isPending}
                disabled={!profileId || !hasSourceValue}
              >
                <PlayIcon className="size-4" />
                Generate
              </Button>
            </div>
          </div>
          <TabsContent value="preview" className="min-h-0">
            <PreviewPane preview={previewMutation.data} isLoading={previewMutation.isPending} />
          </TabsContent>
          <TabsContent value="archive" className="min-h-0">
            <MessageArchive
              messages={messagesQuery.data?.results ?? []}
              isLoading={messagesQuery.isLoading}
              filters={{
                partnerId: archivePartnerId,
                transactionSet: archiveTransactionSet,
                direction: archiveDirection,
                status: archiveStatus,
                generatedFrom: archiveGeneratedFrom,
                generatedTo: archiveGeneratedTo,
                query: archiveQuery,
              }}
              onFiltersChange={(patch) => {
                const nextFilters: Partial<{
                  archivePartnerId: string;
                  archiveTransactionSet: string;
                  archiveDirection: string;
                  archiveStatus: string;
                  archiveGeneratedFrom: string;
                  archiveGeneratedTo: string;
                  archiveQuery: string;
                }> = {};
                if (patch.partnerId !== undefined) nextFilters.archivePartnerId = patch.partnerId;
                if (patch.transactionSet !== undefined) {
                  nextFilters.archiveTransactionSet = patch.transactionSet;
                }
                if (patch.direction !== undefined) nextFilters.archiveDirection = patch.direction;
                if (patch.status !== undefined) nextFilters.archiveStatus = patch.status;
                if (patch.generatedFrom !== undefined) {
                  nextFilters.archiveGeneratedFrom = patch.generatedFrom;
                }
                if (patch.generatedTo !== undefined)
                  nextFilters.archiveGeneratedTo = patch.generatedTo;
                if (patch.query !== undefined) nextFilters.archiveQuery = patch.query;
                void setArchiveUrlState(nextFilters);
              }}
              onOpenMessage={(messageId) => void setInspectorMessageId(messageId)}
            />
          </TabsContent>
        </Tabs>
      </main>
      <MessageDetailInspector
        messageId={inspectorMessageId}
        open={!!inspectorMessageId}
        onOpenChange={(open) => {
          if (!open) void setInspectorMessageId("");
        }}
      />
    </div>
  );
}

type MessageArchiveFilters = {
  partnerId: string;
  transactionSet: string;
  direction: string;
  status: string;
  generatedFrom: string;
  generatedTo: string;
  query: string;
};

function MessageArchive({
  messages,
  isLoading,
  filters,
  onFiltersChange,
  onOpenMessage,
}: {
  messages: EDIMessage[];
  isLoading: boolean;
  filters: MessageArchiveFilters;
  onFiltersChange: (patch: Partial<MessageArchiveFilters>) => void;
  onOpenMessage: (messageId: string) => void;
}) {
  const { copy } = useCopyToClipboard();
  const copyControlNumbers = (message: EDIMessage) => {
    void copy(controlNumberText(message), { withToast: true });
  };

  return (
    <div className="grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)]">
      <div className="grid grid-cols-1 gap-2 border-b p-3 md:grid-cols-2 xl:grid-cols-[repeat(4,minmax(140px,1fr))_minmax(220px,1.4fr)]">
        <EDIPartnerAutocompleteField
          value={filters.partnerId}
          onValueChange={(partnerId) => onFiltersChange({ partnerId })}
          placeholder="All partners"
        />
        <ControlledSelectField
          label="Transaction"
          value={filters.transactionSet}
          onValueChange={(transactionSet) => onFiltersChange({ transactionSet })}
          options={transactionSetOptions}
          placeholder="All sets"
        />
        <ControlledSelectField
          label="Direction"
          value={filters.direction}
          onValueChange={(direction) => onFiltersChange({ direction })}
          options={documentDirectionOptions}
          placeholder="All directions"
        />
        <ControlledSelectField
          label="Status"
          value={filters.status}
          onValueChange={(status) => onFiltersChange({ status })}
          options={messageStatusOptions}
          placeholder="All statuses"
        />
        <InputBlock
          label="Search"
          value={filters.query}
          onChange={(query) => onFiltersChange({ query })}
          placeholder="Message, shipment, transfer, ISA, GS, ST"
        />
        <InputBlock
          label="Generated From"
          value={filters.generatedFrom}
          onChange={(generatedFrom) => onFiltersChange({ generatedFrom })}
          placeholder="YYYY-MM-DD"
        />
        <InputBlock
          label="Generated To"
          value={filters.generatedTo}
          onChange={(generatedTo) => onFiltersChange({ generatedTo })}
          placeholder="YYYY-MM-DD"
        />
      </div>
      <div className="min-h-0 overflow-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="min-w-36">Generated</TableHead>
              <TableHead className="min-w-48">Partner</TableHead>
              <TableHead>Set</TableHead>
              <TableHead>Direction</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Version</TableHead>
              <TableHead>Control Numbers</TableHead>
              <TableHead>Shipment</TableHead>
              <TableHead>Transfer</TableHead>
              <TableHead>Diagnostics</TableHead>
              <TableHead className="w-36">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell colSpan={11} className="h-24 text-center text-muted-foreground">
                  Loading archive messages.
                </TableCell>
              </TableRow>
            ) : messages.length === 0 ? (
              <TableRow>
                <TableCell colSpan={11} className="h-24 text-center text-muted-foreground">
                  No generated messages match the current filters.
                </TableCell>
              </TableRow>
            ) : (
              messages.map((message) => (
                <TableRow key={message.id}>
                  <TableCell className="text-xs whitespace-nowrap">
                    {formatUnix(message.generatedAt)}
                  </TableCell>
                  <TableCell>
                    <div className="max-w-56 truncate text-sm font-medium">
                      {message.partner?.name ??
                        message.partnerDocumentProfile?.partner?.name ??
                        "-"}
                    </div>
                    <div className="font-mono text-xs text-muted-foreground">
                      {message.partner?.code ?? message.partnerDocumentProfile?.partner?.code ?? ""}
                    </div>
                  </TableCell>
                  <TableCell className="font-mono">{message.transactionSet}</TableCell>
                  <TableCell>{message.direction}</TableCell>
                  <TableCell>
                    <Badge variant={message.status === "Generated" ? "active" : "inactive"}>
                      {message.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="font-mono text-xs">{message.x12Version}</TableCell>
                  <TableCell className="font-mono text-xs">
                    <div>ISA {message.interchangeControlNumber}</div>
                    <div>GS {message.groupControlNumber}</div>
                    <div>ST {message.transactionControlNumber}</div>
                  </TableCell>
                  <TableCell className="font-mono text-xs">{message.shipmentId ?? "-"}</TableCell>
                  <TableCell className="font-mono text-xs">{message.transferId ?? "-"}</TableCell>
                  <TableCell>
                    <Badge variant={message.diagnosticCount > 0 ? "warning" : "outline"}>
                      {message.diagnosticCount}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button
                        type="button"
                        size="icon-sm"
                        variant="ghost"
                        title="Open detail"
                        onClick={() => onOpenMessage(message.id)}
                      >
                        <EyeIcon className="size-4" />
                      </Button>
                      <Button
                        type="button"
                        size="icon-sm"
                        variant="ghost"
                        title="Copy control numbers"
                        onClick={() => copyControlNumbers(message)}
                      >
                        <CopyIcon className="size-4" />
                      </Button>
                      <Button
                        type="button"
                        size="icon-sm"
                        variant="ghost"
                        title="Copy raw X12"
                        disabled={!message.rawX12}
                        onClick={() => void copy(message.rawX12, { withToast: true })}
                      >
                        <ClipboardCheckIcon className="size-4" />
                      </Button>
                      <Button
                        type="button"
                        size="icon-sm"
                        variant="ghost"
                        title="Download raw X12"
                        disabled={!message.rawX12}
                        onClick={() =>
                          downloadTextFile(buildX12Filename(message), message.rawX12, "text/plain")
                        }
                      >
                        <DownloadIcon className="size-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}

function MessageDetailInspector({
  messageId,
  open,
  onOpenChange,
}: {
  messageId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { copy } = useCopyToClipboard();
  const messageQuery = useEDIMessageDetailQuery(open ? messageId : "");
  const message = messageQuery.data;
  const diagnostics = useMemo(() => message?.validationErrors ?? [], [message?.validationErrors]);
  const diagnosticGroups = useMemo(() => groupDiagnostics(diagnostics), [diagnostics]);
  const segments = useMemo(
    () => parseX12Segments(message?.rawX12 ?? "", message?.partnerDocumentProfile?.envelope),
    [message?.partnerDocumentProfile?.envelope, message?.rawX12],
  );
  const payloadJson = useMemo(
    () => JSON.stringify(message?.payloadSnapshot ?? {}, null, 2),
    [message?.payloadSnapshot],
  );
  const editorTheme = useEditorTheme();

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-[min(1180px,calc(100vw-2rem))] gap-0 p-0 sm:max-w-none">
        <SheetHeader className="border-b">
          <SheetTitle className="flex items-center gap-2">
            <DatabaseIcon className="size-4 text-muted-foreground" />
            Message {message?.transactionControlNumber ?? messageId}
          </SheetTitle>
          <SheetDescription>
            {message
              ? `${message.transactionSet} ${message.direction} generated ${formatUnix(message.generatedAt)}`
              : "Loading message details."}
          </SheetDescription>
        </SheetHeader>
        {!message ? (
          <div className="p-4 text-sm text-muted-foreground">
            {messageQuery.isLoading ? "Loading message detail." : "Message detail unavailable."}
          </div>
        ) : (
          <Tabs
            defaultValue="overview"
            className="grid min-h-0 flex-1 grid-rows-[auto_minmax(0,1fr)] gap-0"
          >
            <div className="overflow-x-auto p-3 pb-0">
              <TabsList className="grid w-max grid-cols-7">
                <TabsTrigger value="overview">Overview</TabsTrigger>
                <TabsTrigger value="controls">Control Numbers</TabsTrigger>
                <TabsTrigger value="raw">Raw X12</TabsTrigger>
                <TabsTrigger value="segments">Segment Tree</TabsTrigger>
                <TabsTrigger value="diagnostics">Diagnostics</TabsTrigger>
                <TabsTrigger value="payload">Payload</TabsTrigger>
                <TabsTrigger value="provenance">Provenance</TabsTrigger>
              </TabsList>
            </div>
            <TabsContent value="overview" className="min-h-0 overflow-auto p-3">
              <InspectorGrid
                rows={[
                  ["Status", message.status],
                  [
                    "Partner",
                    message.partner?.name ?? message.partnerDocumentProfile?.partner?.name ?? "-",
                  ],
                  [
                    "Document Type",
                    message.documentType?.name ??
                      message.partnerDocumentProfile?.documentType?.name ??
                      "-",
                  ],
                  ["Transaction Set", message.transactionSet],
                  ["Direction", message.direction],
                  ["X12 Version", message.x12Version],
                  ["Generated At", formatUnix(message.generatedAt)],
                  ["Generated By ID", message.generatedById ?? "-"],
                  ["Shipment ID", message.shipmentId ?? "-"],
                  ["Transfer ID", message.transferId ?? "-"],
                  [
                    "Profile",
                    message.partnerDocumentProfile?.name ?? message.partnerDocumentProfileId,
                  ],
                  [
                    "Template",
                    message.template?.name ??
                      message.partnerDocumentProfile?.template?.name ??
                      message.templateId,
                  ],
                  ["Template Version", versionLabel(message)],
                  ["Validation Mode", message.validationMode],
                ]}
              />
            </TabsContent>
            <TabsContent value="controls" className="min-h-0 overflow-auto p-3">
              <div className="mb-3 flex items-center gap-2">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => void copy(controlNumberText(message), { withToast: true })}
                >
                  <CopyIcon className="size-4" />
                  Copy
                </Button>
              </div>
              <InspectorGrid
                rows={[
                  ["Interchange Control Number", message.interchangeControlNumber],
                  ["Group Control Number", message.groupControlNumber],
                  ["Transaction Control Number", message.transactionControlNumber],
                  ["Segment Count", String(message.segmentCount)],
                ]}
              />
            </TabsContent>
            <TabsContent value="raw" className="grid min-h-0 grid-rows-[auto_minmax(0,1fr)] p-3">
              <RawX12Viewer message={message} editorTheme={editorTheme} onCopy={copy} />
            </TabsContent>
            <TabsContent value="segments" className="min-h-0 overflow-auto p-3">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-16">#</TableHead>
                    <TableHead className="w-24">Segment</TableHead>
                    <TableHead>Elements</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {segments.map((segment) => (
                    <TableRow key={`${segment.index}-${segment.raw}`}>
                      <TableCell className="font-mono text-xs">{segment.index}</TableCell>
                      <TableCell className="font-mono font-medium">{segment.segmentId}</TableCell>
                      <TableCell className="font-mono text-xs">
                        {segment.elements.length > 0 ? segment.elements.join(" | ") : "-"}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TabsContent>
            <TabsContent value="diagnostics" className="min-h-0 overflow-auto p-3">
              {diagnosticGroups.length === 0 ? (
                <div className="text-sm text-muted-foreground">No diagnostics.</div>
              ) : (
                <div className="space-y-2">
                  {diagnosticGroups.map((group) => (
                    <div key={group.key} className="rounded-md border p-3">
                      <div className="flex flex-wrap items-center gap-2">
                        <Badge variant={group.severity === "Error" ? "inactive" : "warning"}>
                          {group.severity}
                        </Badge>
                        <span className="font-mono text-xs">
                          {group.segmentId || "Payload"}
                          {group.elementPosition ? `:${group.elementPosition}` : ""}
                        </span>
                        <span className="font-mono text-xs text-muted-foreground">
                          {group.code}
                        </span>
                        {group.path ? (
                          <span className="font-mono text-xs text-muted-foreground">
                            {group.path}
                          </span>
                        ) : null}
                      </div>
                      <div className="mt-2 space-y-2">
                        {group.diagnostics.map((diagnostic) => (
                          <div key={diagnosticKey(diagnostic)} className="text-sm">
                            <div>{diagnostic.message}</div>
                            {diagnostic.suggestedFix ? (
                              <div className="text-xs text-muted-foreground">
                                {diagnostic.suggestedFix}
                              </div>
                            ) : null}
                          </div>
                        ))}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </TabsContent>
            <TabsContent
              value="payload"
              className="grid min-h-0 grid-rows-[auto_minmax(0,1fr)] p-3"
            >
              <div className="mb-2 flex items-center gap-2">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => void copy(payloadJson, { withToast: true })}
                >
                  <CopyIcon className="size-4" />
                  Copy
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() =>
                    downloadJsonFile(buildMessageJsonFilename(message), message.payloadSnapshot)
                  }
                >
                  <FileJsonIcon className="size-4" />
                  Download
                </Button>
              </div>
              <CodeMirror
                value={payloadJson}
                editable={false}
                basicSetup={{ lineNumbers: true, foldGutter: true }}
                extensions={[json(), EditorView.lineWrapping]}
                theme={editorTheme}
                className="min-h-0 overflow-auto rounded-md border text-xs"
              />
            </TabsContent>
            <TabsContent value="provenance" className="min-h-0 overflow-auto p-3">
              <InspectorGrid
                rows={[
                  ["Profile ID", message.partnerDocumentProfileId],
                  ["Profile Name", message.partnerDocumentProfile?.name ?? "-"],
                  ["Template ID", message.templateId],
                  [
                    "Template Name",
                    message.template?.name ?? message.partnerDocumentProfile?.template?.name ?? "-",
                  ],
                  ["Template Version ID", message.templateVersionId],
                  ["Template Version", versionLabel(message)],
                  ["Template Version Status", message.templateVersion?.status ?? "-"],
                  ["Script Libraries", scriptLibraryLabel(message)],
                  ["Source X12 Version", message.templateVersion?.x12Version ?? message.x12Version],
                  ["Validation Mode", message.validationMode],
                ]}
              />
            </TabsContent>
          </Tabs>
        )}
      </SheetContent>
    </Sheet>
  );
}

function RawX12Viewer({
  message,
  editorTheme,
  onCopy,
}: {
  message: EDIMessage;
  editorTheme: ReturnType<typeof useEditorTheme>;
  onCopy: ReturnType<typeof useCopyToClipboard>["copy"];
}) {
  const [wrap, setWrap] = useState(true);
  const displayRawX12 = useMemo(
    () => formatRawX12Display(message.rawX12, message.partnerDocumentProfile?.envelope),
    [message.partnerDocumentProfile?.envelope, message.rawX12],
  );

  return (
    <>
      <div className="mb-2 flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Button
            type="button"
            variant="outline"
            onClick={() => void onCopy(message.rawX12, { withToast: true })}
          >
            <CopyIcon className="size-4" />
            Copy
          </Button>
          <Button
            type="button"
            variant="outline"
            onClick={() =>
              downloadTextFile(buildX12Filename(message), message.rawX12, "text/plain")
            }
          >
            <DownloadIcon className="size-4" />
            Download
          </Button>
        </div>
        <label className="flex items-center gap-2 text-xs text-muted-foreground">
          Wrap
          <Switch checked={wrap} onCheckedChange={setWrap} />
        </label>
      </div>
      <CodeMirror
        value={displayRawX12}
        editable={false}
        basicSetup={{ lineNumbers: true, foldGutter: false }}
        extensions={wrap ? [EditorView.lineWrapping] : []}
        theme={editorTheme}
        className="min-h-0 overflow-auto rounded-md border text-xs"
      />
    </>
  );
}

function InspectorGrid({ rows }: { rows: Array<[string, string]> }) {
  return (
    <div className="grid grid-cols-2 gap-2 lg:grid-cols-3">
      {rows.map(([label, value]) => (
        <div key={label} className="rounded-md border p-3">
          <div className="text-xs text-muted-foreground">{label}</div>
          <div className="mt-1 font-mono text-sm wrap-break-word">{value || "-"}</div>
        </div>
      ))}
    </div>
  );
}

function controlNumberText(message: EDIMessage) {
  return [
    `ISA: ${message.interchangeControlNumber}`,
    `GS: ${message.groupControlNumber}`,
    `ST: ${message.transactionControlNumber}`,
  ].join("\n");
}

function versionLabel(message: EDIMessage) {
  if (message.templateVersion?.versionNumber) {
    return `v${message.templateVersion.versionNumber}`;
  }
  if (message.partnerDocumentProfile?.templateVersion?.versionNumber) {
    return `v${message.partnerDocumentProfile.templateVersion.versionNumber}`;
  }
  return message.templateVersionId;
}

function scriptLibraryLabel(message: EDIMessage) {
  const libraries = message.templateVersion?.scriptLibraries ?? [];
  if (libraries.length === 0) return "-";
  return libraries
    .map((library) => {
      const functions =
        library.functionNames.length > 0 ? ` (${library.functionNames.join(", ")})` : "";
      return `${library.name}${functions}`;
    })
    .join("; ");
}

export default DocumentPreviewArchiveTab;
