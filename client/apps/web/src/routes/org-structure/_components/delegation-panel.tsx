import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import {
  APPROVAL_DELEGATIONS_KEY,
  revokeApprovalDelegation,
  type ApprovalDelegationRow,
} from "@/lib/graphql/org-structure";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  SegmentedControl,
  type SegmentedControlItem,
} from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDate, getTodayDate } from "@trenova/shared/lib/date";
import {
  approvalScopeLabel,
  DELEGATION_STATE_LABELS,
  delegationState,
  delegationStateTone,
  type DelegationState,
} from "@trenova/shared/lib/org-structure";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { ArrowRightIcon, HandshakeIcon, PlusIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { DelegationDialog } from "./delegation-dialog";
import { delegationsGivenQuery, delegationsReceivedQuery } from "./queries";

type View = "given" | "received";

const STATE_ORDER: Record<DelegationState, number> = {
  active: 0,
  scheduled: 1,
  ended: 2,
  revoked: 3,
};

function inForce(source: readonly ApprovalDelegationRow[], today: number): number {
  return source.filter((delegation) => delegationState(delegation, today) === "active").length;
}

/**
 * Who is approving in whose place. "Handed out" is what the signed-in user
 * has given away; "covering for" is what has been handed to them. In force
 * comes first in either list, because that is what somebody opens this for.
 */
export function DelegationPanel() {
  const t = useT();

  const queryClient = useQueryClient();
  const { allowed: canRead } = usePermission(Resource.ApprovalDelegation, Operation.Read);
  const { allowed: canDelegate } = usePermission(Resource.ApprovalDelegation, Operation.Create);
  const userId = useAuthStore((state) => state.user?.id);
  const [view, setView] = useState<View>("given");
  const [dialogOpen, setDialogOpen] = useState(false);
  const today = getTodayDate();

  const enabled = canRead && Boolean(userId);
  const given = useQuery({ ...delegationsGivenQuery(userId ?? ""), enabled });
  const received = useQuery({ ...delegationsReceivedQuery(userId ?? ""), enabled });

  const revokeMutation = useMutation({
    mutationFn: (id: string) => revokeApprovalDelegation(id),
    onSuccess: () => {
      toast.success(t("Delegation called back"), {
        description: t("It is kept on the list so approvals made under it can still be explained."),
      });
      void queryClient.invalidateQueries({ queryKey: [APPROVAL_DELEGATIONS_KEY] });
    },
    onError: (error: Error) =>
      toast.error(t("Could not call it back"), { description: error.message }),
  });

  const rows = useMemo(() => {
    const source = view === "given" ? given.data : received.data;
    return (source ?? [])
      .map((delegation) => ({ delegation, state: delegationState(delegation, today) }))
      .sort(
        (a, b) =>
          STATE_ORDER[a.state] - STATE_ORDER[b.state] ||
          b.delegation.startsAt - a.delegation.startsAt,
      );
  }, [view, given.data, received.data, today]);

  const viewItems = useMemo<SegmentedControlItem<View>[]>(
    () => [
      {
        value: "given",
        label: "Handed out",
        caption: given.data ? `${inForce(given.data, today)} in force` : undefined,
      },
      {
        value: "received",
        label: "Covering for",
        caption: received.data ? `${inForce(received.data, today)} in force` : undefined,
      },
    ],
    [given.data, received.data, today],
  );

  if (!canRead) return null;

  const loading = view === "given" ? given.isLoading : received.isLoading;

  return (
    <section
      aria-labelledby="delegation-heading"
      className="bg-card overflow-hidden rounded-lg border"
    >
      <header className="flex flex-wrap items-center justify-between gap-2 border-b px-3 py-2">
        <div className="flex min-w-0 items-center gap-2">
          <HandshakeIcon className="text-muted-foreground size-3.5" aria-hidden />
          <h3 id="delegation-heading" className="text-sm font-medium">
            {t("Approval cover")}
          </h3>
          <span className="text-muted-foreground hidden truncate text-xs md:inline">
            {t("Cover widens what the stand-in can act on; it never widens what the manager could approve themselves.")}
          </span>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <SegmentedControl<View>
            items={viewItems}
            value={view}
            onValueChange={setView}
            aria-label={t("Which cover to show")}
          />
          {canDelegate ? (
            <Button size="sm" onClick={() => setDialogOpen(true)}>
              <PlusIcon className="size-3.5" />
              {t("Arrange cover")}
            </Button>
          ) : null}
        </div>
      </header>

      {loading ? (
        <div className="flex flex-col gap-2 p-3">
          <Skeleton className="h-9 w-full" />
          <Skeleton className="h-9 w-2/3" />
        </div>
      ) : rows.length === 0 ? (
        <p className="text-muted-foreground px-3 py-3 text-sm">
          {view === "given"
            ? t("You have not handed your approvals to anybody.")
            : t("Nobody has handed you their approvals.")}
        </p>
      ) : (
        <ul className="divide-y" aria-label={view === "given" ? "Handed out" : "Covering for"}>
          {rows.map(({ delegation, state }) => (
            <li
              key={delegation.id}
              className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2"
            >
              <div className="flex min-w-0 flex-col gap-0.5 leading-tight">
                <span className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
                  <span className="flex items-center gap-1.5 text-sm font-medium">
                    <span className="truncate">{delegation.delegator?.name ?? t("Someone")}</span>
                    <ArrowRightIcon
                      className="text-muted-foreground size-3.5 shrink-0"
                      aria-hidden
                    />
                    <span className="truncate">{delegation.delegate?.name ?? "someone"}</span>
                  </span>
                  <Badge variant="secondary">{approvalScopeLabel(delegation.scope)}</Badge>
                  <Badge variant={delegationStateTone(state)}>
                    {DELEGATION_STATE_LABELS[state]}
                  </Badge>
                </span>
                <span className="text-muted-foreground text-xs tabular-nums">
                  {formatUnixDate(delegation.startsAt)}
                  {delegation.endsAt
                    ? ` – ${formatUnixDate(delegation.endsAt)}`
                    : ` ${t("– until called back")}`}
                  {delegation.reason ? ` · ${delegation.reason}` : ""}
                </span>
              </div>
              {state === "active" || state === "scheduled" ? (
                <Button
                  size="xs"
                  variant="ghost"
                  isLoading={revokeMutation.isPending && revokeMutation.variables === delegation.id}
                  disabled={revokeMutation.isPending}
                  onClick={() => revokeMutation.mutate(delegation.id)}
                  aria-label={`Call back the delegation to ${delegation.delegate?.name ?? "them"}`}
                >
                  {t("Call back")}
                </Button>
              ) : null}
            </li>
          ))}
        </ul>
      )}

      <DelegationDialog open={dialogOpen} onOpenChange={setDialogOpen} />
    </section>
  );
}
