import { Badge } from "@trenova/shared/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { activityViewParser, type ActivityViewValue } from "@/hooks/use-organization-setting-state";
import { formatUnixDateTimeOrDash } from "@trenova/shared/lib/date";
import { queries } from "@/lib/queries";
import { cn, formatIdentityProviderName, toTitleCase } from "@trenova/shared/lib/utils";
import type { AuthEvent, ExternalIdentity, MFAAuthenticator, RiskDecision } from "@trenova/shared/types/iam";
import { useQuery } from "@tanstack/react-query";
import {
  ActivityIcon,
  ExternalLinkIcon,
  KeyRoundIcon,
  LockKeyholeIcon,
  ShieldAlertIcon,
} from "lucide-react";
import { useQueryState } from "nuqs";
import type { ReactNode } from "react";
import { memo, useCallback, useMemo, useState } from "react";
import { OuterContent } from "./layout";
import { ConsoleToolbar, EmptyState, ErrorState, RowSkeleton } from "./shared";
import { outcomeVariant, riskVariant } from "./utils";

const activityViewOptions: Array<{ value: ActivityViewValue; label: string }> = [
  { value: "auth", label: "Auth Events" },
  { value: "risk", label: "Risk Decisions" },
  { value: "identities", label: "External identities" },
  { value: "mfa", label: "MFA authenticators" },
];

const activityViewButtonLabels: Record<ActivityViewValue, string> = {
  auth: "Auth events",
  risk: "Risk decisions",
  identities: "External identities",
  mfa: "MFA authenticators",
};

function getActivityViewIcon(view: ActivityViewValue) {
  switch (view) {
    case "auth":
      return KeyRoundIcon;
    case "risk":
      return ShieldAlertIcon;
    case "identities":
      return ExternalLinkIcon;
    case "mfa":
      return LockKeyholeIcon;
  }
}

export function ActivityTab({ organizationId }: { organizationId: string }) {
  const [activityView, setActivityView] = useQueryState("activityView", activityViewParser);
  const [search, setSearch] = useState("");
  const normalizedSearch = useMemo(() => search.trim().toLowerCase(), [search]);
  const handleActivityViewChange = useCallback(
    (value: ActivityViewValue) => {
      void setActivityView(value);
    },
    [setActivityView],
  );

  return (
    <OuterContent>
      <ConsoleToolbar
        title="Activity console"
        description="Authentication outcomes, risk decisions, linked identities, and MFA devices."
        search={search}
        onSearchChange={setSearch}
        searchPlaceholder="Search activity"
        action={
          <ActivityViewSelect value={activityView} onValueChange={handleActivityViewChange} />
        }
      />
      <ActivityViewControls
        organizationId={organizationId}
        activeView={activityView}
        onViewChange={handleActivityViewChange}
      />
      <ActivityTableSection
        organizationId={organizationId}
        view={activityView}
        search={normalizedSearch}
      />
    </OuterContent>
  );
}

