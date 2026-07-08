import {
  ControlledEDIDocumentProfileAutocompleteField,
  ControlledEDIPartnerAutocompleteField,
  ControlledEDITemplateAutocompleteField,
} from "@/components/autocomplete-fields";
import { DocumentSourceControls } from "@/components/edi/document-source-controls";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { ScrollArea } from "@/components/ui/scroll-area";
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
import { downloadTextFile } from "@/lib/utils";
import type {
  EDIMessage,
  EDIPartnerDocumentProfile,
  UpsertEDIPartnerDocumentProfileRequest,
} from "@/types/edi";
import {
  ClipboardCheckIcon,
  CopyIcon,
  DatabaseIcon,
  DownloadIcon,
  EyeIcon,
  FileCode2Icon,
  PlayIcon,
  RefreshCwIcon,
  ShieldCheckIcon
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import { formatUnix } from "../../edi-display-utils";
import { ControlledSelectField } from "../components/designer-fields";
import {
  InputBlock,
  PanelHeader,
  PreviewPane,
  TextareaBlock,
  parsePayload,
  parseSettings,
  profileToDraft,
} from "../components/designer-shared";
import { useDocumentArchiveUrlState } from "../hooks/use-edi-designer-url-state";
import {
  useGenerateEDIDocumentMutation,
  useInvalidateEDIDocumentProfiles,
  useInvalidateEDIMessageArchive,
  usePreviewEDIDocumentMutation,
  useSaveEDIDocumentProfileMutation,
} from "../hooks/use-edi-document-mutations";
import { useEDIDocumentArchiveQueries } from "../hooks/use-edi-document-queries";
import { controlNumberText } from "../inspector/components/control-numbers-tab";
import type { InspectorTab } from "../inspector/components/inspector-tabs";
import MessageInspectorSheet from "../inspector/message-inspector-sheet";
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
  resolveSelectedDocumentTemplateId,
} from "../utils/edi-designer-utils";
import { buildArchiveMessagesQueryString, buildX12Filename } from "../utils/edi-message-utils";

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
        inspectorTab,
        inspectorSegment,
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
  const templateFilters = useMemo(
    () => ({
      limit: 100,
      transactionSet: archiveTransactionSet,
      direction: archiveDirection,
    }),
    [archiveDirection, archiveTransactionSet],
  );
  const { profilesQuery, templatesQuery, messagesQuery } = useEDIDocumentArchiveQueries({
    messagesQueryString,
    profilesQueryString: profileContextQueryString,
    templateFilters,
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
  const selectedPartnerLabel =
    selectedDocumentProfile?.partner?.name ?? selectedProfile?.partner?.name ?? partnerId;
  const documentContextLabel = [
    selectedPartnerLabel || "No partner selected",
    selectedDocumentProfile?.name ??
    (!!partnerId && !profileId ? "New profile draft" : "No profile"),
    activeTemplate?.name ?? "No template",
    activeTemplate?.activeVersion?.versionNumber
      ? `v${activeTemplate.activeVersion.versionNumber}`
      : "no active version",
  ].join(" / ");
  const sourceContext = resolveEDIDocumentSourceContext({
    profile: selectedDocumentProfile,
    template: activeTemplate,
    fallbackTransactionSet: archiveTransactionSet || activeTemplate?.transactionSet,
    fallbackDirection: archiveDirection || activeTemplate?.direction,
  });
  const sourceTransactionSet = sourceContext.transactionSet;
  const sourceDirection = sourceContext.direction;
  const hasSourceValue = hasEDIDocumentSourceValue(sourceValues, sourceTransactionSet);
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
    <div className="grid h-full min-h-0 grid-cols-[360px_minmax(0,1fr)] gap-3 overflow-hidden">
      <aside className="flex min-h-0 flex-col overflow-hidden rounded-md border bg-background">
        <PanelHeader icon={<ShieldCheckIcon />} title="Document Profile" />
        <ScrollArea className="min-h-0 flex-1" viewportClassName="min-h-0">
          <div className="flex flex-col gap-3 p-3">
            <ControlledEDIPartnerAutocompleteField
              value={partnerId}
              placeholder="Select a partner..."
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
            <ControlledEDIDocumentProfileAutocompleteField
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
              noResultsMessage="No document profiles match this partner and document context."
            />
            {/* {selectedPartnerHasNoProfiles && (
              <Alert variant="info" className="py-2 text-xs">
                <InfoIcon className="size-4" />
                <AlertDescription className="text-xs">
                  No document profiles exist for this partner yet. Fill the profile details below
                  and click Save Profile.
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
            )} */}
            <InputBlock
              label="Profile Name"
              value={profileDraft.name}
              onChange={(name) => setProfileDraft((current) => ({ ...current, name }))}
            />
            <ControlledEDITemplateAutocompleteField
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
            {sourceTransactionSet === "214" && sourceDirection === "Outbound" && (
              <ServiceFailure214SettingsEditor
                rawSettings={rawPartnerSettings}
                onChange={setRawPartnerSettings}
              />
            )}
            <TextareaBlock
              label="Partner Settings"
              description="Raw partner settings for advanced profile configuration."
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
        </ScrollArea>
      </aside>
      <main className="min-h-0 overflow-hidden rounded-md border bg-background">
        <Tabs
          defaultValue="preview"
          className="grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)] gap-0"
        >
          <div className="grid gap-2 border-b py-2">
            <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border">
              <TabsList
                variant="underline"
                className="grid w-fit grid-cols-2 border-b border-border px-2"
              >
                <TabsTrigger value="preview">
                  <FileCode2Icon data-icon="inline-start" />
                  Preview
                </TabsTrigger>
                <TabsTrigger value="archive">
                  <DatabaseIcon data-icon="inline-start" />
                  Archive
                </TabsTrigger>
              </TabsList>
              <div className="min-w-0 truncate text-xs text-muted-foreground">
                {documentContextLabel}
              </div>
            </div>
            <div className="flex flex-row items-start justify-between gap-2 px-2">
              <DocumentSourceControls
                transactionSet={sourceTransactionSet}
                values={sourceValues}
                onChange={setSourceValue}
                layout="toolbar"
              />
              <div className="flex flex-wrap items-center gap-2 pt-4">
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
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
                  Preview provisional controls
                </Button>
                <Button
                  type="button"
                  size="sm"
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
                  Generate archive message
                </Button>
              </div>
            </div>
          </div>
          <TabsContent value="preview" className="m-0 min-h-0 overflow-hidden">
            <PreviewPane preview={previewMutation.data} isLoading={previewMutation.isPending} />
          </TabsContent>
          <TabsContent value="archive" className="m-0 min-h-0 overflow-hidden">
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
              onOpenMessage={(messageId) => {
                void setArchiveUrlState({ inspectorTab: "overview", inspectorSegment: 1 });
                void setInspectorMessageId(messageId);
              }}
            />
          </TabsContent>
        </Tabs>
      </main>
      <MessageInspectorSheet
        messageId={inspectorMessageId}
        open={!!inspectorMessageId}
        selectedTab={inspectorTab as InspectorTab}
        selectedSegmentIndex={inspectorSegment}
        onOpenChange={(open) => {
          if (!open) void setInspectorMessageId("");
        }}
        onTabChange={(tab) => void setArchiveUrlState({ inspectorTab: tab })}
        onSelectSegment={(segmentIndex) =>
          void setArchiveUrlState({ inspectorSegment: segmentIndex, inspectorTab: "segments" })
        }
      />
    </div>
  );
}

