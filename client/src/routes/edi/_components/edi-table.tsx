import {
  EDIPartnerAutocompleteField,
  OrganizationAutocompleteField,
} from "@/components/autocomplete-fields";
import { DataTable } from "@/components/data-table/data-table";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
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
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useAuthStore } from "@/stores/auth-store";
import { usePermissionStore } from "@/stores/permission-store";
import type { DataTablePanelProps } from "@/types/data-table";
import type {
  CreateEDIConnectionRequest,
  EDICommunicationProfile,
  EDIConnection,
  EDIMappingProfileItem,
  EDIPartner,
  EDITransfer,
  UpsertEDICommunicationProfileRequest,
} from "@/types/edi";
import type { OrganizationSelectOption } from "@/types/organization";
import { Operation, Resource } from "@/types/permission";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  CheckIcon,
  KeyRoundIcon,
  RadioTowerIcon,
  ServerIcon,
  ShieldCheckIcon,
  Trash2Icon,
  XIcon,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useForm, useWatch, type Control } from "react-hook-form";
import { toast } from "sonner";
import { getCommunicationProfileColumns } from "./edi-communication-profile-columns";
import { DesignerWorkspace } from "./edi-designer-workspace";
import { formatUnix } from "./edi-display-utils";
import { getPartnerColumns } from "./edi-partner-columns";
import { TargetLookup } from "./edi-target-lookup";
import { getTransferColumns } from "./edi-transfer-columns";
import {
  acknowledgmentOptions,
  communicationProfileFormSchema,
  communicationProfileMethodOptions,
  createInternalPartnerPairSchema,
  environmentOptions,
  mappingEntityTypes,
  mdnModeOptions,
  profileStatusOptions,
  sftpAuthModeOptions,
  type CommunicationProfileFormValues,
  type CommunicationProfileMethod,
  type CreateInternalPartnerPairFormValues,
} from "./edi-schemas";
import { EDITransferReviewPanel } from "./panel/edi-transfer-review-panel";
import type { EDIPageKind } from "./edi-types";

export default function EdiTable({ kind }: { kind: EDIPageKind }) {
  return (
    <>
      {kind === "partners" && <PartnersWorkspace />}
      {kind === "communication-profiles" && <CommunicationProfilesWorkspace />}
      {kind === "mapping-profiles" && <MappingProfilesWorkspace />}
      {kind === "designer" && <DesignerWorkspace />}
      {(kind === "inbound" || kind === "outbound") && <TransfersWorkspace direction={kind} />}
    </>
  );
}

function PartnersWorkspace() {
  const columns = useMemo(() => getPartnerColumns(), []);

  return (
    <div className="flex flex-col gap-4">
      <PendingConnectionsPanel />
      <DataTable<EDIPartner>
        name="EDI Connection"
        link="/edi/partners/"
        queryKey="edi-partner-list"
        exportModelName="edi-partner"
        resource={Resource.EDI}
        columns={columns}
        TablePanel={PartnerPanel}
        preferDetailRowForEdit
      />
    </div>
  );
}

