import { usePermission } from "@/hooks/use-permission";
import {
  APPROVAL_DELEGATIONS_KEY,
  fetchApprovalDelegations,
  revokeApprovalDelegation,
} from "@/lib/graphql/org-structure";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDate, getTodayDate } from "@trenova/shared/lib/date";
import {
  approvalScopeLabel,
  DELEGATION_STATE_LABELS,
  delegationState,
  delegationStateTone,
} from "@trenova/shared/lib/org-structure";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { PlusIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { DelegationDialog } from "./delegation-dialog";

/**
 * Who is approving in whose place. Unfiltered the list is what has been handed
 * to the signed-in user; the "handed out" tab is what they have given away.
 */
export function DelegationPanel() {
  const queryClient = useQueryClient();
  const { allowed: canRead } = usePermission(Resource.ApprovalDelegation, Operation.Read);
  const { allowed: canDelegate } = usePermission(Resource.ApprovalDelegation, Operation.Create);
  const userId = useAuthStore((state) => state.user?.id);
  const [view, setView] = useState<"received" | "given">("given");
  const [dialogOpen, setDialogOpen] = useState(false);

  const delegationsQuery = useQuery({
    queryKey: [APPROVAL_DELEGATIONS_KEY, view, userId],
    // Naming one side and leaving the other open is what keeps the two views
    // apart. The server refuses a pair that names neither side as the signed-in
    // user, so neither view can read somebody else's arrangements by accident.
    queryFn: ({ signal }) =>
      view === "given"
        ? fetchApprovalDelegations({ delegatorId: userId }, { signal })
        : fetchApprovalDelegations({ delegateId: userId }, { signal }),
    enabled: canRead && Boolean(userId),
  });

  const revokeMutation = useMutation({
    mutationFn: (id: string) => revokeApprovalDelegation(id),
    onSuccess: () => {
      toast.success("Delegation called back", {
        description: "It is kept on the list so approvals made under it can still be explained.",
      });
      void queryClient.invalidateQueries({ queryKey: [APPROVAL_DELEGATIONS_KEY] });
    },
    onError: (error: Error) =>
      toast.error("Could not call it back", { description: error.message }),
  });

  if (!canRead) return null;

  const delegations = delegationsQuery.data ?? [];
  const today = getTodayDate();

  return (
    <section className="rounded-lg border p-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h3 className="text-sm font-medium">Approval cover</h3>
          <p className="text-muted-foreground text-xs">
            Handing approvals to somebody else while a manager is away. Cover widens what the
            stand-in can act on; it never widens what the manager could approve themselves.
          </p>
        </div>
        <div className="flex items-center gap-1.5">
          <Button
            size="xs"
            variant={view === "given" ? "default" : "outline"}
            onClick={() => setView("given")}
          >
            Handed out
          </Button>
          <Button
            size="xs"
            variant={view === "received" ? "default" : "outline"}
            onClick={() => setView("received")}
          >
            Covering for
          </Button>
          {canDelegate ? (
            <Button size="sm" onClick={() => setDialogOpen(true)}>
              <PlusIcon className="size-3.5" />
              Arrange cover
            </Button>
          ) : null}
        </div>
      </div>

      {delegationsQuery.isLoading ? (
        <Skeleton className="mt-3 h-16 w-full" />
      ) : delegations.length === 0 ? (
        <p className="text-muted-foreground mt-3 rounded-md border border-dashed p-3 text-xs">
          {view === "given"
            ? "You have not handed your approvals to anybody."
            : "Nobody has handed you their approvals."}
        </p>
      ) : (
        <ul className="mt-3 flex flex-col gap-1.5">
          {delegations.map((delegation) => {
            const state = delegationState(delegation, today);
            return (
              <li
                key={delegation.id}
                className="flex flex-wrap items-center justify-between gap-2 border-t pt-1.5 text-xs first:border-t-0 first:pt-0"
              >
                <span className="flex min-w-0 flex-wrap items-center gap-2">
                  <span className="font-medium">
                    {delegation.delegator?.name ?? "Someone"} →{" "}
                    {delegation.delegate?.name ?? "someone"}
                  </span>
                  <Badge variant="secondary">{approvalScopeLabel(delegation.scope)}</Badge>
                  <Badge variant={delegationStateTone(state)}>
                    {DELEGATION_STATE_LABELS[state]}
                  </Badge>
                  <span className="text-muted-foreground">
                    {formatUnixDate(delegation.startsAt)}
                    {delegation.endsAt
                      ? ` – ${formatUnixDate(delegation.endsAt)}`
                      : " – until called back"}
                  </span>
                  {delegation.reason ? (
                    <span className="text-muted-foreground truncate">{delegation.reason}</span>
                  ) : null}
                </span>
                {state !== "revoked" && state !== "ended" ? (
                  <Button
                    size="xs"
                    variant="ghost"
                    isLoading={revokeMutation.isPending}
                    onClick={() => revokeMutation.mutate(delegation.id)}
                    aria-label={`Call back the delegation to ${delegation.delegate?.name ?? "them"}`}
                  >
                    Call back
                  </Button>
                ) : null}
              </li>
            );
          })}
        </ul>
      )}

      <DelegationDialog open={dialogOpen} onOpenChange={setDialogOpen} />
    </section>
  );
}
