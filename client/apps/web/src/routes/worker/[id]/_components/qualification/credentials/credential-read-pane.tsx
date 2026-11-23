import { WorkPane, WorkPaneBody, WorkPaneFooter } from "@/components/work-pane";
import type { WorkerCredentialRow } from "@/lib/graphql/worker-credential";
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
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { CredentialPermissions } from "./credential-table";
import { expiryPhrase, expiryToneClass, healthLabel, healthTone } from "./credential-model";

/**
 * One credential read beside the table: its document, the facts on the card, the versions it
 * replaced, and its actions in the footer. Verify sits there only while nobody has checked it.
 */
export function CredentialReadPane({
  open,
  onOpenChange,
  credential,
  loading,
  earlier,
  permissions,
  verifying,
  onVerify,
  onEdit,
  onRenew,
  onArchive,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  credential: WorkerCredentialRow | null;
  loading: boolean;
  earlier: readonly WorkerCredentialRow[];
  permissions: CredentialPermissions;
  verifying: boolean;
  onVerify: (credential: WorkerCredentialRow) => Promise<void>;
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
      contentKey={credential?.id}
    >
      <WorkPaneBody>
        {credential ? (
          <CredentialDetails credential={credential} earlier={earlier} />
        ) : loading ? (
          <div aria-busy aria-label={t("Loading the credential")} className="flex flex-col gap-3">
            <div aria-hidden className="contents">
              <span className="block h-42 ui-shimmer rounded-md" />
              <span className="block h-32 ui-shimmer rounded-md" />
            </div>
          </div>
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
            <Button type="button" size="sm" variant="outline" onClick={() => onEdit(credential)}>
              {t("Edit")}
            </Button>
          ) : null}
          {permissions.canCreate ? (
            <Button type="button" size="sm" onClick={() => onRenew(credential)}>
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
  earlier,
}: {
  credential: WorkerCredentialRow;
  earlier: readonly WorkerCredentialRow[];
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

      <section aria-label={t("Earlier versions")} className="flex flex-col gap-2">
        <h4 className="text-sm font-semibold">{t("Earlier versions")}</h4>
        {earlier.length === 0 ? (
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
                      ? t(
                          "{0} to {1}",
                          formatUnixDateMedium(version.issuedAt),
                          formatUnixDateMedium(version.expiresAt),
                        )
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
    </>
  );
}