const ActivityViewSelect = memo(function ActivityViewSelect({
  value,
  onValueChange,
}: {
  value: ActivityViewValue;
  onValueChange: (value: ActivityViewValue) => void;
}) {
  return (
    <Select
      value={value}
      items={activityViewOptions}
      onValueChange={(nextValue) => onValueChange(nextValue as ActivityViewValue)}
    >
      <SelectTrigger className="h-7 w-44 text-xs">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {activityViewOptions.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
});

const ActivityViewControls = memo(function ActivityViewControls({
  organizationId,
  activeView,
  onViewChange,
}: {
  organizationId: string;
  activeView: ActivityViewValue;
  onViewChange: (value: ActivityViewValue) => void;
}) {
  const authEventsQuery = useQuery(queries.organization.authEvents(organizationId));
  const riskQuery = useQuery(queries.organization.riskDecisions(organizationId));
  const externalIdentityQuery = useQuery(queries.organization.externalIdentities(organizationId));
  const mfaQuery = useQuery(queries.organization.mfaAuthenticators(organizationId));

  return (
    <div className="grid gap-3 md:grid-cols-4">
      <ActivityViewButton
        view="auth"
        active={activeView === "auth"}
        count={authEventsQuery.data?.length ?? 0}
        onViewChange={onViewChange}
      />
      <ActivityViewButton
        view="risk"
        active={activeView === "risk"}
        count={riskQuery.data?.length ?? 0}
        onViewChange={onViewChange}
      />
      <ActivityViewButton
        view="identities"
        active={activeView === "identities"}
        count={externalIdentityQuery.data?.length ?? 0}
        onViewChange={onViewChange}
      />
      <ActivityViewButton
        view="mfa"
        active={activeView === "mfa"}
        count={mfaQuery.data?.length ?? 0}
        onViewChange={onViewChange}
      />
    </div>
  );
});

const ActivityViewButton = memo(function ActivityViewButton({
  view,
  active,
  count,
  onViewChange,
}: {
  view: ActivityViewValue;
  active: boolean;
  count: number;
  onViewChange: (value: ActivityViewValue) => void;
}) {
  const Icon = getActivityViewIcon(view);

  return (
    <button
      type="button"
      className={cn(
        "flex items-center justify-between gap-2 rounded-lg border bg-background p-3 text-left transition-colors hover:border-brand hover:bg-brand/20",
        active && "border-brand bg-brand/20 hover:bg-brand/30",
      )}
      onClick={() => onViewChange(view)}
    >
      <span className="flex items-center gap-2 text-sm font-medium">
        <span className="text-muted-foreground [&_svg]:size-4">
          <Icon />
        </span>
        {activityViewButtonLabels[view]}
      </span>
      <Badge variant={active ? "info" : "outline"}>{count}</Badge>
    </button>
  );
});

function ActivityTableSection({
  organizationId,
  view,
  search,
}: {
  organizationId: string;
  view: ActivityViewValue;
  search: string;
}) {
  switch (view) {
    case "risk":
      return <RiskDecisionsSection organizationId={organizationId} search={search} />;
    case "identities":
      return <ExternalIdentitiesSection organizationId={organizationId} search={search} />;
    case "mfa":
      return <MFAAuthenticatorsSection organizationId={organizationId} search={search} />;
    case "auth":
    default:
      return <AuthEventsSection organizationId={organizationId} search={search} />;
  }
}

function AuthEventsSection({ organizationId, search }: { organizationId: string; search: string }) {
  const authEventsQuery = useQuery(queries.organization.authEvents(organizationId));
  const records = useMemo(
    () =>
      (authEventsQuery.data ?? []).filter((item) =>
        [item.provider, item.outcome, item.riskOutcome, item.ipAddress, item.errorCode]
          .join(" ")
          .toLowerCase()
          .includes(search),
      ),
    [authEventsQuery.data, search],
  );

  if (authEventsQuery.isError) {
    return <ErrorState label="Authentication events could not be loaded." />;
  }

  return <AuthEventsTable records={records} isLoading={authEventsQuery.isLoading} />;
}

function RiskDecisionsSection({
  organizationId,
  search,
}: {
  organizationId: string;
  search: string;
}) {
  const riskQuery = useQuery(queries.organization.riskDecisions(organizationId));
  const records = useMemo(
    () =>
      (riskQuery.data ?? []).filter((item) =>
        [item.outcome, item.reason, item.signals.join(" ")]
          .join(" ")
          .toLowerCase()
          .includes(search),
      ),
    [riskQuery.data, search],
  );

  if (riskQuery.isError) {
    return <ErrorState label="Risk decisions could not be loaded." />;
  }

  return <RiskDecisionsTable records={records} isLoading={riskQuery.isLoading} />;
}

function ExternalIdentitiesSection({
  organizationId,
  search,
}: {
  organizationId: string;
  search: string;
}) {
  const externalIdentityQuery = useQuery(queries.organization.externalIdentities(organizationId));
  const records = useMemo(
    () =>
      (externalIdentityQuery.data ?? []).filter((item) =>
        [item.externalEmail, item.externalUsername, item.externalSubject]
          .join(" ")
          .toLowerCase()
          .includes(search),
      ),
    [externalIdentityQuery.data, search],
  );

  if (externalIdentityQuery.isError) {
    return <ErrorState label="External identities could not be loaded." />;
  }

  return <ExternalIdentitiesTable records={records} isLoading={externalIdentityQuery.isLoading} />;
}

function MFAAuthenticatorsSection({
  organizationId,
  search,
}: {
  organizationId: string;
  search: string;
}) {
  const mfaQuery = useQuery(queries.organization.mfaAuthenticators(organizationId));
  const records = useMemo(
    () =>
      (mfaQuery.data ?? []).filter((item) =>
        [item.name, item.type, item.enabled ? "enabled" : "disabled"]
          .join(" ")
          .toLowerCase()
          .includes(search),
      ),
    [mfaQuery.data, search],
  );

  if (mfaQuery.isError) {
    return <ErrorState label="MFA authenticators could not be loaded." />;
  }

  return <MFAAuthenticatorsTable records={records} isLoading={mfaQuery.isLoading} />;
}

function AuthEventsTable({ records, isLoading }: { records: AuthEvent[]; isLoading: boolean }) {
  return (
    <ActivityTableShell
      isLoading={isLoading}
      rowCount={records.length}
      emptyLabel="No authentication events found."
      headers={["Provider", "Outcome", "Risk", "When", "Detail"]}
    >
      {records.map((item) => (
        <TableRow key={item.id}>
          <TableCell>
            <div className="font-medium">{formatIdentityProviderName(item.provider)}</div>
            <div className="text-xs text-muted-foreground">
              {item.ipAddress || "No IP captured"}
            </div>
          </TableCell>
          <TableCell>
            <Badge variant={outcomeVariant(item.outcome)}>{toTitleCase(item.outcome)}</Badge>
          </TableCell>
          <TableCell>
            <Badge variant={riskVariant(item.riskOutcome)}>{toTitleCase(item.riskOutcome)}</Badge>
          </TableCell>
          <TableCell className="text-muted-foreground">
            {formatUnixDateTimeOrDash(item.occurredAt)}
          </TableCell>
          <TableCell className="max-w-72 truncate text-muted-foreground">
            {item.errorCode || item.riskSignals.join(", ") || "-"}
          </TableCell>
        </TableRow>
      ))}
    </ActivityTableShell>
  );
}

function RiskDecisionsTable({
  records,
  isLoading,
}: {
  records: RiskDecision[];
  isLoading: boolean;
}) {
  return (
    <ActivityTableShell
      isLoading={isLoading}
      rowCount={records.length}
      emptyLabel="No risk decisions found."
      headers={["Outcome", "Reason", "Signals", "When"]}
    >
      {records.map((item) => (
        <TableRow key={item.id}>
          <TableCell>
            <Badge variant={riskVariant(item.outcome)}>{toTitleCase(item.outcome)}</Badge>
          </TableCell>
          <TableCell className="max-w-80 truncate">{item.reason || "-"}</TableCell>
          <TableCell className="max-w-80 truncate text-muted-foreground">
            {item.signals.join(", ") || "-"}
          </TableCell>
          <TableCell className="text-muted-foreground">
            {formatUnixDateTimeOrDash(item.createdAt)}
          </TableCell>
        </TableRow>
      ))}
    </ActivityTableShell>
  );
}

function ExternalIdentitiesTable({
  records,
  isLoading,
}: {
  records: ExternalIdentity[];
  isLoading: boolean;
}) {
  return (
    <ActivityTableShell
      isLoading={isLoading}
      rowCount={records.length}
      emptyLabel="No external identities found."
      headers={["Identity", "Subject", "Last login", "Created"]}
    >
      {records.map((item) => (
        <TableRow key={item.id}>
          <TableCell>
            <div className="font-medium">{item.externalEmail || "-"}</div>
            <div className="text-xs text-muted-foreground">{item.externalUsername || "-"}</div>
          </TableCell>
          <TableCell className="max-w-80 truncate">{item.externalSubject}</TableCell>
          <TableCell className="text-muted-foreground">
            {item.lastLoginAt ? formatUnixDateTimeOrDash(item.lastLoginAt) : "Never"}
          </TableCell>
          <TableCell className="text-muted-foreground">
            {formatUnixDateTimeOrDash(item.createdAt)}
          </TableCell>
        </TableRow>
      ))}
    </ActivityTableShell>
  );
}

function MFAAuthenticatorsTable({
  records,
  isLoading,
}: {
  records: MFAAuthenticator[];
  isLoading: boolean;
}) {
  return (
    <ActivityTableShell
      isLoading={isLoading}
      rowCount={records.length}
      emptyLabel="No MFA authenticators found."
      headers={["Authenticator", "Status", "Verified", "Last used"]}
    >
      {records.map((item) => (
        <TableRow key={item.id}>
          <TableCell>
            <div className="font-medium">{item.name}</div>
            <div className="text-xs text-muted-foreground">{item.type.toUpperCase()}</div>
          </TableCell>
          <TableCell>
            <Badge variant={item.enabled ? "active" : "inactive"}>
              {item.enabled ? "Enabled" : "Disabled"}
            </Badge>
          </TableCell>
          <TableCell className="text-muted-foreground">
            {item.verifiedAt ? formatUnixDateTimeOrDash(item.verifiedAt) : "Not verified"}
          </TableCell>
          <TableCell className="text-muted-foreground">
            {item.lastUsedAt ? formatUnixDateTimeOrDash(item.lastUsedAt) : "Never"}
          </TableCell>
        </TableRow>
      ))}
    </ActivityTableShell>
  );
}

function ActivityTableShell({
  isLoading,
  rowCount,
  emptyLabel,
  headers,
  children,
}: {
  isLoading: boolean;
  rowCount: number;
  emptyLabel: string;
  headers: string[];
  children: ReactNode;
}) {
  if (isLoading) {
    return <RowSkeleton rows={5} />;
  }

  if (rowCount === 0) {
    return (
      <EmptyState
        icon={<ActivityIcon />}
        label={emptyLabel}
        description="Try a different filter."
      />
    );
  }

  return (
    <div className="overflow-hidden rounded-lg border bg-background">
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              {headers.map((header) => (
                <TableHead key={header}>{header}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>{children}</TableBody>
        </Table>
      </div>
    </div>
  );
}
