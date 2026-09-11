import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import { KpiStat } from "@/components/kpi/kpi-stat";
import { usePermission } from "@/hooks/use-permission";
import {
  fetchWorkerPolicies,
  WORKER_POLICIES_KEY,
  type WorkerPolicyRow,
} from "@/lib/graphql/self-service";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { formatShiftDate } from "@trenova/shared/lib/scheduling";
import { policyAudienceLabel } from "@trenova/shared/lib/self-service";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  ArchiveIcon,
  FileSignatureIcon,
  FileTextIcon,
  PenLineIcon,
  PlusIcon,
  ScrollTextIcon,
  UsersIcon,
} from "lucide-react";
import { useState } from "react";
import { PolicyComplianceDialog } from "./policy-compliance-dialog";
import { PoliciesEmpty } from "./policies-empty";
import { PolicyCardsSkeleton } from "./policies-skeleton";
import { PolicyDialog } from "./policy-dialog";

type Scope = "active" | "all";

const SCOPE_ITEMS = [
  { value: "active", label: "In force" },
  { value: "all", label: "Everything" },
] satisfies { value: Scope; label: string }[];

export default function PoliciesConsole() {
  const t = useT();

  const { allowed: canRead } = usePermission(Resource.WorkerPolicy, Operation.Read);
  const { allowed: canCreate } = usePermission(Resource.WorkerPolicy, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.WorkerPolicy, Operation.Update);
  const [scope, setScope] = useState<Scope>("active");
  const [dialog, setDialog] = useState<{ policy: WorkerPolicyRow | null } | null>(null);
  const [compliance, setCompliance] = useState<WorkerPolicyRow | null>(null);

  const policies = useQuery({
    queryKey: [WORKER_POLICIES_KEY],
    queryFn: ({ signal }) => fetchWorkerPolicies(undefined, { signal }),
    enabled: canRead,
  });

  if (!canRead) return null;

  const rows = policies.data ?? [];
  const active = rows.filter((row) => row.status === "Active");
  const shown = scope === "active" ? active : rows;

  return (
    <div className="flex flex-col gap-4">
      <div className="grid grid-cols-6 gap-3">
        <KpiStat
          label={t("In force")}
          value={String(active.length)}
          icon={<ScrollTextIcon className="size-[11px]" />}
          sub={t("Policies drivers are bound by today")}
        />
        <KpiStat
          label={t("Need a signature")}
          value={String(active.filter((row) => row.requiresSignature).length)}
          tone="warning"
          icon={<FileSignatureIcon className="size-[11px]" />}
          sub={t("The rest only need reading")}
        />
        <KpiStat
          label={t("Retired")}
          value={String(rows.length - active.length)}
          tone="muted"
          icon={<ArchiveIcon className="size-[11px]" />}
          sub={t("Kept so old signatures still point at something")}
        />
      </div>

      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <SegmentedControl<Scope>
            items={SCOPE_ITEMS}
            value={scope}
            onValueChange={setScope}
            aria-label={t("Which policies to show")}
          />
          <InfoPopover title={t("Signatures")}>
            {t("A signature is pinned to the version label it was given for. Correcting the text under the same label keeps every signature; publishing a new label asks everybody it applies to to read and sign again from Dash.")}
          </InfoPopover>
        </div>
        {canCreate ? (
          <Button size="sm" onClick={() => setDialog({ policy: null })}>
            <PlusIcon className="size-3.5" />
            {t("Publish a policy")}
          </Button>
        ) : null}
      </div>

      {policies.isLoading ? (
        <PolicyCardsSkeleton />
      ) : shown.length === 0 ? (
        <PoliciesEmpty
          title={scope === "active" ? "Nothing in force" : "No policies yet"}
          description={t("Publish a handbook or a policy and everybody it applies to is asked to read and sign it from Dash.")}
          onPublish={canCreate ? () => setDialog({ policy: null }) : undefined}
        />
      ) : (
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {shown.map((policy) => (
            <PolicyCard
              key={policy.id}
              policy={policy}
              onCompliance={() => setCompliance(policy)}
              onEdit={canUpdate ? () => setDialog({ policy }) : undefined}
            />
          ))}
        </div>
      )}

      <PolicyDialog
        open={dialog !== null}
        onOpenChange={(open) => !open && setDialog(null)}
        policy={dialog?.policy ?? null}
      />
      <PolicyComplianceDialog
        policy={compliance}
        onOpenChange={(open) => !open && setCompliance(null)}
      />
    </div>
  );
}

function PolicyCard({
  policy,
  onCompliance,
  onEdit,
}: {
  policy: WorkerPolicyRow;
  onCompliance: () => void;
  onEdit?: () => void;
}) {
  const t = useT();

  const retired = policy.status !== "Active";

  return (
    <div
      className={cn(
        "border-border/80 hover:border-border group flex flex-col gap-3 rounded-lg border p-3 transition-colors",
        retired && "opacity-70",
      )}
    >
      <div className="flex items-start gap-3">
        <span className="bg-accent inline-flex size-7 shrink-0 items-center justify-center rounded-md">
          {policy.documentId ? (
            <FileTextIcon className="size-4" />
          ) : (
            <PenLineIcon className="size-4" />
          )}
        </span>
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium">{policy.title}</p>
          <p className="text-muted-foreground truncate text-xs">
            {policy.summary || `${policy.code} · from ${formatShiftDate(policy.effectiveFrom)}`}
          </p>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-1.5">
        <Badge variant="outline">{t("v{0}", policy.versionLabel)}</Badge>
        <Badge variant="secondary">{policyAudienceLabel(policy.appliesTo)}</Badge>
        <Badge variant={policy.requiresSignature ? "warning" : "secondary"}>
          {policy.requiresSignature ? "Signature" : "Read only"}
        </Badge>
        {retired ? <Badge variant="inactive">{t("Retired")}</Badge> : null}
      </div>

      <div className="mt-auto flex items-center justify-between gap-2 border-t pt-3">
        <Button size="xs" variant="outline" onClick={onCompliance}>
          <UsersIcon className="size-3.5" />
          {t("Who has signed")}
        </Button>
        {onEdit ? (
          <Button
            size="xs"
            variant="ghost"
            className="opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100"
            onClick={onEdit}
            aria-label={`Edit ${policy.title}`}
          >
            {t("Edit")}
          </Button>
        ) : null}
      </div>
    </div>
  );
}
