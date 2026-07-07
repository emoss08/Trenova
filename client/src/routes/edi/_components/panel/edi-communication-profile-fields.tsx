import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { SensitiveTextareaField } from "@/components/fields/sensitive-textarea-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Badge } from "@/components/ui/badge";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import type { EDICommunicationProfile } from "@/types/edi";
import type { Control } from "react-hook-form";
import {
  acknowledgmentOptions,
  as2CompressionOptions,
  as2EncryptionAlgorithmOptions,
  as2InboundRequirementOptions,
  as2SigningAlgorithmOptions,
  environmentOptions,
  mdnModeOptions,
  sftpAuthModeOptions,
  type CommunicationProfileFormValues,
  type CommunicationProfileMethod,
} from "../edi-schemas";
import { EDICertificateField } from "./edi-certificate-field";
import { EDIEmptyState } from "./edi-panel-primitives";

type ProfileFieldsProps = {
  control: Control<CommunicationProfileFormValues>;
};

export function TransportProfileFields({
  control,
  method,
  authMode,
}: ProfileFieldsProps & {
  method: CommunicationProfileMethod;
  authMode: string;
}) {
  if (method === "Internal") {
    return (
      <EDIEmptyState message="Internal communication is enabled through accepted organization connections." />
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
              <SelectField
                control={control}
                name="config.mdnMode"
                label="MDN Mode"
                options={mdnModeOptions}
                rules={{ required: true }}
                description="Synchronous MDNs return in the HTTP response; asynchronous MDNs post back to the return URL."
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="config.mdnUrl"
                label="Async MDN Return URL"
                description="Required when MDN mode is asynchronous."
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                name="config.signingAlgorithm"
                label="Signing Algorithm"
                options={as2SigningAlgorithmOptions}
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                name="config.encryptionAlgorithm"
                label="Encryption Algorithm"
                options={as2EncryptionAlgorithmOptions}
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                name="config.compressionAlgorithm"
                label="Compression"
                options={as2CompressionOptions}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="config.basicAuthUsername"
                label="Basic Auth Username"
                description="Optional HTTP basic auth credential the partner endpoint expects."
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                name="config.requireSignedInbound"
                label="Require Signed Inbound"
                options={as2InboundRequirementOptions}
                description="Reject inbound documents that are not signed by the partner. Automatic requires a signature when a partner signing certificate is configured."
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                name="config.requireEncryptedInbound"
                label="Require Encrypted Inbound"
                options={as2InboundRequirementOptions}
                description="Reject inbound documents that are not encrypted to us. Automatic requires encryption when a local certificate and private key are configured."
              />
            </FormControl>
          </FormGroup>
        </FormSection>
        <FormSection title="Certificates" className="rounded-md border bg-muted/20 p-3">
          <FormGroup cols={1}>
            <FormControl cols="full">
              <EDICertificateField
                control={control}
                name="config.localCertificate"
                label="Local Certificate (PEM)"
                description="Our public certificate. Partners use it to encrypt to us and verify our signatures; pair it with the private key secret."
              />
            </FormControl>
            <FormControl cols="full">
              <EDICertificateField
                control={control}
                name="config.partnerSigningCertificate"
                label="Partner Signing Certificate (PEM)"
                description="Used to verify inbound signatures and signed MDNs from this partner."
              />
            </FormControl>
            <FormControl cols="full">
              <EDICertificateField
                control={control}
                name="config.partnerEncryptionCertificate"
                label="Partner Encryption Certificate (PEM)"
                description="Used to encrypt outbound documents. Leave blank to reuse the signing certificate."
              />
            </FormControl>
          </FormGroup>
        </FormSection>
        <DeliveryRetrySection control={control} />
      </>
    );
  }

  if (method === "SFTP") {
    return (
      <>
        <SftpEndpointSections control={control} title="SFTP Endpoint" />
        <DeliveryRetrySection control={control} />
        <EDIEmptyState
          message={`Save a ${authMode === "password" ? "password" : "private key"} in the Secrets tab before activating this profile.`}
        />
      </>
    );
  }

  return (
    <>
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
            <InputField control={control} name="config.contactEmail" label="Contact Email" />
          </FormControl>
        </FormGroup>
      </FormSection>
      <SftpEndpointSections control={control} title="VAN Gateway Endpoint" />
      <DeliveryRetrySection control={control} />
      <EDIEmptyState
        message={`Save a ${authMode === "password" ? "password" : "private key"} in the Secrets tab before activating this profile.`}
      />
    </>
  );
}

function DeliveryRetrySection({ control }: ProfileFieldsProps) {
  return (
    <FormSection title="Delivery Retry" className="rounded-md border bg-muted/20 p-3">
      <FormGroup cols={3}>
        <FormControl>
          <InputField
            control={control}
            name="config.retryMaxAttempts"
            label="Max Attempts"
            type="number"
            placeholder="6"
            description="Delivery attempts before the message is dead-lettered. Defaults to 6."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.retryInitialIntervalSeconds"
            label="Initial Backoff (seconds)"
            type="number"
            placeholder="30"
            description="Wait before the first retry; doubles each attempt. Defaults to 30 seconds."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.retryMaxIntervalSeconds"
            label="Max Backoff (seconds)"
            type="number"
            placeholder="900"
            description="Upper bound on the retry backoff. Defaults to 900 seconds (15 minutes)."
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}

function SftpEndpointSections({ control, title }: ProfileFieldsProps & { title: string }) {
  return (
    <>
      <FormSection title={title} className="rounded-md border bg-muted/20 p-3">
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
    </>
  );
}

export function X12EnvelopeFields({ control }: ProfileFieldsProps) {
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

export function SecretProfileFields({
  control,
  method,
  profile,
  authMode,
}: ProfileFieldsProps & {
  method: CommunicationProfileMethod;
  profile: EDICommunicationProfile | null;
  authMode: string;
}) {
  if (method === "Internal") {
    return <EDIEmptyState message="Internal profiles do not store external credentials." />;
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
            <>
              <FormControl cols="full">
                <SensitiveTextareaField
                  control={control}
                  name="secrets.privateKey"
                  label="AS2 Private Key (PEM)"
                  description="Pairs with the local certificate for signing and decryption. Leave blank to keep the saved value."
                />
              </FormControl>
              <FormControl>
                <SensitiveField
                  control={control}
                  name="secrets.basicAuthPassword"
                  label="Basic Auth Password"
                  description="Leave blank to keep the saved value."
                />
              </FormControl>
            </>
          )}
          {(method === "SFTP" || method === "VAN") && authMode === "password" && (
            <FormControl>
              <SensitiveField
                control={control}
                name="secrets.password"
                label={method === "VAN" ? "VAN Gateway Password" : "SFTP Password"}
                description="Leave blank to keep the saved value."
              />
            </FormControl>
          )}
          {(method === "SFTP" || method === "VAN") && authMode !== "password" && (
            <FormControl cols="full">
              <TextareaField
                control={control}
                name="secrets.privateKey"
                label="Private Key"
                description="Leave blank to keep the saved value."
              />
            </FormControl>
          )}
        </FormGroup>
      </FormSection>
    </div>
  );
}
