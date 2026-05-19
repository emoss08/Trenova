import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormGroup, FormSection } from "@/components/ui/form";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { statusChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { DataTablePanelProps } from "@/types/data-table";
import type { EDICommunicationProfile, UpsertEDICommunicationProfileRequest } from "@/types/edi";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { KeyRoundIcon, RadioTowerIcon, ServerIcon, ShieldCheckIcon } from "lucide-react";
import { useEffect } from "react";
import { useForm, useWatch, type Control } from "react-hook-form";
import { toast } from "sonner";
import {
  acknowledgmentOptions,
  communicationProfileFormSchema,
  communicationProfileMethodOptions,
  environmentOptions,
  mdnModeOptions,
  sftpAuthModeOptions,
  type CommunicationProfileFormValues,
  type CommunicationProfileMethod,
} from "../edi-schemas";

export function CommunicationProfilePanel({
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
            <TabsList variant="underline" className="grid w-full grid-cols-4 border-b border-border">
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
                      options={statusChoices}
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
