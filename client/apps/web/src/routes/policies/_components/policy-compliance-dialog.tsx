import { EmptyState } from "@/components/empty-state";
import {
  fetchWorkerPolicyCompliance,
  POLICY_COMPLIANCE_KEY,
  type WorkerPolicyRow,
} from "@/lib/graphql/self-service";
import { useQuery } from "@tanstack/react-query";
import { Avatar, AvatarFallback } from "@trenova/shared/components/ui/avatar";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Input } from "@trenova/shared/components/ui/input";
import { RingGauge } from "@trenova/shared/components/ui/ring-gauge";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatShiftDate } from "@trenova/shared/lib/scheduling";
import { compliancePercent } from "@trenova/shared/lib/self-service";
import { cn, initials } from "@trenova/shared/lib/utils";
import { CheckIcon, FileSignatureIcon, SearchIcon, UsersIcon } from "lucide-react";
import { useState } from "react";

type Scope = "outstanding" | "signed" | "all";

const SCOPE_ITEMS = [
  { value: "outstanding", label: "Outstanding" },
  { value: "signed", label: "Signed" },
  { value: "all", label: "Everyone" },
] satisfies { value: Scope; label: string }[];

export type PolicyComplianceDialogProps = {
  policy: WorkerPolicyRow | null;
  onOpenChange: (open: boolean) => void;
};

function splitName(name: string): [string, string] {
  const parts = name.trim().split(/\s+/);
  return [parts[0] ?? "", parts.length > 1 ? parts[parts.length - 1] : ""];
}

/**
 * Who has signed the version in force and who has not. Derived on every read:
 * the roster and the signatures both move, and a stored answer would be wrong
 * by the next hire.
 */
export function PolicyComplianceDialog({ policy, onOpenChange }: PolicyComplianceDialogProps) {
  const [scope, setScope] = useState<Scope>("outstanding");
  const [search, setSearch] = useState("");

  const compliance = useQuery({
    queryKey: [POLICY_COMPLIANCE_KEY, policy?.id],
    queryFn: ({ signal }) => fetchWorkerPolicyCompliance(policy?.id ?? "", { signal }),
    enabled: Boolean(policy),
  });

  const view = compliance.data;
  const total = view ? view.signed + view.outstanding : 0;
  const percent = view ? compliancePercent(view.signed, view.outstanding) : 0;
  const needle = search.trim().toLowerCase();
  const rows = (view?.rows ?? []).filter((row) => {
    const done = Boolean(row.acknowledgedAt);
    if (scope === "outstanding" && done) return false;
    if (scope === "signed" && !done) return false;
    return needle === "" || row.workerName.toLowerCase().includes(needle);
  });

  return (
    <Sheet open={policy !== null} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-lg">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2">
            <FileSignatureIcon className="text-muted-foreground size-4" />
            {policy?.title ?? "Policy"}
          </SheetTitle>
          <SheetDescription>
            Version {view?.policy.versionLabel ?? policy?.versionLabel}. A signature on an earlier
            version does not count — those people are outstanding again.
          </SheetDescription>
        </SheetHeader>

        {compliance.isLoading || !view ? (
          <div className="flex flex-col gap-3 px-4">
            <Skeleton className="h-24 w-full rounded-xl" />
            <Skeleton className="h-48 w-full rounded-xl" />
          </div>
        ) : (
          <div className="flex min-h-0 flex-1 flex-col gap-3 px-4 pb-4">
            <section className="border-border/80 bg-card flex items-center gap-4 rounded-xl border p-4">
              <RingGauge
                value={total === 0 ? 0 : percent / 100}
                size={72}
                strokeWidth={6}
                tone={total === 0 ? "muted" : percent === 100 ? "success" : "warning"}
                aria-label="Share signed"
              >
                <span className="font-mono text-sm font-semibold tabular-nums">{percent}%</span>
              </RingGauge>
              <div className="flex-1">
                <p className="text-lg leading-tight font-semibold tabular-nums">
                  {view.signed} of {total} signed
                </p>
                <p className="text-muted-foreground text-xs">
                  {total === 0
                    ? "Nobody it applies to is on the roster yet."
                    : view.outstanding === 0
                      ? "Everybody it applies to has signed."
                      : `${view.outstanding} still to sign.`}
                </p>
              </div>
            </section>

            <div className="flex flex-wrap items-center justify-between gap-2">
              <SegmentedControl<Scope>
                items={SCOPE_ITEMS}
                value={scope}
                onValueChange={setScope}
                aria-label="Who to show"
              />
              <div className="relative">
                <SearchIcon className="text-muted-foreground pointer-events-none absolute top-1/2 left-2 size-3.5 -translate-y-1/2" />
                <Input
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                  placeholder="Find a name"
                  aria-label="Find a name"
                  className="h-7 w-44 pl-7 text-xs"
                />
              </div>
            </div>

            {rows.length === 0 ? (
              <EmptyState
                className="max-w-none p-8"
                title={
                  needle
                    ? "Nobody by that name"
                    : scope === "outstanding"
                      ? "Nobody outstanding"
                      : scope === "signed"
                        ? "Nobody has signed yet"
                        : "Nobody in scope"
                }
                description={
                  scope === "outstanding" && !needle
                    ? "Everybody this policy applies to has signed the version in force."
                    : "Try another view or a different name."
                }
                icons={[UsersIcon, FileSignatureIcon, CheckIcon]}
              />
            ) : (
              <ul className="flex min-h-0 flex-col gap-1 overflow-y-auto">
                {rows.map((row) => {
                  const [first, last] = splitName(row.workerName);
                  const done = Boolean(row.acknowledgedAt);
                  return (
                    <li
                      key={row.workerId}
                      className={cn(
                        "hover:bg-muted/40 flex items-center gap-3 rounded-lg px-2 py-1.5 text-xs transition-colors",
                        !done && "border-l-2 border-l-amber-500/60",
                      )}
                    >
                      <Avatar className="size-7">
                        <AvatarFallback className="text-[10px] font-medium">
                          {initials(first, last)}
                        </AvatarFallback>
                      </Avatar>
                      <span className="flex min-w-0 flex-1 flex-col">
                        <span className="truncate font-medium">{row.workerName}</span>
                        <span className="text-muted-foreground truncate tabular-nums">
                          {row.workerType}
                          {row.acknowledgedAt
                            ? ` · ${formatShiftDate(row.acknowledgedAt)}${
                                row.signatureName ? ` · signed “${row.signatureName}”` : " · read"
                              }`
                            : ""}
                        </span>
                      </span>
                      {done ? (
                        <Badge variant="active">Signed</Badge>
                      ) : (
                        <Badge variant="warning">Outstanding</Badge>
                      )}
                    </li>
                  );
                })}
              </ul>
            )}
          </div>
        )}
      </SheetContent>
    </Sheet>
  );
}
