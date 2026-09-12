import { useT } from "@trenova/shared/i18n/use-t";
import type { WorkerCredentialSummaryItem } from "@/lib/graphql/worker-credential";
import { RowActionsMenu, type RowAction } from "@/components/row-actions-menu";
import { CredentialHealthBadge } from "@trenova/shared/components/credential-health-badge";
import { describeDaysUntil } from "@trenova/shared/lib/credential";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { CREDENTIAL_CATEGORY_LABELS } from "@trenova/shared/types/worker-credential";
import {
  ArchiveIcon,
  PaperclipIcon,
  PencilIcon,
  PlusIcon,
  RefreshCwIcon,
  ShieldCheckIcon,
} from "lucide-react";

export type CredentialSlotPermissions = {
  canCreate: boolean;
  canUpdate: boolean;
  canVerify: boolean;
  canArchive: boolean;
};

type CredentialSlotRowProps = {
  item: WorkerCredentialSummaryItem;
  permissions: CredentialSlotPermissions;
  verifying?: boolean;
  onAdd: (typeId: string) => void;
  onRenew: (item: WorkerCredentialSummaryItem) => void;
  onEdit: (item: WorkerCredentialSummaryItem) => void;
  onVerify: (item: WorkerCredentialSummaryItem) => void;
  onArchive: (item: WorkerCredentialSummaryItem) => void;
};

/**
 * One credential type as a row in the file: what it is, what is on file,
 * when it runs out, and how it stands. The health badge is the only colour;
 * a missing slot reads as an empty line, not an alarm.
 */
export function CredentialSlotRow({
  item,
  permissions,
  verifying = false,
  onAdd,
  onRenew,
  onEdit,
  onVerify,
  onArchive,
}: CredentialSlotRowProps) {
  const t = useT();

  const { credentialType: type, credential, health } = item;
  const verified = Boolean(credential?.verifiedAt);
  const caption = [
    CREDENTIAL_CATEGORY_LABELS[type.category] ?? type.category,
    credential?.number,
    credential?.issuingAuthority,
  ]
    .filter(Boolean)
    .join(" · ");

  const actions: RowAction[] = [];
  if (!credential && permissions.canCreate) {
    actions.push({
      id: "add",
      label: `Add ${type.name}`,
      icon: PlusIcon,
      onSelect: () => onAdd(type.id),
    });
  }
  if (credential) {
    if (!verified && permissions.canVerify) {
      actions.push({
        id: "verify",
        label: `Verify ${type.name}`,
        icon: ShieldCheckIcon,
        disabled: verifying,
        onSelect: () => onVerify(item),
      });
    }
    if (permissions.canUpdate) {
      actions.push({
        id: "edit",
        label: `Edit ${type.name}`,
        icon: PencilIcon,
        onSelect: () => onEdit(item),
      });
    }
    if (permissions.canCreate) {
      actions.push({
        id: "renew",
        label: `Renew ${type.name}`,
        icon: RefreshCwIcon,
        onSelect: () => onRenew(item),
      });
    }
    if (permissions.canArchive) {
      actions.push({
        id: "archive",
        label: `Archive ${type.name}`,
        icon: ArchiveIcon,
        destructive: true,
        onSelect: () => onArchive(item),
      });
    }
  }

  return (
    <div
      data-testid={`credential-slot-${type.id}`}
      data-health={health}
      className="group hover:bg-muted/30 grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-4 gap-y-1 px-3 py-2.5 transition-colors sm:grid-cols-[minmax(0,1fr)_minmax(0,11rem)_auto]"
    >
      <div className="min-w-0">
        <p className={cn("truncate text-sm font-medium", !credential && "text-muted-foreground")}>
          {type.name}
        </p>
        <p className="text-muted-foreground flex min-w-0 items-center gap-1.5 truncate text-xs">
          <span className="truncate">{caption}</span>
          {credential ? (
            verified ? (
              <span className="flex shrink-0 items-center gap-1">
                <ShieldCheckIcon className="size-3" />
                <span>{t("Verified")}</span>
                {credential.verifiedBy?.name ? <span>{t("by {0}", credential.verifiedBy.name)}</span> : null}
              </span>
            ) : (
              <span className="shrink-0">{t("Unverified")}</span>
            )
          ) : null}
          {credential?.document ? (
            <span
              className="flex min-w-0 shrink items-center gap-1"
              title={credential.document.originalName}
            >
              <PaperclipIcon className="size-3 shrink-0" />
              <span className="truncate">{credential.document.originalName}</span>
            </span>
          ) : null}
        </p>
      </div>

      <div className="col-span-2 text-xs sm:col-span-1">
        {credential ? (
          <>
            <p className="font-medium tabular-nums">
              {credential.expiresAt ? formatUnixDateMedium(credential.expiresAt) : t("No expiry")}
            </p>
            <p className="text-muted-foreground">{describeDaysUntil(item.daysUntilExpiry)}</p>
          </>
        ) : (
          <p className="text-muted-foreground">{t("Nothing on file")}</p>
        )}
      </div>

      <div className="col-start-2 row-start-1 flex items-center justify-end gap-2 sm:col-start-3">
        <CredentialHealthBadge health={health} daysUntilExpiry={item.daysUntilExpiry} />
        <RowActionsMenu label={`Actions for ${type.name}`} actions={actions} />
      </div>
    </div>
  );
}