function PendingConnectionsPanel() {
  const queryClient = useQueryClient();
  const currentOrganizationId = useAuthStore((state) => state.user?.currentOrganizationId) ?? "";
  const [rejecting, setRejecting] = useState<EDIConnection | null>(null);
  const [reason, setReason] = useState("");
  const { data, isLoading } = useQuery(queries.edi.connections("?limit=25"));
  const pending = (data?.results ?? []).filter(
    (connection) =>
      connection.status === "PendingAcceptance" &&
      connection.targetOrganizationId === currentOrganizationId,
  );
  const acceptMutation = useApiMutation({
    mutationFn: (connectionId: string) => apiService.ediService.acceptConnection(connectionId),
    onSuccess: async () => {
      toast.success("EDI connection accepted");
      await invalidateEDIConnections(queryClient);
    },
    onError: () => toast.error("Failed to accept EDI connection"),
  });
  const rejectMutation = useApiMutation({
    mutationFn: (connection: EDIConnection) =>
      apiService.ediService.rejectConnection(connection.id, { reason }),
    onSuccess: async () => {
      toast.success("EDI connection rejected");
      setRejecting(null);
      setReason("");
      await invalidateEDIConnections(queryClient);
    },
    onError: () => toast.error("Failed to reject EDI connection"),
  });

  if (!isLoading && pending.length === 0) {
    return null;
  }

  return (
    <div className="rounded-md border bg-background">
      <div className="flex items-center justify-between gap-2 border-b px-3 py-2">
        <div>
          <div className="text-sm font-medium">Pending EDI connection requests</div>
          <div className="text-xs text-muted-foreground">
            Accepting creates reciprocal internal partners and communication profiles.
          </div>
        </div>
        <Badge variant="outline">{pending.length}</Badge>
      </div>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Requester</TableHead>
            <TableHead>Target</TableHead>
            <TableHead>Method</TableHead>
            <TableHead>Requested</TableHead>
            <TableHead />
          </TableRow>
        </TableHeader>
        <TableBody>
          {pending.map((connection) => (
            <TableRow key={connection.id}>
              <TableCell>
                {connection.sourceOrganization?.name ?? connection.sourceOrganizationId}
              </TableCell>
              <TableCell>
                {connection.targetOrganization?.name ?? connection.targetOrganizationId}
              </TableCell>
              <TableCell>{connection.method}</TableCell>
              <TableCell>{formatUnix(connection.requestedAt)}</TableCell>
              <TableCell className="text-right">
                <div className="flex justify-end gap-2">
                  <Button variant="outline" size="sm" onClick={() => setRejecting(connection)}>
                    <XIcon data-icon="inline-start" />
                    Reject
                  </Button>
                  <Button
                    size="sm"
                    isLoading={acceptMutation.isPending}
                    onClick={() => acceptMutation.mutate(connection.id)}
                  >
                    <CheckIcon data-icon="inline-start" />
                    Accept
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          ))}
          {isLoading && (
            <TableRow>
              <TableCell colSpan={5} className="h-16 text-center text-muted-foreground">
                Loading connection requests.
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
      <Sheet open={!!rejecting} onOpenChange={(open) => !open && setRejecting(null)}>
        <SheetContent>
          <SheetHeader>
            <SheetTitle>Reject EDI Connection</SheetTitle>
            <SheetDescription>
              {rejecting?.sourceOrganization?.name ?? rejecting?.id}
            </SheetDescription>
          </SheetHeader>
          <div className="px-4">
            <Input
              placeholder="Rejection reason"
              value={reason}
              onChange={(event) => setReason(event.target.value)}
            />
          </div>
          <SheetFooter>
            <Button variant="outline" onClick={() => setRejecting(null)}>
              Cancel
            </Button>
            <Button
              disabled={!reason.trim() || !rejecting}
              isLoading={rejectMutation.isPending}
              onClick={() => rejecting && rejectMutation.mutate(rejecting)}
            >
              Reject
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </div>
  );
}

function CommunicationProfilesWorkspace() {
  const columns = useMemo(() => getCommunicationProfileColumns(), []);

  return (
    <DataTable<EDICommunicationProfile>
      name="EDI Communication Profile"
      link="/edi/communication-profiles/"
      queryKey="edi-communication-profile-list"
      exportModelName="edi-communication-profile"
      resource={Resource.EDI}
      columns={columns}
      TablePanel={CommunicationProfilePanel}
      preferDetailRowForEdit
    />
  );
}

function CommunicationProfilePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<EDICommunicationProfile>) {
  const queryClient = useQueryClient();

  return (
    <CommunicationProfileFormSheet
      open={open}
      profile={mode === "edit" ? row : null}
      onOpenChange={onOpenChange}
      onSaved={async () => {
        onOpenChange(false);
        await queryClient.invalidateQueries({ queryKey: ["edi-communication-profile-list"] });
        await queryClient.invalidateQueries({ queryKey: queries.edi.communicationProfiles._def });
        await queryClient.invalidateQueries({ queryKey: queries.edi.partners._def });
        await queryClient.invalidateQueries({ queryKey: ["edi-partner-list"] });
      }}
    />
  );
}

function CommunicationProfileFormSheet({
  open,
  profile,
  onOpenChange,
  onSaved,
}: {
  open: boolean;
  profile: EDICommunicationProfile | null;
  onOpenChange: (open: boolean) => void;
  onSaved: () => Promise<void>;
}) {
  const form = useForm<CommunicationProfileFormValues>({
    resolver: zodResolver(communicationProfileFormSchema),
    defaultValues: getProfileFormDefaults(profile),
    mode: "onChange",
  });
  const { control, handleSubmit, reset } = form;
  const method = useWatch({ control, name: "method" });
  const authMode = useWatch({ control, name: "config.authMode" });
  const isEdit = !!profile;
  const mutation = useApiMutation({
    mutationFn: (values: CommunicationProfileFormValues) => {
      const request = toCommunicationProfileRequest(values, profile);
      return isEdit
        ? apiService.ediService.updateCommunicationProfile(profile.id, request)
        : apiService.ediService.createCommunicationProfile(request);
    },
    onSuccess: async () => {
      toast.success(isEdit ? "Communication profile updated" : "Communication profile created");
      await onSaved();
    },
    onError: () => toast.error("Failed to save communication profile"),
  });

  useEffect(() => {
    if (!open) return;
    reset(getProfileFormDefaults(profile));
  }, [open, profile, reset]);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-4xl">
        <SheetHeader>
          <SheetTitle>
            {isEdit ? "Edit Communication Profile" : "New Communication Profile"}
          </SheetTitle>
          <SheetDescription>
            Configure the transport profile and envelope values used for this organization.
          </SheetDescription>
        </SheetHeader>
        <Form
          id="edi-profile-form"
          className="min-h-0 overflow-auto px-4"
          onSubmit={(event) => {
            event.stopPropagation();
            void handleSubmit((values) => mutation.mutate(values))(event);
          }}
        >
          <Tabs defaultValue="overview" className="gap-3">
            <TabsList className="grid w-full grid-cols-4">
              <TabsTrigger value="overview">
                <ServerIcon data-icon="inline-start" />
                Overview
              </TabsTrigger>
              <TabsTrigger value="transport">
                <RadioTowerIcon data-icon="inline-start" />
                Transport
              </TabsTrigger>
              <TabsTrigger value="envelope">
                <ShieldCheckIcon data-icon="inline-start" />
                Envelope
              </TabsTrigger>
              <TabsTrigger value="secrets">
                <KeyRoundIcon data-icon="inline-start" />
                Secrets
              </TabsTrigger>
            </TabsList>
            <TabsContent value="overview" className="space-y-3">
              <FormSection title="Profile Identity" className="rounded-md border bg-muted/20 p-3">
                <FormGroup cols={2}>
                  <FormControl>
                    <InputField
                      control={control}
                      name="name"
                      label="Name"
                      placeholder="Profile name"
                      rules={{ required: true }}
                    />
                  </FormControl>
                  <FormControl>
                    <SelectField
                      control={control}
                      name="method"
                      label="Method"
                      options={communicationProfileMethodOptions}
                      rules={{ required: true }}
                    />
                  </FormControl>
                  <FormControl>
                    <SelectField
                      control={control}
                      name="status"
                      label="Status"
                      options={profileStatusOptions}
                      rules={{ required: true }}
                    />
                  </FormControl>
                  <FormControl>
                    <InputField
                      control={control}
                      name="ediPartnerId"
                      label="Partner ID"
                      placeholder="Optional partner ID"
                    />
                  </FormControl>
                  <FormControl cols="full">
                    <TextareaField
                      control={control}
                      name="description"
                      label="Description"
                      placeholder="Operational notes for this profile"
                    />
                  </FormControl>
                </FormGroup>
              </FormSection>
              {method === "Internal" && (
                <FormSection title="Internal Routing" className="rounded-md border bg-muted/20 p-3">
                  <FormGroup cols={2}>
                    <FormControl>
                      <InputField
                        control={control}
                        name="ediConnectionId"
                        label="Connection ID"
                        placeholder="Optional connection ID"
                      />
                    </FormControl>
                    <FormControl>
                      <InputField
                        control={control}
                        name="config.connectedOrganizationId"
                        label="Connected Organization ID"
                        placeholder="Organization ID"
                      />
                    </FormControl>
                  </FormGroup>
                </FormSection>
              )}
            </TabsContent>
            <TabsContent value="transport" className="space-y-3">
              <TransportProfileFields control={control} method={method} authMode={authMode} />
            </TabsContent>
            <TabsContent value="envelope" className="space-y-3">
              {method === "Internal" ? (
                <EmptyProfileSection message="Internal profiles use organization routing and do not require X12 interchange identifiers." />
              ) : (
                <X12EnvelopeFields control={control} />
              )}
            </TabsContent>
            <TabsContent value="secrets" className="space-y-3">
              <SecretProfileFields
                control={control}
                method={method}
                profile={profile}
                authMode={authMode}
              />
            </TabsContent>
          </Tabs>
        </Form>
        <SheetFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="submit" form="edi-profile-form" isLoading={mutation.isPending}>
            Save Profile
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

function TransportProfileFields({
  control,
  method,
  authMode,
}: {
  control: Control<CommunicationProfileFormValues>;
  method: CommunicationProfileMethod;
  authMode: string;
}) {
  if (method === "Internal") {
    return (
      <EmptyProfileSection message="Internal communication is enabled through accepted organization connections." />
    );
  }

  if (method === "AS2") {
    return (
      <>
        <FormSection title="AS2 Identifiers" className="rounded-md border bg-muted/20 p-3">
          <FormGroup cols={2}>
            <FormControl>
              <InputField
                control={control}
                name="config.localAS2Id"
                label="Local AS2 ID"
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="config.partnerAS2Id"
                label="Partner AS2 ID"
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl cols="full">
              <InputField
                control={control}
                name="config.endpointUrl"
                label="Endpoint URL"
                rules={{ required: true }}
              />
            </FormControl>
          </FormGroup>
        </FormSection>
        <FormSection title="Security and MDN" className="rounded-md border bg-muted/20 p-3">
          <FormGroup cols={2}>
            <FormControl>
              <InputField
                control={control}
                name="config.signingCertificateRef"
                label="Signing Certificate Ref"
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="config.encryptionCertificateRef"
                label="Encryption Certificate Ref"
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                name="config.mdnMode"
                label="MDN Mode"
                options={mdnModeOptions}
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl>
              <InputField control={control} name="config.mdnUrl" label="Async MDN URL" />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="config.signingAlgorithm"
                label="Signing Algorithm"
                placeholder="sha256"
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="config.encryptionAlgorithm"
                label="Encryption Algorithm"
                placeholder="aes256-cbc"
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="config.compressionAlgorithm"
                label="Compression"
                placeholder="zlib"
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="config.basicAuthUsername"
                label="Basic Auth Username"
              />
            </FormControl>
          </FormGroup>
        </FormSection>
      </>
    );
  }

  if (method === "SFTP") {
    return (
      <>
        <FormSection title="SFTP Endpoint" className="rounded-md border bg-muted/20 p-3">
          <FormGroup cols={2}>
            <FormControl>
              <InputField
                control={control}
                name="config.host"
                label="Host"
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="config.port"
                label="Port"
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="config.username"
                label="Username"
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                name="config.authMode"
                label="Authentication"
                options={sftpAuthModeOptions}
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl cols="full">
              <TextareaField
                control={control}
                name="config.knownHostKey"
                label="Known Host Key"
                rules={{ required: true }}
              />
            </FormControl>
          </FormGroup>
        </FormSection>
        <FormSection title="Directories" className="rounded-md border bg-muted/20 p-3">
          <FormGroup cols={2}>
            <FormControl>
              <InputField
                control={control}
                name="config.inboundDirectory"
                label="Inbound Directory"
                placeholder="/inbound"
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="config.outboundDirectory"
                label="Outbound Directory"
                placeholder="/outbound"
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="config.archiveDirectory"
                label="Archive Directory"
                placeholder="/archive"
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="config.fileNamingPattern"
                label="File Naming Pattern"
                placeholder="{partner}-{timestamp}.edi"
              />
            </FormControl>
          </FormGroup>
        </FormSection>
        <EmptyProfileSection
          message={`Save a ${authMode === "password" ? "password" : "private key"} in the Secrets tab before activating this profile.`}
        />
      </>
    );
  }

  return (
    <FormSection title="VAN Mailbox" className="rounded-md border bg-muted/20 p-3">
      <FormGroup cols={2}>
        <FormControl>
          <InputField
            control={control}
            name="config.providerName"
            label="Provider Name"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.mailboxId"
            label="Mailbox ID"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <InputField control={control} name="config.accountId" label="Account ID" />
        </FormControl>
        <FormControl>
          <InputField control={control} name="config.endpoint" label="Endpoint" />
        </FormControl>
        <FormControl>
          <InputField control={control} name="config.senderMailboxId" label="Sender Mailbox ID" />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.receiverMailboxId"
            label="Receiver Mailbox ID"
          />
        </FormControl>
        <FormControl>
          <InputField control={control} name="config.isaRoutingId" label="ISA Routing ID" />
        </FormControl>
        <FormControl>
          <InputField control={control} name="config.gsRoutingId" label="GS Routing ID" />
        </FormControl>
        <FormControl cols="full">
          <InputField control={control} name="config.contactEmail" label="Contact Email" />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}

function X12EnvelopeFields({ control }: { control: Control<CommunicationProfileFormValues> }) {
  return (
    <FormSection title="X12 Envelope" className="rounded-md border bg-muted/20 p-3">
      <FormGroup cols={2}>
        <FormControl>
          <InputField
            control={control}
            name="config.isaSenderQualifier"
            label="ISA Sender Qualifier"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.isaSenderId"
            label="ISA Sender ID"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.isaReceiverQualifier"
            label="ISA Receiver Qualifier"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.isaReceiverId"
            label="ISA Receiver ID"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.gsSenderId"
            label="GS Sender ID"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.gsReceiverId"
            label="GS Receiver ID"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.x12Version"
            label="X12 Version"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="config.environment"
            label="Environment"
            options={environmentOptions}
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl cols="full">
          <SelectField
            control={control}
            name="config.acknowledgmentPreference"
            label="Acknowledgment Preference"
            options={acknowledgmentOptions}
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}

function SecretProfileFields({
  control,
  method,
  profile,
  authMode,
}: {
  control: Control<CommunicationProfileFormValues>;
  method: CommunicationProfileMethod;
  profile: EDICommunicationProfile | null;
  authMode: string;
}) {
  if (method === "Internal") {
    return <EmptyProfileSection message="Internal profiles do not store external credentials." />;
  }

  return (
    <div className="grid gap-3 xl:grid-cols-2">
      {profile && profile.secretState.length > 0 && (
        <div className="rounded-md border bg-muted/20 p-3">
          <div className="mb-2 text-sm font-medium">Saved Secrets</div>
          <div className="flex flex-wrap gap-1.5">
            {profile.secretState.map((secret) => (
              <Badge key={secret.key} variant="secondary">
                {secret.key} saved
              </Badge>
            ))}
          </div>
        </div>
      )}
      <FormSection title="Secret Values" className="rounded-md border bg-muted/20 p-3">
        <FormGroup cols={2}>
          {method === "AS2" && (
            <FormControl>
              <SensitiveField
                control={control}
                name="secrets.basicAuthPassword"
                label="Basic Auth Password"
                description="Leave blank to keep the saved value."
              />
            </FormControl>
          )}
          {method === "SFTP" && authMode === "password" && (
            <FormControl>
              <SensitiveField
                control={control}
                name="secrets.password"
                label="SFTP Password"
                description="Leave blank to keep the saved value."
              />
            </FormControl>
          )}
          {method === "SFTP" && authMode !== "password" && (
            <FormControl cols="full">
              <TextareaField
                control={control}
                name="secrets.privateKey"
                label="Private Key"
                description="Leave blank to keep the saved value."
              />
            </FormControl>
          )}
          {method === "VAN" && (
            <>
              <FormControl>
                <SensitiveField
                  control={control}
                  name="secrets.credential"
                  label="Credential"
                  description="Leave blank to keep the saved value."
                />
              </FormControl>
              <FormControl>
                <SensitiveField
                  control={control}
                  name="secrets.token"
                  label="Token"
                  description="Leave blank to keep the saved value."
                />
              </FormControl>
            </>
          )}
        </FormGroup>
      </FormSection>
    </div>
  );
}

function EmptyProfileSection({ message }: { message: string }) {
  return (
    <div className="rounded-md border border-dashed bg-muted/20 px-3 py-6 text-center text-sm text-muted-foreground">
      {message}
    </div>
  );
}

function getProfileFormDefaults(
  profile: EDICommunicationProfile | null,
): CommunicationProfileFormValues {
  const config = profile?.config ?? {};
  return {
    ediConnectionId: profile?.ediConnectionId ?? "",
    ediPartnerId: profile?.ediPartnerId ?? "",
    method: (profile?.method ?? "Internal") as CommunicationProfileMethod,
    status: profile?.status ?? "Active",
    name: profile?.name ?? "",
    description: profile?.description ?? "",
    config: {
      connectedOrganizationId: stringConfig(config, "connectedOrganizationId"),
      localAS2Id: stringConfig(config, "localAS2Id"),
      partnerAS2Id: stringConfig(config, "partnerAS2Id"),
      endpointUrl: stringConfig(config, "endpointUrl"),
      signingCertificateRef: stringConfig(config, "signingCertificateRef"),
      encryptionCertificateRef: stringConfig(config, "encryptionCertificateRef"),
      mdnMode: stringConfig(config, "mdnMode", "sync"),
      mdnUrl: stringConfig(config, "mdnUrl"),
      compressionAlgorithm: stringConfig(config, "compressionAlgorithm"),
      signingAlgorithm: stringConfig(config, "signingAlgorithm", "sha256"),
      encryptionAlgorithm: stringConfig(config, "encryptionAlgorithm", "aes256-cbc"),
      basicAuthUsername: stringConfig(config, "basicAuthUsername"),
      host: stringConfig(config, "host"),
      port: stringConfig(config, "port", "22"),
      username: stringConfig(config, "username"),
      authMode: stringConfig(config, "authMode", "privateKey"),
      inboundDirectory: stringConfig(config, "inboundDirectory"),
      outboundDirectory: stringConfig(config, "outboundDirectory"),
      archiveDirectory: stringConfig(config, "archiveDirectory"),
      fileNamingPattern: stringConfig(config, "fileNamingPattern", "{partner}-{timestamp}.edi"),
      knownHostKey: stringConfig(config, "knownHostKey"),
      providerName: stringConfig(config, "providerName"),
      mailboxId: stringConfig(config, "mailboxId"),
      accountId: stringConfig(config, "accountId"),
      senderMailboxId: stringConfig(config, "senderMailboxId"),
      receiverMailboxId: stringConfig(config, "receiverMailboxId"),
      isaRoutingId: stringConfig(config, "isaRoutingId"),
      gsRoutingId: stringConfig(config, "gsRoutingId"),
      endpoint: stringConfig(config, "endpoint"),
      contactEmail: stringConfig(config, "contactEmail"),
      isaSenderQualifier: stringConfig(config, "isaSenderQualifier"),
      isaSenderId: stringConfig(config, "isaSenderId"),
      isaReceiverQualifier: stringConfig(config, "isaReceiverQualifier"),
      isaReceiverId: stringConfig(config, "isaReceiverId"),
      gsSenderId: stringConfig(config, "gsSenderId"),
      gsReceiverId: stringConfig(config, "gsReceiverId"),
      x12Version: stringConfig(config, "x12Version", "004010"),
      environment: stringConfig(config, "environment", "test"),
      acknowledgmentPreference: stringConfig(config, "acknowledgmentPreference", "997"),
    },
    secrets: {
      basicAuthPassword: "",
      password: "",
      privateKey: "",
      credential: "",
      token: "",
    },
  };
}

function toCommunicationProfileRequest(
  values: CommunicationProfileFormValues,
  profile: EDICommunicationProfile | null,
): UpsertEDICommunicationProfileRequest {
  return {
    ediConnectionId: emptyToUndefined(values.ediConnectionId),
    ediPartnerId: emptyToUndefined(values.ediPartnerId),
    method: values.method,
    status: values.status,
    name: values.name,
    description: values.description,
    config: compactProfileConfig(values.method, values.config),
    secrets: compactSecrets(values.secrets),
    version: profile?.version,
  };
}

function compactProfileConfig(
  method: CommunicationProfileMethod,
  config: CommunicationProfileFormValues["config"],
): Record<string, unknown> {
  const keysByMethod: Record<
    CommunicationProfileMethod,
    Array<keyof CommunicationProfileFormValues["config"]>
  > = {
    Internal: ["connectedOrganizationId"],
    AS2: [
      "localAS2Id",
      "partnerAS2Id",
      "endpointUrl",
      "signingCertificateRef",
      "encryptionCertificateRef",
      "mdnMode",
      "mdnUrl",
      "compressionAlgorithm",
      "signingAlgorithm",
      "encryptionAlgorithm",
      "basicAuthUsername",
      ...x12ProfileKeys,
    ],
    SFTP: [
      "host",
      "port",
      "username",
      "authMode",
      "inboundDirectory",
      "outboundDirectory",
      "archiveDirectory",
      "fileNamingPattern",
      "knownHostKey",
      ...x12ProfileKeys,
    ],
    VAN: [
      "providerName",
      "mailboxId",
      "accountId",
      "senderMailboxId",
      "receiverMailboxId",
      "isaRoutingId",
      "gsRoutingId",
      "endpoint",
      "contactEmail",
      ...x12ProfileKeys,
    ],
  };
  return Object.fromEntries(
    keysByMethod[method]
      .map((key) => [key, config[key]] as const)
      .filter(([, value]) => String(value ?? "").trim() !== ""),
  );
}

const x12ProfileKeys: Array<keyof CommunicationProfileFormValues["config"]> = [
  "isaSenderQualifier",
  "isaSenderId",
  "isaReceiverQualifier",
  "isaReceiverId",
  "gsSenderId",
  "gsReceiverId",
  "x12Version",
  "environment",
  "acknowledgmentPreference",
];

function compactSecrets(
  secrets: CommunicationProfileFormValues["secrets"],
): Record<string, string> {
  return Object.fromEntries(Object.entries(secrets).filter(([, value]) => value.trim() !== ""));
}

function stringConfig(config: Record<string, unknown>, key: string, fallback = "") {
  const value = config[key];
  if (value === undefined || value === null) return fallback;
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return value.toString();
  return fallback;
}

function emptyToUndefined(value: string) {
  const trimmed = value.trim();
  return trimmed ? trimmed : undefined;
}

function PartnerPanel({ open, onOpenChange, mode, row }: DataTablePanelProps<EDIPartner>) {
  if (mode === "create") {
    return <CreatePairPanel open={open} onOpenChange={onOpenChange} />;
  }

  return <PartnerEditPanel open={open} onOpenChange={onOpenChange} partner={row} />;
}

function CreatePairPanel({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  const currentOrganizationId = useAuthStore((state) => state.user?.currentOrganizationId) ?? "";
  const form = useForm<CreateInternalPartnerPairFormValues>({
    resolver: zodResolver(createInternalPartnerPairSchema),
    defaultValues: getCreatePairDefaults(),
    mode: "onChange",
  });
  const { control, handleSubmit, reset, setValue, watch } = form;
  const values = watch();
  const { data: currentOrganization } = useQuery({
    queryKey: ["organization", "edi-current", currentOrganizationId],
    queryFn: () => apiService.organizationService.getByID(currentOrganizationId),
    enabled: open && Boolean(currentOrganizationId),
  });
  const mutation = useApiMutation({
    mutationFn: (values: CreateInternalPartnerPairFormValues) =>
      apiService.ediService.createConnection(toConnectionRequest(values)),
    onSuccess: async () => {
      toast.success("EDI connection requested");
      reset(getCreatePairDefaults());
      onOpenChange(false);
      await queryClient.invalidateQueries({ queryKey: ["edi-partner-list"] });
      await invalidateEDIConnections(queryClient);
    },
    onError: () => toast.error("Failed to request EDI connection"),
  });
  const canSubmit =
    values.targetOrganizationId &&
    values.sourceCode &&
    values.sourceName &&
    values.targetCode &&
    values.targetName;

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) {
      reset(getCreatePairDefaults());
    }
    onOpenChange(nextOpen);
  };
  const fillCurrentOrganizationPartner = useCallback(() => {
    if (!currentOrganization) return;

    setValue("targetCode", currentOrganization.scacCode, { shouldDirty: true });
    setValue("targetName", currentOrganization.name, { shouldDirty: true });
  }, [currentOrganization, setValue]);
  const handleTargetOrganizationChange = useCallback(
    (organization: OrganizationSelectOption | null) => {
      if (!organization) {
        setValue("sourceCode", "", { shouldDirty: true });
        setValue("sourceName", "", { shouldDirty: true });
        return;
      }

      setValue("sourceCode", organization.scacCode ?? "", { shouldDirty: true });
      setValue("sourceName", organization.name, { shouldDirty: true });
      fillCurrentOrganizationPartner();
    },
    [fillCurrentOrganizationPartner, setValue],
  );

  useEffect(() => {
    if (!open || !currentOrganization) return;
    if (values.targetCode || values.targetName) return;

    fillCurrentOrganizationPartner();
  }, [
    currentOrganization,
    fillCurrentOrganizationPartner,
    open,
    values.targetCode,
    values.targetName,
  ]);

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={handleOpenChange}
      title="Request EDI Connection"
      description="Configure the partner records that will be created after acceptance."
      size="xl"
      footer={
        <>
          <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
            Cancel
          </Button>
          <Button
            type="submit"
            form="edi-create-pair-form"
            disabled={!canSubmit}
            isLoading={mutation.isPending}
          >
            Request Connection
          </Button>
        </>
      }
    >
      <Form
        id="edi-create-pair-form"
        className="flex flex-col gap-4"
        onSubmit={(event) => {
          event.stopPropagation();
          void handleSubmit((submittedValues) => mutation.mutate(submittedValues))(event);
        }}
      >
        <FormGroup cols={2} className="gap-x-5 gap-y-3">
          <FormControl cols="full">
            <OrganizationAutocompleteField
              control={control}
              name="targetOrganizationId"
              label="Target Organization"
              placeholder="Select organization"
              description="Only organizations in the current business unit are available."
              rules={{ required: true }}
              extraSearchParams={{
                scope: "business-unit",
                excludeCurrent: "true",
              }}
              onOptionChange={handleTargetOrganizationChange}
            />
          </FormControl>
          <PartnerSideFields title="Current Organization View" prefix="source" control={control} />
          <PartnerSideFields title="Target Organization View" prefix="target" control={control} />
        </FormGroup>
      </Form>
    </DataTablePanelContainer>
  );
}

function toConnectionRequest(
  values: CreateInternalPartnerPairFormValues,
): CreateEDIConnectionRequest {
  return {
    targetOrganizationId: values.targetOrganizationId,
    method: "Internal",
    capabilities: {
      loadTenderOutbound: true,
      loadTenderInbound: true,
      shipmentStatus: false,
      invoice: false,
    },
    sourcePartnerConfig: {
      code: values.sourceCode,
      name: values.sourceName,
      description: values.sourceDescription,
      contactName: values.sourceContactName,
      contactEmail: values.sourceContactEmail,
      contactPhone: values.sourceContactPhone,
      enabledForInbound: values.sourceEnabledForInbound,
      enabledForOutbound: values.sourceEnabledForOutbound,
      settings: values.sourceSettings,
    },
    targetPartnerConfig: {
      code: values.targetCode,
      name: values.targetName,
      description: values.targetDescription,
      contactName: values.targetContactName,
      contactEmail: values.targetContactEmail,
      contactPhone: values.targetContactPhone,
      enabledForInbound: values.targetEnabledForInbound,
      enabledForOutbound: values.targetEnabledForOutbound,
      settings: values.targetSettings,
    },
  };
}

function getCreatePairDefaults(): CreateInternalPartnerPairFormValues {
  return {
    targetOrganizationId: "",
    sourceCode: "",
    sourceName: "",
    sourceDescription: "",
    sourceContactName: "",
    sourceContactEmail: "",
    sourceContactPhone: "",
    sourceEnabledForInbound: true,
    sourceEnabledForOutbound: true,
    sourceSettings: {},
    targetCode: "",
    targetName: "",
    targetDescription: "",
    targetContactName: "",
    targetContactEmail: "",
    targetContactPhone: "",
    targetEnabledForInbound: true,
    targetEnabledForOutbound: true,
    targetSettings: {},
  };
}

function PartnerSideFields({
  title,
  prefix,
  control,
}: {
  title: string;
  prefix: "source" | "target";
  control: Control<CreateInternalPartnerPairFormValues>;
}) {
  const codeName = `${prefix}Code` as const;
  const partnerName = `${prefix}Name` as const;
  const contactName = `${prefix}ContactName` as const;
  const contactEmail = `${prefix}ContactEmail` as const;
  const contactPhone = `${prefix}ContactPhone` as const;
  const inboundName = `${prefix}EnabledForInbound` as const;
  const outboundName = `${prefix}EnabledForOutbound` as const;

  return (
    <FormSection title={title} className="rounded-md border bg-muted/20 p-3">
      <FormGroup cols={2}>
        <FormControl>
          <InputField
            control={control}
            name={codeName}
            label="Partner Code"
            placeholder="Partner code"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name={partnerName}
            label="Partner Name"
            placeholder="Partner name"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name={contactName}
            label="Contact Name"
            placeholder="Contact name"
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name={contactEmail}
            label="Contact Email"
            placeholder="ops@example.com"
          />
        </FormControl>
        <FormControl cols="full">
          <InputField
            control={control}
            name={contactPhone}
            label="Contact Phone"
            placeholder="Contact phone"
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name={inboundName}
            label="Inbound Enabled"
            description="Allow this partner record to receive load tenders."
            outlined
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name={outboundName}
            label="Outbound Enabled"
            description="Allow this partner record to send load tenders."
            outlined
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}

function PartnerEditPanel({
  partner,
  open,
  onOpenChange,
}: {
  partner: EDIPartner | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  const canUpdate = usePermissionStore((state) =>
    state.hasPermission(Resource.EDI, Operation.Update),
  );
  const [draft, setDraft] = useState<EDIPartner | null>(partner);

  useEffect(() => {
    if (open) {
      setDraft(partner);
    }
  }, [open, partner]);

  const mutation = useApiMutation({
    mutationFn: (values: EDIPartner) => apiService.ediService.updatePartner(values),
    onSuccess: async () => {
      toast.success("EDI partner updated");
      await queryClient.invalidateQueries({ queryKey: ["edi-partner-list"] });
      await queryClient.invalidateQueries({ queryKey: queries.edi.partners._def });
    },
    onError: () => toast.error("Failed to update EDI partner"),
  });
  const handleClose = () => {
    onOpenChange(false);
    setDraft(null);
  };

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={partner?.name ?? "EDI Partner"}
      description="Partner settings and saved mapping profile."
      size="xl"
      footer={
        <>
          <Button type="button" variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          {draft && canUpdate && (
            <Button isLoading={mutation.isPending} onClick={() => mutation.mutate(draft)}>
              Save Partner
            </Button>
          )}
        </>
      }
    >
      {draft && (
        <Tabs defaultValue="details" className="min-h-0">
          <TabsList>
            <TabsTrigger value="details">Details</TabsTrigger>
            <TabsTrigger value="mappings">Mappings</TabsTrigger>
          </TabsList>
          <TabsContent value="details" className="pt-3">
            <div className="grid gap-3 md:grid-cols-2">
              <Input
                value={draft.code}
                disabled={!canUpdate}
                placeholder="Partner code"
                onChange={(event) => setDraft({ ...draft, code: event.target.value })}
              />
              <Input
                value={draft.name}
                disabled={!canUpdate}
                placeholder="Partner name"
                onChange={(event) => setDraft({ ...draft, name: event.target.value })}
              />
              <Input
                value={draft.contactName ?? ""}
                disabled={!canUpdate}
                placeholder="Contact name"
                onChange={(event) => setDraft({ ...draft, contactName: event.target.value })}
              />
              <Input
                value={draft.contactEmail ?? ""}
                disabled={!canUpdate}
                placeholder="Contact email"
                onChange={(event) => setDraft({ ...draft, contactEmail: event.target.value })}
              />
              <Input
                value={draft.contactPhone ?? ""}
                disabled={!canUpdate}
                placeholder="Contact phone"
                onChange={(event) => setDraft({ ...draft, contactPhone: event.target.value })}
              />
              <div className="flex items-center justify-between gap-2 rounded-md border p-2 text-sm">
                Inbound enabled
                <Switch
                  checked={draft.enabledForInbound}
                  disabled={!canUpdate}
                  onCheckedChange={(checked) => setDraft({ ...draft, enabledForInbound: checked })}
                />
              </div>
              <div className="flex items-center justify-between gap-2 rounded-md border p-2 text-sm">
                Outbound enabled
                <Switch
                  checked={draft.enabledForOutbound}
                  disabled={!canUpdate}
                  onCheckedChange={(checked) => setDraft({ ...draft, enabledForOutbound: checked })}
                />
              </div>
            </div>
          </TabsContent>
          <TabsContent value="mappings" className="pt-3">
            <MappingProfilePanel partnerId={draft.id} canUpdate={canUpdate} />
          </TabsContent>
        </Tabs>
      )}
    </DataTablePanelContainer>
  );
}

function MappingProfilePanel({ partnerId, canUpdate }: { partnerId: string; canUpdate: boolean }) {
  const queryClient = useQueryClient();
  const { data } = useQuery(queries.edi.mappingProfile(partnerId));
  const [draft, setDraft] = useState<EDIMappingProfileItem>({
    entityType: "Customer",
    sourceId: "",
    sourceLabel: "",
    targetId: "",
    targetLabel: "",
  });
  const saveMutation = useApiMutation({
    mutationFn: (item: EDIMappingProfileItem) =>
      data?.id
        ? apiService.ediService.saveMappingProfileItems(data.id, [item])
        : apiService.ediService.saveMappingProfile(partnerId, [item]),
    onSuccess: async () => {
      toast.success("Mapping saved");
      setDraft((current) => ({
        ...current,
        sourceId: "",
        sourceLabel: "",
        targetId: "",
        targetLabel: "",
      }));
      await queryClient.invalidateQueries({
        queryKey: queries.edi.mappingProfile(partnerId).queryKey,
      });
    },
    onError: () => toast.error("Failed to save mapping"),
  });
  const deleteMutation = useApiMutation({
    mutationFn: (itemId: string) =>
      data?.id
        ? apiService.ediService.deleteMappingProfileItem(data.id, itemId)
        : apiService.ediService.deleteMappingItem(partnerId, itemId),
    onSuccess: async () => {
      toast.success("Mapping deleted");
      await queryClient.invalidateQueries({
        queryKey: queries.edi.mappingProfile(partnerId).queryKey,
      });
    },
    onError: () => toast.error("Failed to delete mapping"),
  });

  return (
    <Tabs defaultValue="Customer" className="gap-3">
      <TabsList className="flex-wrap">
        {mappingEntityTypes.map((entityType) => (
          <TabsTrigger key={entityType} value={entityType}>
            {entityType}
          </TabsTrigger>
        ))}
      </TabsList>
      {mappingEntityTypes.map((entityType) => {
        const entries = (data?.entries ?? []).filter((entry) => entry.entityType === entityType);
        return (
          <TabsContent key={entityType} value={entityType} className="flex flex-col gap-3">
            {canUpdate && (
              <div className="grid gap-2 md:grid-cols-5">
                <Input
                  placeholder="Source value key"
                  value={draft.entityType === entityType ? draft.sourceId : ""}
                  onChange={(event) =>
                    setDraft({ ...draft, entityType, sourceId: event.target.value })
                  }
                />
                <Input
                  placeholder="Source label"
                  value={draft.entityType === entityType ? (draft.sourceLabel ?? "") : ""}
                  onChange={(event) =>
                    setDraft({ ...draft, entityType, sourceLabel: event.target.value })
                  }
                />
                <TargetLookup
                  entityType={entityType}
                  value={draft.entityType === entityType ? draft.targetId : ""}
                  onChange={(target) =>
                    setDraft({
                      ...draft,
                      entityType,
                      targetId: target.targetId,
                      targetLabel: target.targetLabel,
                    })
                  }
                />
                <Input
                  placeholder="Target label"
                  value={draft.entityType === entityType ? (draft.targetLabel ?? "") : ""}
                  onChange={(event) =>
                    setDraft({ ...draft, entityType, targetLabel: event.target.value })
                  }
                />
                <Button
                  disabled={!draft.sourceId || !draft.targetId || draft.entityType !== entityType}
                  onClick={() => saveMutation.mutate(draft)}
                >
                  <CheckIcon data-icon="inline-start" />
                  Save
                </Button>
              </div>
            )}
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Source</TableHead>
                    <TableHead>Target</TableHead>
                    <TableHead />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {entries.map((entry) => (
                    <TableRow key={entry.id ?? `${entry.entityType}-${entry.sourceId}`}>
                      <TableCell>{entry.sourceLabel || "Unlabeled source value"}</TableCell>
                      <TableCell>{entry.targetLabel || "Mapped local record"}</TableCell>
                      <TableCell className="text-right">
                        {canUpdate && entry.id && (
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            onClick={() => deleteMutation.mutate(entry.id!)}
                          >
                            <Trash2Icon />
                          </Button>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                  {entries.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={3} className="h-16 text-center text-muted-foreground">
                        No mappings saved for {entityType}.
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </div>
          </TabsContent>
        );
      })}
    </Tabs>
  );
}

function MappingProfilesWorkspace() {
  const canUpdate = usePermissionStore((state) =>
    state.hasPermission(Resource.EDI, Operation.Update),
  );
  const [selectedPartner, setSelectedPartner] = useState<EDIPartner | null>(null);
  const { control } = useForm<{ partnerId: string }>({
    defaultValues: { partnerId: "" },
  });
  const selectedPartnerId = useWatch({ control, name: "partnerId" });

  return (
    <div className="grid min-h-0 gap-4 lg:grid-cols-[18rem_1fr]">
      <div className="rounded-md border bg-background">
        <div className="border-b px-3 py-2">
          <div className="text-sm font-medium">Partner</div>
          <div className="text-xs text-muted-foreground">
            Choose which partner source values should map into local records.
          </div>
        </div>
        <div className="p-3">
          <EDIPartnerAutocompleteField
            control={control}
            name="partnerId"
            placeholder="Select partner"
            clearable
            onOptionChange={setSelectedPartner}
          />
          {selectedPartner && (
            <div className="mt-3 rounded-md border bg-muted/20 p-3 text-sm">
              <div className="font-medium">{selectedPartner.name}</div>
              <div className="text-xs text-muted-foreground">{selectedPartner.code}</div>
            </div>
          )}
        </div>
      </div>
      <div className="min-w-0 rounded-md border bg-background p-3">
        {selectedPartnerId ? (
          <MappingProfilePanel partnerId={selectedPartnerId} canUpdate={canUpdate} />
        ) : (
          <EmptyReviewState message="Select a partner to manage mapping records." />
        )}
      </div>
    </div>
  );
}

function TransfersWorkspace({ direction }: { direction: "inbound" | "outbound" }) {
  const columns = useMemo(() => getTransferColumns(direction), [direction]);
  const TransferPanel = useCallback(
    (props: DataTablePanelProps<EDITransfer>) => (
      <EDITransferReviewPanel {...props} direction={direction} />
    ),
    [direction],
  );

  return (
    <DataTable<EDITransfer>
      name="EDI Transfer"
      link={direction === "inbound" ? "/edi/transfers/inbound/" : "/edi/transfers/outbound/"}
      detailLink="/edi/transfers/"
      queryKey={
        direction === "inbound" ? "edi-inbound-transfer-list" : "edi-outbound-transfer-list"
      }
      exportModelName={`edi-${direction}-transfer`}
      resource={Resource.EDI}
      columns={columns}
      TablePanel={TransferPanel}
      preferDetailRowForEdit
      enableCreateAction={false}
      enableReadOnlyPanel
    />
  );
}

function EmptyReviewState({ message }: { message: string }) {
  return (
    <div className="rounded-md border border-dashed bg-muted/20 px-3 py-6 text-center text-sm text-muted-foreground">
      {message}
    </div>
  );
}

async function invalidateEDIConnections(queryClient: ReturnType<typeof useQueryClient>) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: queries.edi.connections._def }),
    queryClient.invalidateQueries({ queryKey: queries.edi.partners._def }),
    queryClient.invalidateQueries({ queryKey: queries.edi.communicationProfiles._def }),
    queryClient.invalidateQueries({ queryKey: ["edi-partner-list"] }),
  ]);
}
