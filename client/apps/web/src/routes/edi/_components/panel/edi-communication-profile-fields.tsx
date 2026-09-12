import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { SensitiveTextareaField } from "@/components/fields/sensitive-textarea-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Badge } from "@trenova/shared/components/ui/badge";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import type { EDICommunicationProfileRow } from "@/lib/graphql/edi-table";
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
  const t = useT();

  if (method === "Internal") {
    return (
      <EDIEmptyState
        message={t("Internal communication is enabled through accepted organization connections.")}
      />
    );
  }

  if (method === "AS2") {
    return (
      <>
        <FormSection title={t("AS2 Identifiers")} className="bg-muted/20 rounded-md border p-3">
          <FormGroup cols={2}>
            <FormControl>
              <InputField
                control={control}
                name="config.localAS2Id"
                label={t("Local AS2 ID")}
                placeholder="TRENOVA"
                rules={{ required: true }}
                description={t(
                  "Our AS2 identifier that the partner uses to address messages to us.",
                )}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="config.partnerAS2Id"
                label={t("Partner AS2 ID")}
                placeholder="PARTNERCO"
                rules={{ required: true }}
                description={t(
                  "The partner's AS2 identifier that we address outbound messages to.",
                )}
              />
            </FormControl>
            <FormControl cols="full">
              <InputField
                control={control}
                name="config.endpointUrl"
                label={t("Endpoint URL")}
                placeholder="https://edi.partner.com/as2"
                rules={{ required: true }}
                description={t("The partner's HTTPS URL where we POST outbound AS2 messages.")}
              />
            </FormControl>
          </FormGroup>
        </FormSection>
        <FormSection title={t("Security and MDN")} className="bg-muted/20 rounded-md border p-3">
          <FormGroup cols={2}>
            <FormControl>
              <SelectField
                control={control}
                name="config.mdnMode"
                label={t("MDN Mode")}
                options={mdnModeOptions}
                rules={{ required: true }}
                description={t(
                  "Synchronous MDNs return in the HTTP response; asynchronous MDNs post back to the return URL.",
                )}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="config.mdnUrl"
                label={t("Async MDN Return URL")}
                placeholder="https://edi.trenova.com/as2/mdn"
                description={t("Required when MDN mode is asynchronous.")}
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                name="config.signingAlgorithm"
                label={t("Signing Algorithm")}
                options={as2SigningAlgorithmOptions}
                description={t("The hashing algorithm used to sign outbound messages and MDNs.")}
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                name="config.encryptionAlgorithm"
                label={t("Encryption Algorithm")}
                options={as2EncryptionAlgorithmOptions}
                description={t("The cipher used to encrypt outbound message payloads.")}
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                name="config.compressionAlgorithm"
                label={t("Compression")}
                options={as2CompressionOptions}
                description={t(
                  "Compresses outbound payloads before encryption to reduce transfer size.",
                )}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="config.basicAuthUsername"
                label={t("Basic Auth Username")}
                placeholder="trenova"
                description={t("Optional HTTP basic auth credential the partner endpoint expects.")}
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                name="config.requireSignedInbound"
                label={t("Require Signed Inbound")}
                options={as2InboundRequirementOptions}
                description={t(
                  "Reject inbound documents that are not signed by the partner. Automatic requires a signature when a partner signing certificate is configured.",
                )}
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                name="config.requireEncryptedInbound"
                label={t("Require Encrypted Inbound")}
                options={as2InboundRequirementOptions}
                description={t(
                  "Reject inbound documents that are not encrypted to us. Automatic requires encryption when a local certificate and private key are configured.",
                )}
              />
            </FormControl>
          </FormGroup>
        </FormSection>
        <FormSection title={t("Certificates")} className="bg-muted/20 rounded-md border p-3">
          <FormGroup cols={1}>
            <FormControl cols="full">
              <EDICertificateField
                control={control}
                name="config.localCertificate"
                label={t("Local Certificate (PEM)")}
                description={t(
                  "Our public certificate. Partners use it to encrypt to us and verify our signatures; pair it with the private key secret.",
                )}
              />
            </FormControl>
            <FormControl cols="full">
              <EDICertificateField
                control={control}
                name="config.partnerSigningCertificate"
                label={t("Partner Signing Certificate (PEM)")}
                description={t(
                  "Used to verify inbound signatures and signed MDNs from this partner.",
                )}
              />
            </FormControl>
            <FormControl cols="full">
              <EDICertificateField
                control={control}
                name="config.partnerEncryptionCertificate"
                label={t("Partner Encryption Certificate (PEM)")}
                description={t(
                  "Used to encrypt outbound documents. Leave blank to reuse the signing certificate.",
                )}
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
        <SftpEndpointSections control={control} title={t("SFTP Endpoint")} />
        <DeliveryRetrySection control={control} />
        <EDIEmptyState
          message={`Save a ${authMode === "password" ? "password" : "private key"} in the Secrets tab before activating this profile.`}
        />
      </>
    );
  }

  return (
    <>
      <FormSection title={t("VAN Mailbox")} className="bg-muted/20 rounded-md border p-3">
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="config.providerName"
              label={t("Provider Name")}
              placeholder={t("OpenText / SPS Commerce")}
              rules={{ required: true }}
              description={t("The name of the VAN provider hosting this mailbox.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="config.mailboxId"
              label={t("Mailbox ID")}
              placeholder="MB123456"
              rules={{ required: true }}
              description={t(
                "The mailbox identifier assigned by the VAN provider for routing documents.",
              )}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="config.accountId"
              label={t("Account ID")}
              placeholder={t("ACCT-0001")}
              description={t(
                "The account identifier with the VAN provider, if separate from the mailbox.",
              )}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="config.contactEmail"
              label={t("Contact Email")}
              placeholder={t("edi@trenova.com")}
              description={t("The email address the VAN provider uses for service notifications.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <SftpEndpointSections control={control} title={t("VAN Gateway Endpoint")} />
      <DeliveryRetrySection control={control} />
      <EDIEmptyState
        message={`Save a ${authMode === "password" ? "password" : "private key"} in the Secrets tab before activating this profile.`}
      />
    </>
  );
}

function DeliveryRetrySection({ control }: ProfileFieldsProps) {
  const t = useT();

  return (
    <FormSection title={t("Delivery Retry")}>
      <FormGroup cols={3}>
        <FormControl>
          <InputField
            control={control}
            name="config.retryMaxAttempts"
            label={t("Max Attempts")}
            type="number"
            placeholder="6"
            description={t("Delivery attempts before the message is dead-lettered. Defaults to 6.")}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.retryInitialIntervalSeconds"
            label={t("Initial Backoff (seconds)")}
            type="number"
            placeholder="30"
            description={t(
              "Wait before the first retry; doubles each attempt. Defaults to 30 seconds.",
            )}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.retryMaxIntervalSeconds"
            label={t("Max Backoff (seconds)")}
            type="number"
            placeholder="900"
            description={t(
              "Upper bound on the retry backoff. Defaults to 900 seconds (15 minutes).",
            )}
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}

function SftpEndpointSections({ control, title }: ProfileFieldsProps & { title: string }) {
  const t = useT();

  return (
    <>
      <FormSection title={title}>
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="config.host"
              label={t("Host")}
              placeholder={t("sftp.partner.com")}
              description={t("The host name or IP address of the SFTP server.")}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="config.port"
              label={t("Port")}
              placeholder="22"
              description={t("The port number of the SFTP server.")}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="config.username"
              label={t("Username")}
              placeholder="trenova"
              description={t("The username for the SFTP server.")}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="config.authMode"
              label={t("Authentication")}
              options={sftpAuthModeOptions}
              description={t("The authentication mode for the SFTP server.")}
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="config.knownHostKey"
              label={t("Known Host Key")}
              placeholder={t("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5...")}
              description={t("The known host key for the SFTP server.")}
              rules={{ required: true }}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <FormSection title={t("Directories")}>
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="config.inboundDirectory"
              label={t("Inbound Directory")}
              placeholder="/inbound"
              description={t("The directory where inbound files are stored.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="config.outboundDirectory"
              label={t("Outbound Directory")}
              placeholder="/outbound"
              description={t("The directory where outbound files are written for pickup.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="config.archiveDirectory"
              label={t("Archive Directory")}
              placeholder="/archive"
              description={t("The directory where processed files are moved for retention.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="config.fileNamingPattern"
              label={t("File Naming Pattern")}
              placeholder="{partner}-{timestamp}.edi"
              description={t("The template used to name outbound files, with token substitution.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </>
  );
}

export function X12EnvelopeFields({ control }: ProfileFieldsProps) {
  const t = useT();

  return (
    <FormSection title={t("X12 Envelope")}>
      <FormGroup cols={2}>
        <FormControl>
          <InputField
            control={control}
            name="config.isaSenderQualifier"
            label={t("ISA Sender Qualifier")}
            placeholder={t("ZZ")}
            rules={{ required: true }}
            description={t(
              "The qualifier code that identifies the type of our ISA sender ID (e.g. 01, ZZ).",
            )}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.isaSenderId"
            label={t("ISA Sender ID")}
            placeholder="TRENOVA"
            rules={{ required: true }}
            description={t("Our sender identifier placed in the ISA interchange header.")}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.isaReceiverQualifier"
            label={t("ISA Receiver Qualifier")}
            placeholder={t("ZZ")}
            rules={{ required: true }}
            description={t(
              "The qualifier code that identifies the type of the partner's ISA receiver ID.",
            )}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.isaReceiverId"
            label={t("ISA Receiver ID")}
            placeholder="PARTNERCO"
            rules={{ required: true }}
            description={t(
              "The partner's receiver identifier placed in the ISA interchange header.",
            )}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.gsSenderId"
            label={t("GS Sender ID")}
            placeholder="TRENOVA"
            rules={{ required: true }}
            description={t("Our application sender code placed in the GS functional group header.")}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.gsReceiverId"
            label={t("GS Receiver ID")}
            placeholder="PARTNERCO"
            rules={{ required: true }}
            description={t(
              "The partner's application receiver code placed in the GS functional group header.",
            )}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="config.x12Version"
            label={t("X12 Version")}
            placeholder="004010"
            rules={{ required: true }}
            description={t("The X12 release version the partner expects (e.g. 004010, 005010).")}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="config.environment"
            label={t("Environment")}
            options={environmentOptions}
            rules={{ required: true }}
            description={t(
              "Whether this envelope targets the partner's test or production system.",
            )}
          />
        </FormControl>
        <FormControl cols="full">
          <SelectField
            control={control}
            name="config.acknowledgmentPreference"
            label={t("Acknowledgment Preference")}
            options={acknowledgmentOptions}
            description={t(
              "Which functional acknowledgments (997/999) to request from the partner.",
            )}
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
  profile: EDICommunicationProfileRow | null;
  authMode: string;
}) {
  const t = useT();

  if (method === "Internal") {
    return <EDIEmptyState message={t("Internal profiles do not store external credentials.")} />;
  }

  const secretState = profile ? profile.secretState : null;

  return (
    <div className="space-y-3">
      {secretState && secretState.length > 0 && (
        <div className="bg-muted/20 rounded-md border p-3">
          <div className="mb-2 text-sm font-medium">{t("Saved Secrets")}</div>
          <div className="flex flex-wrap gap-1.5">
            {secretState.map((secret) => (
              <Badge key={secret.key} variant="secondary">
                {t("{0} saved", secret.key)}
              </Badge>
            ))}
          </div>
        </div>
      )}
      <FormSection title={t("Secret Values")}>
        <FormGroup cols={1}>
          {method === "AS2" && (
            <>
              <FormControl cols="full">
                <SensitiveTextareaField
                  control={control}
                  name="secrets.privateKey"
                  label={t("AS2 Private Key (PEM)")}
                  description={t(
                    "Pairs with the local certificate for signing and decryption. Leave blank to keep the saved value.",
                  )}
                />
              </FormControl>
              <FormControl cols="full">
                <SensitiveField
                  control={control}
                  name="secrets.basicAuthPassword"
                  label={t("Basic Auth Password")}
                  description={t("Leave blank to keep the saved value.")}
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
                description={t("Leave blank to keep the saved value.")}
              />
            </FormControl>
          )}
          {(method === "SFTP" || method === "VAN") && authMode !== "password" && (
            <FormControl cols="full">
              <TextareaField
                control={control}
                name="secrets.privateKey"
                label={t("Private Key")}
                description={t("Leave blank to keep the saved value.")}
              />
            </FormControl>
          )}
        </FormGroup>
      </FormSection>
    </div>
  );
}
