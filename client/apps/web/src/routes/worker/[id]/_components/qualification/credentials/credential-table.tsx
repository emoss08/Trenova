import { RowActionsMenu, type RowAction } from "@/components/row-actions-menu";
import type { WorkerCredentialSummaryItem } from "@/lib/graphql/worker-credential";
import {
  ArchiveIcon,
  Edit03Icon,
  EyeIcon,
  PaperclipIcon,
  PlusIcon,
  RefreshCw01Icon,
  ShieldTickIcon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { KeyboardEvent } from "react";
import {
  BOUNDED_CELL_CLASS,
  BoundedGroupRow,
  BoundedTable,
  type BoundedColumn,
} from "../bounded-table";
import { ROW_CHANGED_CLASS, type FlashPhase } from "../use-row-flash";
import {
  credentialCaption,
  expiryPhrase,
  expiryToneClass,
  healthLabel,
  healthTone,
} from "./credential-model";

export type CredentialPermissions = {
  canCreate: boolean;
  canUpdate: boolean;
  canVerify: boolean;
  canArchive: boolean;
};

export type CredentialRowHandlers = {
  /** A held credential's row: opens its read pane. */
  onOpen: (item: WorkerCredentialSummaryItem) => void;
  /** A required slot with nothing on file: opens the add form with its type. */
  onAdd: (item: WorkerCredentialSummaryItem) => void;
  onVerify: (item: WorkerCredentialSummaryItem) => Promise<void>;
  onRenew: (item: WorkerCredentialSummaryItem) => void;
  onEdit: (item: WorkerCredentialSummaryItem) => void;
  onArchive: (item: WorkerCredentialSummaryItem) => void;
};

/**
 * Every credential on the file in one bounded table: the required slots (held or not), then the
 * other credentials held. A row opens the credential's read pane; its menu carries the same
 * actions as the pane's footer.
 */
export function CredentialTable({
  required,
  other,
  permissions,
  verifyingId,
  flashOf,
  rowRef,
  handlers,
}: {
  required: readonly WorkerCredentialSummaryItem[];
  other: readonly WorkerCredentialSummaryItem[];
  permissions: CredentialPermissions;
  verifyingId: string | null;
  flashOf: (key: string) => FlashPhase | undefined;
  rowRef: (key: string, element: HTMLTableRowElement | null) => void;
  handlers: CredentialRowHandlers;
}) {
  const t = useT();
  const columns: BoundedColumn[] = [
    { key: "credential", label: t("Credential") },
    { key: "verified", label: t("Verified") },
    { key: "document", label: t("Document") },
    { key: "expires", label: t("Expires"), align: "end" },
    { key: "health", label: t("Health") },
    { key: "actions", label: <span className="sr-only">{t("Actions")}</span>, shrink: true },
  ];

  const rows = (items: readonly WorkerCredentialSummaryItem[]) =>
    items.map((item) => (
      <CredentialRow
        key={item.credentialType.id}
        item={item}
        permissions={permissions}
        verifying={item.credential != null && item.credential.id === verifyingId}
        flash={item.credential ? flashOf(item.credential.id) : undefined}
        rowRef={rowRef}
        handlers={handlers}
      />
    ));

  return (
    <BoundedTable columns={columns} label={t("Credentials on file")} maxHeightClassName="max-h-152">
      <BoundedGroupRow span={columns.length} label={t("Required")} count={required.length} />
      {required.length === 0 ? (
        <tr>
          <td
            colSpan={columns.length}
            className={cn(BOUNDED_CELL_CLASS, "text-xs text-muted-foreground")}
          >
            {t("No credential type is required for this worker.")}
          </td>
        </tr>
      ) : (
        rows(required)
      )}
      {other.length > 0 ? (
        <>
          <BoundedGroupRow span={columns.length} label={t("Other")} count={other.length} />
          {rows(other)}
        </>
      ) : null}
    </BoundedTable>
  );
}

function CredentialRow({
  item,
  permissions,
  verifying,
  flash,
  rowRef,
  handlers,
}: {
  item: WorkerCredentialSummaryItem;
  permissions: CredentialPermissions;
  verifying: boolean;
  flash: FlashPhase | undefined;
  rowRef: (key: string, element: HTMLTableRowElement | null) => void;
  handlers: CredentialRowHandlers;
}) {
  const t = useT();
  const { credentialType: type, credential } = item;
  const caption = credentialCaption(item);
  const opens = credential != null || permissions.canCreate;

  const actions: RowAction[] = [];
  if (credential) {
    actions.push({
      id: "open",
      label: t("Open"),
      icon: EyeIcon,
      onSelect: () => handlers.onOpen(item),
    });
    if (!credential.verifiedAt && permissions.canVerify) {
      actions.push({
        id: "verify",
        label: t("Verify"),
        icon: ShieldTickIcon,
        disabled: verifying,
        onSelect: () => void handlers.onVerify(item),
      });
    }
    if (permissions.canCreate) {
      actions.push({
        id: "renew",
        label: t("Renew"),
        icon: RefreshCw01Icon,
        onSelect: () => handlers.onRenew(item),
      });
    }
    if (permissions.canUpdate) {
      actions.push({
        id: "edit",
        label: t("Edit"),
        icon: Edit03Icon,
        onSelect: () => handlers.onEdit(item),
      });
    }
    if (permissions.canArchive) {
      actions.push({
        id: "archive",
        label: t("Archive"),
        icon: ArchiveIcon,
        destructive: true,
        onSelect: () => handlers.onArchive(item),
      });
    }
  } else if (permissions.canCreate) {
    actions.push({
      id: "add",
      label: t("Add {0}", type.name),
      icon: PlusIcon,
      onSelect: () => handlers.onAdd(item),
    });
  }

  const activate = () => {
    if (credential) handlers.onOpen(item);
    else if (permissions.canCreate) handlers.onAdd(item);
  };
  const onKeyDown = (event: KeyboardEvent<HTMLTableRowElement>) => {
    if (event.target !== event.currentTarget) return;
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      activate();
    }
  };

  return (
    <tr
      ref={(element) => rowRef(credential?.id ?? type.id, element)}
      data-glide-target
      data-testid={`credential-row-${type.id}`}
      data-health={item.health}
      data-changed={flash}
      tabIndex={opens ? 0 : undefined}
      aria-label={type.name}
      onClick={opens ? activate : undefined}
      onKeyDown={opens ? onKeyDown : undefined}
      className={cn(
        "outline-none focus-visible:[&>td]:bg-surface-hover",
        opens && "cursor-pointer",
        ROW_CHANGED_CLASS,
      )}
    >
      <td className={cn(BOUNDED_CELL_CLASS, "align-top")}>
        <span className={cn("block font-medium", !credential && "text-muted-foreground")}>
          {type.name}
        </span>
        <span className="block text-xs text-muted-foreground">
          {credential ? (
            <>
              {caption.category}
              {caption.number ? (
                <>
                  {" · "}
                  <span className="font-mono">{caption.number}</span>
                </>
              ) : null}
              {caption.authority ? ` · ${caption.authority}` : null}
            </>
          ) : (
            t("Nothing on file")
          )}
        </span>
      </td>
      <td className={BOUNDED_CELL_CLASS}>
        {credential?.verifiedAt ? (
          <span
            data-testid="credential-verified"
            className="inline-flex items-center gap-1.5 whitespace-nowrap"
          >
            <ShieldTickIcon
              aria-hidden
              className={cn(
                "size-3 text-success-foreground",
                flash && "animate-confirm motion-reduce:animate-none",
              )}
            />
            {credential.verifiedBy?.name ?? t("Verified")}
          </span>
        ) : credential ? (
          <span className="text-foreground-subtle">{t("Unverified")}</span>
        ) : (
          <span className="text-foreground-subtle">—</span>
        )}
      </td>
      <td className={BOUNDED_CELL_CLASS}>
        {credential?.document ? (
          <span className="inline-flex max-w-52 min-w-0 items-center gap-1.5">
            <PaperclipIcon aria-hidden className="size-3 shrink-0 text-foreground-subtle" />
            <span className="truncate">{credential.document.originalName}</span>
          </span>
        ) : (
          <span className="text-foreground-subtle">—</span>
        )}
      </td>
      <td className={cn(BOUNDED_CELL_CLASS, "text-right align-top")}>
        {credential?.expiresAt ? (
          <span className="block font-mono text-xs whitespace-nowrap">
            {formatUnixDateMedium(credential.expiresAt)}
          </span>
        ) : null}
        {credential ? (
          <span className={cn("block text-xs", expiryToneClass(item.health))}>
            {expiryPhrase(t, credential.expiresAt ? item.daysUntilExpiry : null)}
          </span>
        ) : null}
      </td>
      <td className={BOUNDED_CELL_CLASS}>
        <Badge variant={healthTone(item.health)} dot>
          {healthLabel(t, item.health)}
        </Badge>
      </td>
      <td
        className={BOUNDED_CELL_CLASS}
        onClick={(event) => event.stopPropagation()}
        onKeyDown={(event) => event.stopPropagation()}
      >
        <RowActionsMenu label={t("Actions for {0}", type.name)} actions={actions} />
      </td>
    </tr>
  );
}
