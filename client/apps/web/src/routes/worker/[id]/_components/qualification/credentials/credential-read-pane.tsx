import { WorkPane, WorkPaneBody, WorkPaneFooter, type PreloadIntent } from "@/components/work-pane";
import {
  fetchWorkerCredentials,
  WORKER_CREDENTIALS_KEY,
  type WorkerCredentialRow,
} from "@/lib/graphql/worker-credential";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { DocumentPanePreview } from "@/components/documents/document-pane-preview";
import { DocumentThumbnail } from "@/components/documents/document-thumbnail";
import { documentContentUrl } from "@/services/document";
import { ShieldTickIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatRange, formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { CredentialPermissions } from "./credential-table";
import {
  earlierVersions,
  expiryPhrase,
  expiryToneClass,
  healthLabel,
  healthTone,
} from "./credential-model";

/**
 * One credential read beside the table: its document, the facts on the card, the versions it
 * replaced, and its actions in the footer. Verify sits there only while nobody has checked it.
 */
export function CredentialReadPane({
  open,
  onOpenChange,
  credentialId,
  credential,
  workerId,
  loadedEarlier,
  permissions,
  verifying,
  onVerify,
  editIntent,
  onEdit,
  onRenew,
  onArchive,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The credential the address names, found or not. */
  credentialId: string | null;
  credential: WorkerCredentialRow | null;
  workerId: string;
  /** The versions the tab's capped list already holds, shown while the type's own read lands. */
  loadedEarlier: readonly WorkerCredentialRow[];
  permissions: CredentialPermissions;
  verifying: boolean;
  onVerify: (credential: WorkerCredentialRow) => Promise<void>;
  /** Reads the form's code as the pointer reaches Edit or Renew. */
  editIntent: PreloadIntent;
  onEdit: (credential: WorkerCredentialRow) => void;
  onRenew: (credential: WorkerCredentialRow) => void;
  onArchive: (credential: WorkerCredentialRow) => void;
}) {
  const t = useT();
  const active = credential?.status === "Active";

  return (
    <WorkPane
      open={open}
      onOpenChange={onOpenChange}
      title={credential?.credentialType?.name ?? t("Credential")}
      contentKey={credentialId ?? undefined}
    >
      <WorkPaneBody>
        {credential ? (
          <CredentialDetails
            credential={credential}
            workerId={workerId}
            loadedEarlier={loadedEarlier}
          />
        ) : (
          <p className="text-sm text-muted-foreground">
            {t("This credential is no longer on the file.")}
          </p>
        )}
      </WorkPaneBody>
      {credential && active ? (
        <WorkPaneFooter
          leading={
            permissions.canArchive ? (
              <Button
                type="button"
                size="sm"
                variant="outline"
                className="text-danger-foreground"
                onClick={() => onArchive(credential)}
              >
                {t("Archive")}
              </Button>
            ) : null
          }
        >
          {!credential.verifiedAt && permissions.canVerify ? (
            <Button
              type="button"
              size="sm"
              variant="outline"
              isLoading={verifying}
              onClick={() => void onVerify(credential)}
            >
              <ShieldTickIcon className="size-3.5" />
              {t("Verify")}
            </Button>
          ) : null}
          {permissions.canUpdate ? (
            <Button
              type="button"
              size="sm"
              variant="outline"
              {...editIntent}
              onClick={() => onEdit(credential)}
            >
              {t("Edit")}
            </Button>
          ) : null}
          {permissions.canCreate ? (
            <Button type="button" size="sm" {...editIntent} onClick={() => onRenew(credential)}>
              {t("Renew")}
            </Button>
          ) : null}
        </WorkPaneFooter>
      ) : null}
    </WorkPane>
  );
}

function CredentialDetails({
  credential,
  workerId,
  loadedEarlier,
}: {
  credential: WorkerCredentialRow;
  workerId: string;
  loadedEarlier: readonly WorkerCredentialRow[];
}) {
  const t = useT();

  return (
    <>
      <div className="flex items-center gap-2">
        <Badge variant={healthTone(credential.health)} dot>
          {healthLabel(t, credential.health)}
        </Badge>
        {credential.status === "Archived" ? (
          <span className="text-xs text-muted-foreground">
            {credential.archivedAt
              ? t("Archived {0}", formatUnixDateMedium(credential.archivedAt))
              : t("Archived")}
            {credential.archiveReason ? ` · ${credential.archiveReason}` : null}
          </span>
        ) : null}
      </div>

      <DocumentPanePreview
        document={credential.document ?? null}
        empty={
          credential.credentialType?.requiresDocument
            ? t(
                "No document attached. This type needs one before it can be verified; add it with Edit.",
              )
            : t("No document attached. Add one with Edit so it can be verified.")
        }
      />

      <DescriptionList layout="ruled" columns={2}>
        <DescriptionItem label={t("Number")}>
          {credential.number ? (
            <span className="font-mono">{credential.number}</span>
          ) : (
            <DescriptionEmpty />
          )}
        </DescriptionItem>
        <DescriptionItem label={t("Issuing authority")}>
          {credential.issuingAuthority || <DescriptionEmpty />}
        </DescriptionItem>
        <DescriptionItem label={t("Issued")} numeric>
          {credential.issuedAt ? formatUnixDateMedium(credential.issuedAt) : <DescriptionEmpty />}
        </DescriptionItem>
        <DescriptionItem label={t("Expires")} numeric>
          {credential.expiresAt ? formatUnixDateMedium(credential.expiresAt) : t("No expiry")}
          {credential.expiresAt ? (
            <span className={cn("block text-xs", expiryToneClass(credential.health))}>
              {expiryPhrase(t, credential.daysUntilExpiry)}
            </span>
          ) : null}
        </DescriptionItem>
        <DescriptionItem label={t("Verified by")}>
          {credential.verifiedAt ? (
            <>
              {credential.verifiedBy?.name ?? t("Verified")}
              <span className="block text-xs text-muted-foreground tabular-nums">
                {formatUnixDateMedium(credential.verifiedAt)}
              </span>
            </>
          ) : (
            <span className="text-foreground-subtle">{t("Not verified")}</span>
          )}
        </DescriptionItem>
        <DescriptionItem label={t("Notes")}>
          {credential.notes ? (
            <span className="whitespace-pre-line">{credential.notes}</span>
          ) : (
            <span className="text-foreground-subtle">{t("None")}</span>
          )}
        </DescriptionItem>
      </DescriptionList>

      <EarlierVersions credential={credential} workerId={workerId} loadedEarlier={loadedEarlier} />
    </>
  );
}

/**
 * The versions this credential replaced, read for its type alone, so they are all here even when
 * the tab's list stopped at the server's cap. What that list already held shows until the read
 * lands; the read sits under the credentials key, so a write that refreshes them refreshes this.
 */
function EarlierVersions({
  credential,
  workerId,
  loadedEarlier,
}: {
  credential: WorkerCredentialRow;
  workerId: string;
  loadedEarlier: readonly WorkerCredentialRow[];
}) {
  const t = useT();
  const code = credential.credentialType?.code ?? null;
  const versions = useQuery({
    queryKey: [WORKER_CREDENTIALS_KEY, workerId, "type", code],
    queryFn: ({ signal }) =>
      fetchWorkerCredentials({ workerId, credentialTypeCode: code ?? undefined }, { signal }),
    enabled: code !== null,
    select: (rows) => earlierVersions(rows, credential),
  });
  const earlier = versions.data ?? loadedEarlier;

  return (
    <section aria-label={t("Earlier versions")} className="flex flex-col gap-2">
      <h4 className="text-sm font-semibold">{t("Earlier versions")}</h4>
      {versions.isError ? (
        <ErrorState
          layout="compact"
          error={versions.error}
          title={t("The earlier versions could not be loaded")}
          onRetry={() => void versions.refetch()}
        />
      ) : earlier.length === 0 ? (
        <p className="text-xs text-muted-foreground">{t("No earlier versions.")}</p>
      ) : (
        <ul className="rounded-md border border-border-subtle">
          {earlier.map((version) => (
            <li
              key={version.id}
              className="flex items-center gap-2.5 px-2.5 py-2 text-xs not-first:border-t not-first:border-border-subtle"
            >
              <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                <span className="font-mono text-xs">
                  {version.issuedAt && version.expiresAt
                    ? formatRange(version.issuedAt, version.expiresAt, {
                        year: "always",
                        join: "words",
                      })
                    : version.expiresAt
                      ? t("Until {0}", formatUnixDateMedium(version.expiresAt))
                      : t("No expiry")}
                </span>
                {version.document?.originalName || version.archiveReason ? (
                  <span
                    className="truncate text-xs text-muted-foreground"
                    title={version.document?.originalName}
                  >
                    {[version.document?.originalName, version.archiveReason]
                      .filter(Boolean)
                      .join(" · ")}
                  </span>
                ) : null}
              </span>
              {version.document ? (
                <a
                  href={documentContentUrl(version.document.id, "view")}
                  target="_blank"
                  rel="noopener noreferrer"
                  aria-label={t("Open {0}", version.document.originalName)}
                  className="shrink-0 rounded-sm ui-focus-ring"
                >
                  <DocumentThumbnail
                    fileType={version.document.fileType}
                    fileName={version.document.originalName}
                    previewStatus={version.document.previewStatus}
                    previewUrl={documentContentUrl(version.document.id, "preview")}
                    size="sm"
                    className="border border-border-subtle"
                  />
                </a>
              ) : null}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
