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
                placeholder="TRENOVA"
                rules={{ required: true }}
                description="Our AS2 identifier that the partner uses to address messages to us."
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="config.partnerAS2Id"
                label="Partner AS2 ID"
                placeholder="PARTNERCO"
                rules={{ required: true }}
                description="The partner's AS2 identifier that we address outbound messages to."
              />
            </FormControl>
            <FormControl cols="full">
              <InputField
                control={control}
                name="config.endpointUrl"
                label="Endpoint URL"
                placeholder="https://edi.partner.com/as2"
                rules={{ required: true }}
                description="The partner's HTTPS URL where we POST outbound AS2 messages."
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
                placeholder="https://edi.trenova.com/as2/mdn"
                description="Required when MDN mode is asynchronous."
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                name="config.signingAlgorithm"
                label="Signing Algorithm"
                options={as2SigningAlgorithmOptions}
                description="The hashing algorithm used to sign outbound messages and MDNs."
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                name="config.encryptionAlgorithm"
                label="Encryption Algorithm"
                options={as2EncryptionAlgorithmOptions}
                description="The cipher used to encrypt outbound message payloads."
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                name="config.compressionAlgorithm"
                label="Compression"
                options={as2CompressionOptions}
                description="Compresses outbound payloads before encryption to reduce transfer size."
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="config.basicAuthUsername"
                label="Basic Auth Username"
                placeholder="trenova"
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
              placeholder="OpenText / SPS Commerce"
              rules={{ required: true }}
              description="The name of the VAN provider hosting this mailbox."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="config.mailboxId"
              label="Mailbox ID"
              placeholder="MB123456"
              rules={{ required: true }}
              description="The mailbox identifier assigned by the VAN provider for routing documents."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="config.accountId"
              label="Account ID"
              placeholder="ACCT-0001"
              description="The account identifier with the VAN provider, if separate from the mailbox."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="config.contactEmail"
              label="Contact Email"
              placeholder="edi@trenova.com"
              description="The email address the VAN provider uses for service notifications."
            />
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
    <FormSection title="Delivery Retry">
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
      <FormSection title={title}>
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="config.host"
              label="Host"
              placeholder="sftp.partner.com"
              description="The host name or IP address of the SFTP server."
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="config.port"
              label="Port"
              placeholder="22"
              description="The port number of the SFTP server."
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="config.username"
              label="Username"
              placeholder="trenova"
              description="The username for the SFTP server."
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="config.authMode"
              label="Authentication"
              options={sftpAuthModeOptions}
              description="The authentication mode for the SFTP server."
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="config.knownHostKey"
              label="Known Host Key"
              placeholder="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5..."
              description="The known host key for the SFTP server."
              rules={{ required: true }}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection title="Directories">
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="config.inboundDirectory"
              label="Inbound Directory"
              placeholder="/inbound"
              description="The directory where inbound files are stored."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="config.outboundDirectory"
              label="Outbound Directory"
              placeholder="/outbound"
              description="The directory where outbound files are written for pickup."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="config.archiveDirectory"
              label="Archive Directory"
              placeholder="/archive"
              description="The directory where processed files are moved for retention."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="config.fileNamingPattern"
              label="File Naming Pattern"
              placeholder="{partner}-{timestamp}.edi"
              description="The template used to name outbound files, with token substitution."
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </>
  );
}

export function X12EnvelopeFields({ control }: ProfileFieldsProps) {
  return (
    <FormSection title="X12 Envelope">
      <FormGroup cols={2}>
        <FormControl>
          <InputField
            control={control}
            name="config.isaSenderQualifier"
            label="ISA Sender Qualifier"
            placeholder="ZZ"
            rules={{ required: true }}
            description="The qualifier code that identifies the type of our ISA sender ID (e.g. 01, ZZ)."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.isaSenderId"
            label="ISA Sender ID"
            placeholder="TRENOVA"
            rules={{ required: true }}
            description="Our sender identifier placed in the ISA interchange header."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.isaReceiverQualifier"
            label="ISA Receiver Qualifier"
            placeholder="ZZ"
            rules={{ required: true }}
            description="The qualifier code that identifies the type of the partner's ISA receiver ID."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.isaReceiverId"
            label="ISA Receiver ID"
            placeholder="PARTNERCO"
            rules={{ required: true }}
            description="The partner's receiver identifier placed in the ISA interchange header."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.gsSenderId"
            label="GS Sender ID"
            placeholder="TRENOVA"
            rules={{ required: true }}
            description="Our application sender code placed in the GS functional group header."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.gsReceiverId"
            label="GS Receiver ID"
            placeholder="PARTNERCO"
            rules={{ required: true }}
            description="The partner's application receiver code placed in the GS functional group header."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.x12Version"
            label="X12 Version"
            placeholder="004010"
            rules={{ required: true }}
            description="The X12 release version the partner expects (e.g. 004010, 005010)."
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="config.environment"
            label="Environment"
            options={environmentOptions}
            rules={{ required: true }}
            description="Whether this envelope targets the partner's test or production system."
          />
        </FormControl>
        <FormControl cols="full">
          <SelectField
            control={control}
            name="config.acknowledgmentPreference"
            label="Acknowledgment Preference"
            options={acknowledgmentOptions}
            description="Which functional acknowledgments (997/999) to request from the partner."
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

  const secretState = profile ? profile.secretState : null;

  return (
    <div className="space-y-3">
      {secretState && secretState.length > 0 && (
        <div className="rounded-md border bg-muted/20 p-3">
          <div className="mb-2 text-sm font-medium">Saved Secrets</div>
          <div className="flex flex-wrap gap-1.5">
            {secretState.map((secret) => (
              <Badge key={secret.key} variant="secondary">
                {secret.key} saved
              </Badge>
            ))}
          </div>
        </div>
      )}
      <FormSection title="Secret Values">
        <FormGroup cols={1}>
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
              <FormControl cols="full">
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
            <FormControl cols="full">
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
