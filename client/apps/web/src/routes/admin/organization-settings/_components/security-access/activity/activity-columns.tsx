import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import type {
  AuthEventRow,
  ExternalIdentityRow,
  MFAAuthenticatorRow,
  RiskDecisionRow,
} from "@/lib/graphql/security-activity-table";
import { Badge } from "@trenova/shared/components/ui/badge";
import { defineLabels } from "@trenova/shared/i18n/labels";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { authEventLabel } from "../activity-copy";
import { outcomeVariant, riskVariant } from "../utils";

const AUTH_OUTCOME_LABELS = defineLabels({
  success: "Succeeded",
  challenge: "Challenged",
  denied: "Refused",
  failed: "Failed",
} as const);

const RISK_OUTCOME_LABELS = defineLabels({
  allow: "Allowed",
  challenge: "Challenged",
  deny: "Blocked",
} as const);

const AUTHENTICATOR_TYPE_LABELS = defineLabels({
  totp: "Authenticator app",
  webauthn: "Security key",
} as const);

function labelFor<T extends Record<string, string>>(labels: T, value: string): string {
  return value in labels ? labels[value as keyof T] : value;
}

function optionsOf<T extends Record<string, string>>(labels: T) {
  return Object.entries(labels).map(([value, label]) => ({ value, label }));
}

type ActivityUser = {
  id: string;
  name: string;
  emailAddress: string;
  profilePicUrl: string;
  thumbnailUrl: string;
} | null;

/** Who a row is about: their avatar, name and email, or a plain line when nobody was known. */
function UserCell({ user, unknown }: { user: ActivityUser | undefined; unknown: string }) {
  if (!user) {
    return <span className="text-muted-foreground">{unknown}</span>;
  }

  return (
    <div className="flex min-w-0 items-center gap-2.5">
      <ResolvedUserAvatar
        userId={user.id}
        name={user.name}
        profilePicUrl={user.profilePicUrl}
        thumbnailUrl={user.thumbnailUrl}
        className="bg-muted size-6 rounded-md"
        imageClassName="rounded-md bg-muted"
        fallbackClassName="text-2xs"
        alt={user.name}
      />
      <div className="flex min-w-0 flex-col">
        <span className="truncate">{user.name}</span>
        <span className="text-muted-foreground truncate text-xs">{user.emailAddress}</span>
      </div>
    </div>
  );
}

function userColumn<TData extends { user?: ActivityUser }>(
  t: TranslateFn,
  unknown: string,
): ColumnDef<TData> {
  return {
    id: "user",
    accessorKey: "user",
    header: t("Person"),
    cell: ({ row }) => <UserCell user={row.original.user} unknown={unknown} />,
    size: 240,
    minSize: 200,
    maxSize: 320,
    meta: {
      label: t("Person"),
      apiField: "user.name",
      filterable: true,
      sortable: false,
      filterType: "text",
      defaultFilterOperator: "contains",
    },
  };
}

function timestampColumn<TData extends Record<string, unknown>>(
  t: TranslateFn,
  { key, label, never }: { key: keyof TData & string; label: string; never?: string },
): ColumnDef<TData> {
  return {
    accessorKey: key,
    header: label,
    cell: ({ row }) => {
      const value = row.original[key];
      if (typeof value !== "number" || value <= 0) {
        return <span className="text-muted-foreground">{never ?? "—"}</span>;
      }
      return <HoverCardTimestamp timestamp={value} />;
    },
    size: 190,
    minSize: 160,
    maxSize: 240,
    meta: {
      label,
      apiField: key,
      filterable: true,
      sortable: true,
      filterType: "date",
      defaultFilterOperator: "daterange",
    },
  };
}

