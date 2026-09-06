import type { WorkerCredentialSummaryItem } from "@/lib/graphql/worker-credential";
import { CredentialHealthBadge } from "@trenova/shared/components/credential-health-badge";
import { Button } from "@trenova/shared/components/ui/button";
import { credentialHealthMeta, describeDaysUntil } from "@trenova/shared/lib/credential";
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

type CredentialSlotCardProps = {
  item: WorkerCredentialSummaryItem;
  permissions: CredentialSlotPermissions;
  verifying?: boolean;
  onAdd: (typeId: string) => void;
  onRenew: (item: WorkerCredentialSummaryItem) => void;
  onEdit: (item: WorkerCredentialSummaryItem) => void;
  onVerify: (item: WorkerCredentialSummaryItem) => void;
  onArchive: (item: WorkerCredentialSummaryItem) => void;
};

export function CredentialSlotCard({
  item,
  permissions,
  verifying = false,
  onAdd,
  onRenew,
  onEdit,
  onVerify,
  onArchive,
}: CredentialSlotCardProps) {
  const { credentialType: type, credential, health } = item;
  const meta = credentialHealthMeta(health);
  const verified = Boolean(credential?.verifiedAt);

  return (
    <div
      data-testid={`credential-slot-${type.id}`}
      data-health={health}
      className={cn(
        "group bg-card flex flex-col gap-2.5 rounded-xl border p-3 transition-shadow hover:shadow-sm",
        meta.ringClass,
      )}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="text-muted-foreground text-[10px] font-medium tracking-wide uppercase">
            {CREDENTIAL_CATEGORY_LABELS[type.category] ?? type.category}
            {item.required ? " · Required" : ""}
          </p>
          <p className="truncate text-sm font-semibold">{type.name}</p>
        </div>
        <CredentialHealthBadge health={health} daysUntilExpiry={item.daysUntilExpiry} />
      </div>

      {credential ? (
        <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5 text-xs">
          {credential.number ? (
            <>
              <dt className="text-muted-foreground">Number</dt>
              <dd className="truncate font-medium tabular-nums">{credential.number}</dd>
            </>
          ) : null}
          {credential.issuingAuthority ? (
            <>
              <dt className="text-muted-foreground">Issued by</dt>
              <dd className="truncate">{credential.issuingAuthority}</dd>
            </>
          ) : null}
          <dt className="text-muted-foreground">Expires</dt>
          <dd className={cn("font-medium", meta.textClass)}>
            {credential.expiresAt ? formatUnixDateMedium(credential.expiresAt) : "Never"}
            <span className="text-muted-foreground ml-1 font-normal" aria-hidden>
              ·
            </span>
            <span className="text-muted-foreground ml-1 font-normal">
              {describeDaysUntil(item.daysUntilExpiry)}
            </span>
          </dd>
        </dl>
      ) : (
        <p className="text-muted-foreground text-xs">
          Nothing on file. Add the {type.name.toLowerCase()} to complete the qualification file.
        </p>
      )}

      <div className="mt-auto flex items-center justify-between gap-2 pt-1">
        <div className="flex min-w-0 items-center gap-2 text-[11px]">
          {credential ? (
            verified ? (
              <span className="flex items-center gap-1 text-green-700 dark:text-green-400">
                <ShieldCheckIcon className="size-3.5" />
                Verified
                {credential.verifiedBy?.name ? (
                  <span className="text-muted-foreground">by {credential.verifiedBy.name}</span>
                ) : null}
              </span>
            ) : (
              <span className="text-muted-foreground">Unverified</span>
            )
          ) : null}
          {credential?.document ? (
            <span
              className="text-muted-foreground flex min-w-0 items-center gap-1"
              title={credential.document.originalName}
            >
              <PaperclipIcon className="size-3.5 shrink-0" />
              <span className="truncate">{credential.document.originalName}</span>
            </span>
          ) : null}
        </div>

        <div className="flex shrink-0 items-center gap-0.5 opacity-70 transition-opacity group-hover:opacity-100">
          {!credential && permissions.canCreate ? (
            <Button
              size="sm"
              variant="outline"
              aria-label={`Add ${type.name}`}
              onClick={() => onAdd(type.id)}
            >
              <PlusIcon className="size-3.5" />
              Add
            </Button>
          ) : null}
          {credential && !verified && permissions.canVerify ? (
            <Button
              size="icon"
              variant="ghost"
              className="size-7"
              aria-label={`Verify ${type.name}`}
              title="Mark as verified"
              disabled={verifying}
              onClick={() => onVerify(item)}
            >
              <ShieldCheckIcon className="size-3.5" />
            </Button>
          ) : null}
          {credential && permissions.canUpdate ? (
            <Button
              size="icon"
              variant="ghost"
              className="size-7"
              aria-label={`Edit ${type.name}`}
              title="Edit"
              onClick={() => onEdit(item)}
            >
              <PencilIcon className="size-3.5" />
            </Button>
          ) : null}
          {credential && permissions.canCreate ? (
            <Button
              size="icon"
              variant="ghost"
              className="size-7"
              aria-label={`Renew ${type.name}`}
              title="Renew"
              onClick={() => onRenew(item)}
            >
              <RefreshCwIcon className="size-3.5" />
            </Button>
          ) : null}
          {credential && permissions.canArchive ? (
            <Button
              size="icon"
              variant="ghost"
              className="text-muted-foreground hover:text-destructive size-7"
              aria-label={`Archive ${type.name}`}
              title="Archive"
              onClick={() => onArchive(item)}
            >
              <ArchiveIcon className="size-3.5" />
            </Button>
          ) : null}
        </div>
      </div>
    </div>
  );
}
