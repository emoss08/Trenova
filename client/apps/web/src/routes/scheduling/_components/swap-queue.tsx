import { EmptyState } from "@/components/empty-state";
import { usePermission } from "@/hooks/use-permission";
import {
  fetchShiftSwapRequests,
  ROTA_KEY,
  SHIFT_SWAPS_KEY,
  transitionShiftSwap,
  type ShiftSwapRow,
} from "@/lib/graphql/scheduling";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Avatar, AvatarFallback } from "@trenova/shared/components/ui/avatar";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatShiftDate, isSwapOpen, SWAP_STATUS_TONES } from "@trenova/shared/lib/scheduling";
import { initials } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { ArrowRightIcon, CheckIcon, RepeatIcon, UsersIcon, XIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

type Scope = "open" | "all";

const SCOPE_ITEMS = [
  { value: "open", label: "Waiting" },
  { value: "all", label: "Everything" },
] satisfies { value: Scope; label: string }[];

/**
 * The office half of a swap. Two drivers agree it between themselves and the
 * office decides it: a swap nobody in the office saw is a shift nobody is
 * covering, which is why Accepted is a waiting room rather than a done deal.
 */
export function SwapQueue() {
  const queryClient = useQueryClient();
  const { allowed: canApprove } = usePermission(Resource.ShiftSwap, Operation.Approve);
  const { allowed: canReject } = usePermission(Resource.ShiftSwap, Operation.Reject);
  const [scope, setScope] = useState<Scope>("open");

  const swapsQuery = useQuery({
    queryKey: [SHIFT_SWAPS_KEY, scope],
    queryFn: ({ signal }) => fetchShiftSwapRequests({ openOnly: scope === "open" }, { signal }),
  });

  const { mutate: decide, isPending } = useMutation({
    mutationFn: (input: { id: string; status: "Approved" | "Rejected" }) =>
      transitionShiftSwap(input),
    onSuccess: (_data, input) => {
      toast.success(input.status === "Approved" ? "Swap approved" : "Swap rejected");
      void queryClient.invalidateQueries({ queryKey: [SHIFT_SWAPS_KEY] });
      void queryClient.invalidateQueries({ queryKey: [ROTA_KEY] });
    },
    onError: (error: Error) => toast.error(error.message),
  });

  const swaps = swapsQuery.data ?? [];
  const decidable = swaps.filter((swap) => swap.status === "Accepted").length;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-muted-foreground max-w-2xl text-xs">
          A swap needs two acceptances: the colleague&apos;s and the office&apos;s. Only a swap the
          colleague has accepted is yours to decide.
          {decidable > 0 ? (
            <span className="text-foreground font-medium"> {decidable} waiting on you.</span>
          ) : null}
        </p>
        <SegmentedControl<Scope>
          items={SCOPE_ITEMS}
          value={scope}
          onValueChange={setScope}
          aria-label="Which swaps to show"
        />
      </div>

      {swapsQuery.isLoading ? (
        <div className="flex flex-col gap-2">
          <Skeleton className="h-16 rounded-lg" />
          <Skeleton className="h-16 rounded-lg" />
        </div>
      ) : swaps.length === 0 ? (
        <EmptyState
          className="max-w-none"
          title={scope === "open" ? "Nothing waiting" : "No swaps yet"}
          description={
            scope === "open"
              ? "Swaps drivers agree between themselves land here for the office's say."
              : "When a driver offers a day to a colleague from Dash, it shows up here."
          }
          icons={[RepeatIcon, UsersIcon, CheckIcon]}
        />
      ) : (
        <ul className="flex flex-col gap-2">
          {swaps.map((swap) => (
            <SwapRow
              key={swap.id}
              swap={swap}
              canDecide={{ approve: canApprove, reject: canReject }}
              busy={isPending}
              onDecide={(status) => decide({ id: swap.id, status })}
            />
          ))}
        </ul>
      )}
    </div>
  );
}

function SwapRow({
  swap,
  canDecide,
  busy,
  onDecide,
}: {
  swap: ShiftSwapRow;
  canDecide: { approve: boolean; reject: boolean };
  busy: boolean;
  onDecide: (status: "Approved" | "Rejected") => void;
}) {
  const tone = SWAP_STATUS_TONES[swap.status] ?? SWAP_STATUS_TONES.Withdrawn;
  const decidable = swap.status === "Accepted";

  return (
    <li className="border-border/80 hover:border-border flex flex-wrap items-center justify-between gap-3 rounded-lg border p-3 text-xs transition-colors">
      <div className="flex min-w-0 items-center gap-3">
        <div className="flex items-center gap-1.5">
          <Person person={swap.requestingWorker} />
          <ArrowRightIcon className="text-muted-foreground size-3.5" />
          <Person person={swap.counterpartyWorker} placeholder="?" />
        </div>
        <div className="flex min-w-0 flex-col">
          <span className="flex flex-wrap items-center gap-2">
            <span className="font-medium">{nameOf(swap.requestingWorker) ?? "A driver"}</span>
            <span className="text-muted-foreground">
              → {nameOf(swap.counterpartyWorker) ?? "whoever the office finds"}
            </span>
            <Badge variant={tone.variant}>{tone.label}</Badge>
          </span>
          <span className="text-muted-foreground tabular-nums">
            Giving up {formatShiftDate(swap.shiftDate)}
            {swap.counterpartyShiftDate
              ? ` · taking ${formatShiftDate(swap.counterpartyShiftDate)}`
              : ""}
          </span>
          {swap.reason ? <span className="text-muted-foreground">“{swap.reason}”</span> : null}
          {swap.responseNote ? (
            <span className="text-muted-foreground">Response: {swap.responseNote}</span>
          ) : null}
        </div>
      </div>

      {decidable && (canDecide.approve || canDecide.reject) ? (
        <div className="flex shrink-0 items-center gap-1.5">
          {canDecide.reject ? (
            <Button
              size="sm"
              variant="outline"
              disabled={busy}
              onClick={() => onDecide("Rejected")}
            >
              <XIcon className="size-3.5" />
              Reject
            </Button>
          ) : null}
          {canDecide.approve ? (
            <Button size="sm" disabled={busy} onClick={() => onDecide("Approved")}>
              <CheckIcon className="size-3.5" />
              Approve
            </Button>
          ) : null}
        </div>
      ) : isSwapOpen(swap.status) ? (
        <span className="text-muted-foreground shrink-0">Waiting on the colleague</span>
      ) : null}
    </li>
  );
}

type PersonLike = { firstName: string; lastName: string } | null | undefined;

function Person({ person, placeholder = "" }: { person: PersonLike; placeholder?: string }) {
  return (
    <Avatar className="size-7">
      <AvatarFallback className="text-[10px] font-medium">
        {person ? initials(person.firstName, person.lastName) : placeholder}
      </AvatarFallback>
    </Avatar>
  );
}

function nameOf(person: PersonLike) {
  if (!person) return null;
  return `${person.firstName} ${person.lastName}`;
}