export function authEventColumns(t: TranslateFn): ColumnDef<AuthEventRow>[] {
  return [
    {
      id: "event",
      accessorKey: "provider",
      header: t("Event"),
      cell: ({ row }) => <span className="truncate">{authEventLabel(t, row.original)}</span>,
      size: 260,
      minSize: 200,
      maxSize: 360,
      meta: {
        label: t("Provider"),
        apiField: "provider",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "outcome",
      header: t("Outcome"),
      cell: ({ row }) => (
        <Badge variant={outcomeVariant(row.original.outcome)}>
          {labelFor(AUTH_OUTCOME_LABELS, row.original.outcome)}
        </Badge>
      ),
      size: 130,
      minSize: 110,
      maxSize: 160,
      meta: {
        label: t("Outcome"),
        apiField: "outcome",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: optionsOf(AUTH_OUTCOME_LABELS),
        defaultFilterOperator: "eq",
      },
    },
    userColumn<AuthEventRow>(t, t("Not identified")),
    {
      accessorKey: "riskOutcome",
      header: t("Risk check"),
      cell: ({ row }) => (
        <Badge variant={riskVariant(row.original.riskOutcome)}>
          {labelFor(RISK_OUTCOME_LABELS, row.original.riskOutcome)}
        </Badge>
      ),
      size: 130,
      minSize: 110,
      maxSize: 160,
      meta: {
        label: t("Risk check"),
        apiField: "riskOutcome",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: optionsOf(RISK_OUTCOME_LABELS),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "ipAddress",
      header: t("IP address"),
      cell: ({ row }) =>
        row.original.ipAddress ? (
          <span className="font-mono text-xs tabular-nums">{row.original.ipAddress}</span>
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
      size: 150,
      minSize: 120,
      maxSize: 200,
      meta: {
        label: t("IP address"),
        apiField: "ipAddress",
        filterable: false,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "errorCode",
      header: t("Detail"),
      cell: ({ row }) => {
        const detail = row.original.errorCode || row.original.riskSignals.join(", ");
        return detail ? (
          <span className="text-muted-foreground truncate">{detail}</span>
        ) : (
          <span className="text-muted-foreground">—</span>
        );
      },
      size: 220,
      minSize: 160,
      maxSize: 360,
      meta: {
        label: t("Error code"),
        apiField: "errorCode",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "userAgent",
      header: t("Browser"),
      cell: ({ row }) => (
        <span className="text-muted-foreground truncate text-xs">
          {row.original.userAgent || "—"}
        </span>
      ),
      size: 260,
      minSize: 180,
      maxSize: 420,
      meta: {
        label: t("Browser"),
        apiField: "userAgent",
        filterable: true,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    timestampColumn<AuthEventRow>(t, { key: "occurredAt", label: t("When") }),
  ];
}

export function riskDecisionColumns(t: TranslateFn): ColumnDef<RiskDecisionRow>[] {
  return [
    {
      accessorKey: "outcome",
      header: t("Decision"),
      cell: ({ row }) => (
        <Badge variant={riskVariant(row.original.outcome)}>
          {labelFor(RISK_OUTCOME_LABELS, row.original.outcome)}
        </Badge>
      ),
      size: 130,
      minSize: 110,
      maxSize: 160,
      meta: {
        label: t("Decision"),
        apiField: "outcome",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: optionsOf(RISK_OUTCOME_LABELS),
        defaultFilterOperator: "eq",
      },
    },
    userColumn<RiskDecisionRow>(t, t("Not identified")),
    {
      accessorKey: "reason",
      header: t("Reason"),
      cell: ({ row }) => <span className="truncate">{row.original.reason || "—"}</span>,
      size: 320,
      minSize: 220,
      maxSize: 480,
      meta: {
        label: t("Reason"),
        apiField: "reason",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    {
      accessorKey: "signals",
      header: t("Signals"),
      cell: ({ row }) =>
        row.original.signals.length > 0 ? (
          <div className="flex min-w-0 flex-wrap gap-1">
            {row.original.signals.map((signal) => (
              <Badge key={signal} variant="neutral" appearance="outline">
                {signal}
              </Badge>
            ))}
          </div>
        ) : (
          <span className="text-muted-foreground">—</span>
        ),
      size: 260,
      minSize: 180,
      maxSize: 400,
      meta: {
        label: t("Signals"),
        apiField: "signals",
        filterable: false,
        sortable: false,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    timestampColumn<RiskDecisionRow>(t, { key: "createdAt", label: t("When") }),
  ];
}

export function externalIdentityColumns(t: TranslateFn): ColumnDef<ExternalIdentityRow>[] {
  return [
    {
      accessorKey: "externalEmail",
      header: t("Identity"),
      cell: ({ row }) => (
        <div className="flex min-w-0 flex-col">
          <span className="truncate">{row.original.externalEmail || "—"}</span>
          {row.original.externalUsername ? (
            <span className="text-muted-foreground truncate text-xs">
              {row.original.externalUsername}
            </span>
          ) : null}
        </div>
      ),
      size: 260,
      minSize: 200,
      maxSize: 360,
      meta: {
        label: t("Email"),
        apiField: "externalEmail",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    userColumn<ExternalIdentityRow>(t, t("Unknown person")),
    {
      accessorKey: "externalSubject",
      header: t("Subject"),
      cell: ({ row }) => (
        <span className="truncate font-mono text-xs">{row.original.externalSubject}</span>
      ),
      size: 240,
      minSize: 180,
      maxSize: 360,
      meta: {
        label: t("Subject"),
        apiField: "externalSubject",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    timestampColumn<ExternalIdentityRow>(t, {
      key: "lastLoginAt",
      label: t("Last sign-in"),
      never: t("Never"),
    }),
    timestampColumn<ExternalIdentityRow>(t, { key: "createdAt", label: t("Linked") }),
  ];
}

export function mfaAuthenticatorColumns(t: TranslateFn): ColumnDef<MFAAuthenticatorRow>[] {
  return [
    {
      accessorKey: "name",
      header: t("Device"),
      cell: ({ row }) => (
        <div className="flex min-w-0 flex-col">
          <span className="truncate">{row.original.name}</span>
          <span className="text-muted-foreground truncate text-xs">
            {labelFor(AUTHENTICATOR_TYPE_LABELS, row.original.type)}
          </span>
        </div>
      ),
      size: 240,
      minSize: 180,
      maxSize: 320,
      meta: {
        label: t("Device"),
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
        defaultFilterOperator: "contains",
      },
    },
    userColumn<MFAAuthenticatorRow>(t, t("Unknown person")),
    {
      accessorKey: "enabled",
      header: t("Status"),
      cell: ({ row }) =>
        row.original.enabled ? (
          <Badge variant="success">{t("On")}</Badge>
        ) : row.original.verifiedAt > 0 ? (
          <Badge variant="neutral">{t("Off")}</Badge>
        ) : (
          <Badge variant="warning">{t("Not finished")}</Badge>
        ),
      size: 120,
      minSize: 100,
      maxSize: 160,
      meta: {
        label: t("On"),
        apiField: "enabled",
        filterable: true,
        sortable: true,
        filterType: "boolean",
        defaultFilterOperator: "eq",
      },
    },
    timestampColumn<MFAAuthenticatorRow>(t, {
      key: "verifiedAt",
      label: t("Set up"),
      never: t("Not finished"),
    }),
    timestampColumn<MFAAuthenticatorRow>(t, {
      key: "lastUsedAt",
      label: t("Last used"),
      never: t("Never"),
    }),
  ];
}
