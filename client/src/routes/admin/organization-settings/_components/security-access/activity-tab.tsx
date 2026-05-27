import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { formatUnixDateTimeOrDash } from "@/lib/date";
import { queries } from "@/lib/queries";
import { cn, formatIdentityProviderName, toTitleCase } from "@/lib/utils";
import type {
  AuthEvent,
  ExternalIdentity,
  MFAAuthenticator,
  RiskDecision,
} from "@/types/iam";
import { useQuery } from "@tanstack/react-query";
import {
  ActivityIcon,
  ExternalLinkIcon,
  KeyRoundIcon,
  LockKeyholeIcon,
  ShieldAlertIcon,
} from "lucide-react";
import type { ReactNode } from "react";
import { useState } from "react";
import { ConsoleToolbar, EmptyState, ErrorState, RowSkeleton } from "./shared";
import { outcomeVariant, riskVariant } from "./utils";

type ActivityView = "auth" | "risk" | "identities" | "mfa";

export function ActivityTab({ organizationId }: { organizationId: string }) {
  const authEventsQuery = useQuery(queries.organization.authEvents(organizationId));
  const riskQuery = useQuery(queries.organization.riskDecisions(organizationId));
  const externalIdentityQuery = useQuery(queries.organization.externalIdentities(organizationId));
  const mfaQuery = useQuery(queries.organization.mfaAuthenticators(organizationId));
  const [view, setView] = useState<ActivityView>("auth");
  const [search, setSearch] = useState("");

  const query = search.trim().toLowerCase();
  const authEvents = (authEventsQuery.data ?? []).filter((item) =>
    [item.provider, item.outcome, item.riskOutcome, item.ipAddress, item.errorCode]
      .join(" ")
      .toLowerCase()
      .includes(query),
  );
  const riskDecisions = (riskQuery.data ?? []).filter((item) =>
    [item.outcome, item.reason, item.signals.join(" ")].join(" ").toLowerCase().includes(query),
  );
  const identities = (externalIdentityQuery.data ?? []).filter((item) =>
    [item.externalEmail, item.externalUsername, item.externalSubject]
      .join(" ")
      .toLowerCase()
      .includes(query),
  );
  const authenticators = (mfaQuery.data ?? []).filter((item) =>
    [item.name, item.type, item.enabled ? "enabled" : "disabled"]
      .join(" ")
      .toLowerCase()
      .includes(query),
  );

  return (
    <div className="space-y-3">
      <ConsoleToolbar
        title="Activity console"
        description="Authentication outcomes, risk decisions, linked identities, and MFA devices."
        search={search}
        onSearchChange={setSearch}
        searchPlaceholder="Search activity"
        action={
          <Select
            value={view}
            onValueChange={(value) => setView((value ?? "auth") as ActivityView)}
          >
            <SelectTrigger className="h-8 w-44 bg-background text-xs">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="auth">Auth events</SelectItem>
              <SelectItem value="risk">Risk decisions</SelectItem>
              <SelectItem value="identities">External identities</SelectItem>
              <SelectItem value="mfa">MFA authenticators</SelectItem>
            </SelectContent>
          </Select>
        }
      />
      <div className="grid gap-3 md:grid-cols-4">
        <ActivityViewButton
          active={view === "auth"}
          icon={<KeyRoundIcon />}
          label="Auth events"
          count={authEventsQuery.data?.length ?? 0}
          onClick={() => setView("auth")}
        />
        <ActivityViewButton
          active={view === "risk"}
          icon={<ShieldAlertIcon />}
          label="Risk decisions"
          count={riskQuery.data?.length ?? 0}
          onClick={() => setView("risk")}
        />
        <ActivityViewButton
          active={view === "identities"}
          icon={<ExternalLinkIcon />}
          label="External identities"
          count={externalIdentityQuery.data?.length ?? 0}
          onClick={() => setView("identities")}
        />
        <ActivityViewButton
          active={view === "mfa"}
          icon={<LockKeyholeIcon />}
          label="MFA authenticators"
          count={mfaQuery.data?.length ?? 0}
          onClick={() => setView("mfa")}
        />
      </div>
      {view === "auth" &&
        (authEventsQuery.isError ? (
          <ErrorState label="Authentication events could not be loaded." />
        ) : (
          <AuthEventsTable records={authEvents} isLoading={authEventsQuery.isLoading} />
        ))}
      {view === "risk" &&
        (riskQuery.isError ? (
          <ErrorState label="Risk decisions could not be loaded." />
        ) : (
          <RiskDecisionsTable records={riskDecisions} isLoading={riskQuery.isLoading} />
        ))}
      {view === "identities" &&
        (externalIdentityQuery.isError ? (
          <ErrorState label="External identities could not be loaded." />
        ) : (
          <ExternalIdentitiesTable
            records={identities}
            isLoading={externalIdentityQuery.isLoading}
          />
        ))}
      {view === "mfa" &&
        (mfaQuery.isError ? (
          <ErrorState label="MFA authenticators could not be loaded." />
        ) : (
          <MFAAuthenticatorsTable records={authenticators} isLoading={mfaQuery.isLoading} />
        ))}
    </div>
  );
}

function ActivityViewButton({
  active,
  icon,
  label,
  count,
  onClick,
}: {
  active: boolean;
  icon: ReactNode;
  label: string;
  count: number;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      className={cn(
        "flex items-center justify-between gap-2 rounded-lg border bg-background p-3 text-left transition-colors hover:bg-muted/40",
        active && "border-primary/40 bg-primary/5",
      )}
      onClick={onClick}
    >
      <span className="flex items-center gap-2 text-sm font-medium">
        <span className="text-muted-foreground [&_svg]:size-4">{icon}</span>
        {label}
      </span>
      <Badge variant={active ? "info" : "outline"}>{count}</Badge>
    </button>
  );
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