function ServiceFailure214SettingsEditor({
  rawSettings,
  onChange,
}: {
  rawSettings: string;
  onChange: (value: string) => void;
}) {
  const root = parseRawSettings(rawSettings);
  const settings = serviceFailure214Settings(root);
  const updateSettings = (patch: Record<string, unknown>) => {
    onChange(
      JSON.stringify(
        {
          ...root,
          serviceFailure214: {
            ...settings,
            ...patch,
          },
        },
        null,
        2,
      ),
    );
  };

  return (
    <div className="space-y-3 rounded-md border p-3">
      <div className="text-xs font-medium text-muted-foreground">Service Failure 214</div>
      <div className="grid grid-cols-2 gap-2">
        {serviceFailure214BooleanFields.map((field) => (
          <label
            key={field.key}
            className="flex min-h-8 items-center gap-2 rounded border px-2 text-xs"
          >
            <Checkbox
              checked={Boolean(settings[field.key])}
              onCheckedChange={(checked) => updateSettings({ [field.key]: checked === true })}
            />
            <span className="truncate">{field.label}</span>
          </label>
        ))}
      </div>
      <div className="grid grid-cols-2 gap-2">
        <InputBlock
          label="Status Code"
          value={settingString(settings.statusCode)}
          onChange={(statusCode) => updateSettings({ statusCode: statusCode.trim().toUpperCase() })}
        />
        <InputBlock
          label="Time Code"
          value={settingString(settings.timeCode)}
          onChange={(timeCode) => updateSettings({ timeCode: timeCode.trim().toUpperCase() })}
        />
      </div>
      <InputBlock
        label="Accepted Reason Codes"
        value={settingStringArray(settings.acceptedReasonCodes).join(", ")}
        onChange={(value) =>
          updateSettings({
            acceptedReasonCodes: value
              .split(",")
              .map((item) => item.trim().toUpperCase())
              .filter(Boolean),
          })
        }
      />
    </div>
  );
}

