import { RowActionsMenu, type RowAction } from "@/components/row-actions-menu";
import type { PreloadIntent } from "@/components/work-pane";
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
import type { KeyboardEvent, Ref } from "react";
import {
  BOUNDED_CELL_CLASS,
  BoundedGroupRow,
  BoundedTable,
  type BoundedColumn,
  type BoundedRow,
  type BoundedRowProps,
  type BoundedTableHandle,
} from "../../bounded-table";
import { ROW_CHANGED_CLASS, type FlashPhase } from "../../use-row-flash";
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
  handleRef,
  rowIntent,
  menuIntent,
  handlers,
}: {
  required: readonly WorkerCredentialSummaryItem[];
  other: readonly WorkerCredentialSummaryItem[];
  permissions: CredentialPermissions;
  verifyingId: string | null;
  flashOf: (key: string) => FlashPhase | undefined;
  /** Brings a row into view by its type's id. */
  handleRef: Ref<BoundedTableHandle>;
  /** Reads the read pane's code as the pointer or focus reaches a row. */
  rowIntent: PreloadIntent;
  /** Reads the form pane's code as the pointer or focus reaches a row's menu. */
  menuIntent: PreloadIntent;
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

  const itemRow = (item: WorkerCredentialSummaryItem): BoundedRow => ({
    key: item.credentialType.id,
    render: (props) => (
      <CredentialRow
        props={props}
        item={item}
        permissions={permissions}
        verifying={item.credential != null && item.credential.id === verifyingId}
        flash={item.credential ? flashOf(item.credential.id) : undefined}
        rowIntent={rowIntent}
        menuIntent={menuIntent}
        handlers={handlers}
      />
    ),
  });
  const groupRow = (key: string, label: string, count: number): BoundedRow => ({
    key,
    render: (props) => (
      <BoundedGroupRow props={props} span={columns.length} label={label} count={count} />
    ),
  });

  const rows: BoundedRow[] = [
    groupRow("group:required", t("Required"), required.length),
    ...(required.length === 0
      ? [
          {
            key: "required:none",
            render: (props: BoundedRowProps) => (
              <tr {...props}>
                <td
                  colSpan={columns.length}
                  className={cn(BOUNDED_CELL_CLASS, "text-xs text-muted-foreground")}
                >
                  {t("No credential type is required for this worker.")}
                </td>
              </tr>
            ),
          },
        ]
      : required.map(itemRow)),
    ...(other.length > 0
      ? [groupRow("group:other", t("Other"), other.length), ...other.map(itemRow)]
      : []),
  ];

  return (
    <BoundedTable
      columns={columns}
      label={t("Credentials on file")}
      maxHeightClassName="max-h-152"
      estimateRowHeight={52}
      handleRef={handleRef}
      rows={rows}
    />
  );
}

function CredentialRow({
  props,
  item,
  permissions,
  verifying,
  flash,
  rowIntent,
  menuIntent,
  handlers,
}: {
  props: BoundedRowProps;
  item: WorkerCredentialSummaryItem;
  permissions: CredentialPermissions;
  verifying: boolean;
  flash: FlashPhase | undefined;
  rowIntent: PreloadIntent;
  menuIntent: PreloadIntent;
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
      preload: rowIntent.onFocus,
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
        preload: menuIntent.onFocus,
        onSelect: () => handlers.onRenew(item),
      });
    }
    if (permissions.canUpdate) {
      actions.push({
        id: "edit",
        label: t("Edit"),
        icon: Edit03Icon,
        preload: menuIntent.onFocus,
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
      preload: menuIntent.onFocus,
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
      {...props}
      {...(credential ? rowIntent : {})}
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
