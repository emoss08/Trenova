import { Button } from "@trenova/shared/components/ui/button";
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
} from "@trenova/shared/components/ui/drawer";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import {
  dateToUnixTimestamp,
  formatRange,
  generateDateOnly,
  inclusiveDays,
} from "@trenova/shared/lib/date";
import {
  cancelMyPto,
  fetchMyPto,
  requestMyPto,
  type PortalPtoBalance,
} from "@trenova/shared/lib/graphql/driver-portal";
import { cn } from "@trenova/shared/lib/utils";
import type { PortalPtoType } from "@trenova/graphql/generated/graphql";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CalendarDaysIcon, PlusIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { PtoStatusBadge, ptoTypeLabels } from "./portal-badges";
import { PtoBalanceStrip, useMyPtoBalances } from "./pto-balance-strip";
import { useDashFeatures } from "./use-dash-features";

const ptoTypes = Object.keys(ptoTypeLabels) as PortalPtoType[];

export function PtoSection() {
  const queryClient = useQueryClient();
  const [requestOpen, setRequestOpen] = useState(false);
  const pto = useQuery({ queryKey: ["dash-pto"], queryFn: ({ signal }) => fetchMyPto({ signal }) });
  const features = useDashFeatures();
  const balances = useMyPtoBalances(features.ptoBalances);

  const cancel = useMutation({
    mutationFn: (id: string) => cancelMyPto(id),
    onSuccess: async () => {
      toast.success("Request cancelled.");
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["dash-pto"] }),
        queryClient.invalidateQueries({ queryKey: ["dash-pto-balances"] }),
      ]);
    },
    onError: (error: Error) => toast.error(error.message || "We couldn't cancel that request."),
  });

  return (
    <div className="rounded-2xl border border-border bg-card p-4">
      <div className="mb-3 flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <CalendarDaysIcon className="size-4 text-muted-foreground" />
          <h2 className="text-sm font-semibold">Time off</h2>
        </div>
        <Button variant="outline" size="sm" className="h-8" onClick={() => setRequestOpen(true)}>
          <PlusIcon className="size-3.5" />
          Request
        </Button>
      </div>

      {features.ptoBalances ? (
        <PtoBalanceStrip balances={balances.data} isPending={balances.isPending} />
      ) : null}

      {pto.isPending ? (
        <Skeleton className="h-16 w-full rounded-xl" />
      ) : pto.data && pto.data.length > 0 ? (
        <ul className="divide-y divide-border">
          {pto.data.map((request) => (
            <li key={request.id} className="py-2.5">
              <div className="flex items-center justify-between gap-2">
                <p className="text-sm font-medium">
                  {ptoTypeLabels[request.type] ?? request.type}
                  <span className="font-normal text-muted-foreground">
                    {" "}
                    · {formatRange(request.startDate, request.endDate)}
                  </span>
                </p>
                <PtoStatusBadge status={request.status} />
              </div>
              {request.reason ? (
                <p className="mt-0.5 line-clamp-2 text-xs text-muted-foreground">
                  {request.reason}
                </p>
              ) : null}
              {request.status === "Requested" ? (
                <Button
                  variant="ghost"
                  size="sm"
                  className="mt-1 h-7 px-2 text-xs text-muted-foreground"
                  disabled={cancel.isPending}
                  onClick={() => cancel.mutate(request.id)}
                >
                  Cancel request
                </Button>
              ) : null}
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-xs text-muted-foreground">
          Need days off? Send a request and your fleet manager will review it.
        </p>
      )}

      <PtoRequestDrawer balances={balances.data} open={requestOpen} onOpenChange={setRequestOpen} />
    </div>
  );
}

type PtoRequestDrawerProps = {
  balances?: PortalPtoBalance[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

function PtoRequestDrawer({ open, onOpenChange, balances }: PtoRequestDrawerProps) {
  const queryClient = useQueryClient();
  const [type, setType] = useState<PortalPtoType>("Personal");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [reason, setReason] = useState("");

  const reset = () => {
    setType("Personal");
    setStartDate("");
    setEndDate("");
    setReason("");
  };

  const submit = useMutation({
    mutationFn: () => {
      const start = generateDateOnly(startDate);
      const end = generateDateOnly(endDate);
      if (!start || !end) {
        throw new Error("Pick both a start and end date.");
      }
      if (end < start) {
        throw new Error("End date can't be before the start date.");
      }
      return requestMyPto({
        type,
        startDate: dateToUnixTimestamp(start),
        endDate: dateToUnixTimestamp(end),
        reason: reason.trim(),
      });
    },
    onSuccess: async () => {
      toast.success("Request sent — you'll get a notification when it's reviewed.");
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["dash-pto"] }),
        queryClient.invalidateQueries({ queryKey: ["dash-pto-balances"] }),
      ]);
      reset();
      onOpenChange(false);
    },
    onError: (error: Error) => toast.error(error.message || "We couldn't send your request."),
  });

  const canSubmit = startDate.length > 0 && endDate.length > 0 && reason.trim().length > 0;

  const balance = balances?.find((entry) => entry.ptoType === type);
  const requestedDays = (() => {
    const start = generateDateOnly(startDate);
    const end = generateDateOnly(endDate);
    if (!start || !end || end < start) return 0;
    return inclusiveDays(dateToUnixTimestamp(start), dateToUnixTimestamp(end));
  })();
  const available = balance ? Number(balance.availableDays) : null;
  const overdrawn =
    balance?.enforced && available !== null && requestedDays > 0 && requestedDays > available;

  return (
    <Drawer
      open={open}
      onOpenChange={(next) => {
        if (!next) reset();
        onOpenChange(next);
      }}
    >
      <DrawerContent>
        <DrawerHeader>
          <DrawerTitle>Request time off</DrawerTitle>
          <DrawerDescription>
            Your fleet manager reviews requests — you&apos;ll get a notification either way.
          </DrawerDescription>
        </DrawerHeader>

        <div className="flex flex-col gap-4 px-4">
          <div className="flex flex-wrap gap-2">
            {ptoTypes.map((value) => (
              <button
                key={value}
                type="button"
                onClick={() => setType(value)}
                className={cn(
                  "rounded-full border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground transition-colors",
                  type === value && "border-primary bg-primary text-primary-foreground",
                )}
              >
                {ptoTypeLabels[value]}
              </button>
            ))}
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="flex flex-col gap-1.5">
              <Label className="text-xs text-muted-foreground">First day</Label>
              <Input
                type="date"
                value={startDate}
                onChange={(event) => setStartDate(event.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label className="text-xs text-muted-foreground">Last day</Label>
              <Input
                type="date"
                value={endDate}
                min={startDate || undefined}
                onChange={(event) => setEndDate(event.target.value)}
              />
            </div>
          </div>
          {requestedDays > 0 ? (
            <p
              className={cn("text-xs text-muted-foreground", overdrawn && "text-destructive")}
              data-testid="pto-request-days"
            >
              {requestedDays} day{requestedDays === 1 ? "" : "s"}
              {available !== null
                ? ` · ${available.toFixed(available % 1 === 0 ? 0 : 2)} available`
                : ""}
              {overdrawn
                ? " — more than you have banked; your manager may still approve it if days accrue by then."
                : ""}
            </p>
          ) : null}
          <Textarea
            value={reason}
            onChange={(event) => setReason(event.target.value)}
            placeholder="What's it for? A quick note helps your manager plan coverage."
            rows={3}
            maxLength={1000}
          />
        </div>

        <DrawerFooter>
          <Button
            className="h-11"
            disabled={!canSubmit || submit.isPending}
            onClick={() => submit.mutate()}
          >
            {submit.isPending ? "Sending..." : "Send request"}
          </Button>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  );
}