const serviceFailure214BooleanFields = [
  { key: "enabled", label: "Enabled" },
  { key: "sendOnReviewed", label: "Send Reviewed" },
  { key: "sendOnResolved", label: "Send Resolved" },
  { key: "mandatoryOnReviewed", label: "Mandatory Reviewed" },
  { key: "mandatoryOnResolved", label: "Mandatory Resolved" },
  { key: "requireStatusReasonCode", label: "Require Reason" },
  { key: "requireLocation", label: "Require Location" },
  { key: "requireLocationName", label: "Require Location Name" },
  { key: "requireCityState", label: "Require City/State" },
  { key: "requirePostalCode", label: "Require Postal" },
  { key: "requireTimeCode", label: "Require Time Code" },
  { key: "requireStop", label: "Require Stop" },
  { key: "requireProNumber", label: "Require PRO" },
  { key: "requireBol", label: "Require BOL" },
] as const;

function parseRawSettings(value: string): Record<string, unknown> {
  try {
    const parsed = JSON.parse(value) as unknown;
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
  } catch {
    return {};
  }
  return {};
}

function serviceFailure214Settings(root: Record<string, unknown>) {
  const value = root.serviceFailure214;
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as Record<string, unknown>;
  }
  return {};
}

function settingString(value: unknown) {
  return typeof value === "string" ? value : "";
}

function settingStringArray(value: unknown) {
  return Array.isArray(value)
    ? value.filter((item): item is string => typeof item === "string")
    : [];
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
        <ControlledEDIPartnerAutocompleteField
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
      <ScrollArea className="min-h-0" viewportClassName="min-h-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="min-w-36">Archived At</TableHead>
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
                  No archived generated messages match the current filters.
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
                      {message.status === "Generated" ? "Archived" : message.status}
                    </Badge>
                    {message.deliveryStatus && (
                      <div className="mt-1 text-2xs text-muted-foreground">
                        Delivery {message.deliveryStatus}
                      </div>
                    )}
                    {message.ackStatus && (
                      <div className="text-2xs text-muted-foreground">ACK {message.ackStatus}</div>
                    )}
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
      </ScrollArea>
    </div>
  );
}

export default DocumentPreviewArchiveTab;
