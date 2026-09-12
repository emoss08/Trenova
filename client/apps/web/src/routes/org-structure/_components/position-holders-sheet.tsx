import { useT } from "@trenova/shared/i18n/use-t";
import { UserAutocompleteField, WorkerAutocompleteField } from "@/components/autocomplete-fields";
import { usePermission } from "@/hooks/use-permission";
import {
  assignUserPosition,
  assignWorkerPosition,
  fetchPositionHolders,
  HEADCOUNT_KEY,
  JOB_POSITIONS_KEY,
  POSITION_HOLDERS_KEY,
  type JobPositionRow,
  type PositionHolderRow,
} from "@/lib/graphql/org-structure";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Avatar, AvatarFallback } from "@trenova/shared/components/ui/avatar";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { jobDepartmentLabel } from "@trenova/shared/lib/org-structure";
import { getNameInitials } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { PlusIcon, XIcon } from "lucide-react";
import { useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { Link } from "react-router";
import { toast } from "sonner";

type PickerValues = { holderId: string };

export function positionHoldersQuery(positionId: string) {
  return {
    queryKey: [POSITION_HOLDERS_KEY, positionId] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) => fetchPositionHolders(positionId, { signal }),
  };
}

function holderHref(holder: PositionHolderRow): string {
  return holder.kind === "Worker"
    ? `/hr/workers?entityId=${holder.id}&modType=edit`
    : `/admin/users?entityId=${holder.id}&modType=edit`;
}

type PositionHoldersSheetProps = {
  position: JobPositionRow | null;
  onOpenChange: (open: boolean) => void;
};

/**
 * Who holds a position, and the place to change it. A driving title lists
 * workers and takes one from the worker picker; a front-office title lists
 * users and takes one from the user picker. The chart is where the office
 * thinks about titles, so this is where a title is filled.
 */
export function PositionHoldersSheet({ position, onOpenChange }: PositionHoldersSheetProps) {
  const t = useT();

  const queryClient = useQueryClient();
  const { allowed: canAssign } = usePermission(Resource.JobPosition, Operation.Update);
  const driving = position?.isDrivingPosition ?? true;
  const form = useForm<PickerValues>({ defaultValues: { holderId: "" } });

  useEffect(() => {
    form.reset({ holderId: "" });
  }, [form, position?.id]);

  const holders = useQuery({
    ...positionHoldersQuery(position?.id ?? ""),
    enabled: Boolean(position),
  });

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: [POSITION_HOLDERS_KEY] });
    void queryClient.invalidateQueries({ queryKey: [HEADCOUNT_KEY] });
    void queryClient.invalidateQueries({ queryKey: [JOB_POSITIONS_KEY] });
  };

  const assign = useMutation({
    mutationFn: ({ holderId, positionId }: { holderId: string; positionId: string | null }) =>
      driving
        ? assignWorkerPosition(holderId, positionId)
        : assignUserPosition(holderId, positionId),
    onSuccess: (_data, { positionId }) => {
      toast.success(positionId ? "Put on the position" : "Taken off the position");
      form.reset({ holderId: "" });
      invalidate();
    },
    onError: (error: Error) =>
      toast.error(t("Could not change that"), { description: error.message }),
  });

  const rows = holders.data ?? [];
  const active = rows.filter((row) => row.status === "Active").length;

  return (
    <Sheet open={Boolean(position)} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-lg">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2">
            {position?.title ?? t("Position")}
            {position ? (
              <Badge variant={position.isDrivingPosition ? "info" : "secondary"}>
                {position.isDrivingPosition ? t("Driving") : t("Front office")}
              </Badge>
            ) : null}
          </SheetTitle>
          <SheetDescription>
            {position
              ? `${position.code} · ${jobDepartmentLabel(position.department)} · ${
                  position.isDrivingPosition
                    ? "held by workers on the roster"
                    : "held by people who log in"
                }`
              : t("Loading")}
          </SheetDescription>
        </SheetHeader>

        <div className="flex flex-col gap-4 overflow-y-auto px-4 pb-4">
          {canAssign && position && position.status === "Active" ? (
            <FormProvider {...form}>
              <Form
                onSubmit={(event) => {
                  event.preventDefault();
                  event.stopPropagation();
                  void form.handleSubmit((values) => {
                    if (!values.holderId) return;
                    assign.mutate({ holderId: values.holderId, positionId: position.id });
                  })(event);
                }}
              >
                <FormGroup cols={1}>
                  <FormControl>
                    <div className="flex items-end gap-2">
                      <div className="min-w-0 flex-1">
                        {driving ? (
                          <WorkerAutocompleteField<PickerValues>
                            control={form.control}
                            name="holderId"
                            label={t("Put a worker on it")}
                            placeholder={t("Find a worker")}
                            description={t(
                              "A worker holds one position, so this replaces any title they hold now.",
                            )}
                            clearable
                          />
                        ) : (
                          <UserAutocompleteField<PickerValues>
                            control={form.control}
                            name="holderId"
                            label={t("Put somebody on it")}
                            placeholder={t("Find a user")}
                            description={t(
                              "A person holds one position, so this replaces any title they hold now.",
                            )}
                            clearable
                          />
                        )}
                      </div>
                      <Button type="submit" size="sm" isLoading={assign.isPending}>
                        <PlusIcon className="size-3.5" />
                        {t("Add")}
                      </Button>
                    </div>
                  </FormControl>
                </FormGroup>
              </Form>
            </FormProvider>
          ) : null}

          <section aria-label={t("People in the position")} className="flex flex-col gap-1.5">
            <header className="flex items-center justify-between px-1">
              <h4 className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
                {t("{0} in it", driving ? t("Workers") : t("People"))}
              </h4>
              {holders.data ? (
                <span className="text-muted-foreground text-xs tabular-nums">
                  {t("{0} active · {1} on record", active, rows.length)}
                </span>
              ) : null}
            </header>
            {holders.isLoading ? (
              <div className="flex flex-col gap-2">
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-2/3" />
              </div>
            ) : rows.length === 0 ? (
              <p className="text-muted-foreground rounded-lg border border-dashed p-4 text-sm">
                {t("Nobody holds this title yet.")}
              </p>
            ) : (
              <ul className="bg-card divide-y overflow-hidden rounded-lg border">
                {rows.map((holder) => (
                  <li
                    key={`${holder.kind}:${holder.id}`}
                    className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2"
                  >
                    <Link
                      to={holderHref(holder)}
                      className="flex min-w-0 items-center gap-2.5 hover:underline"
                    >
                      <Avatar size="sm">
                        <AvatarFallback className="text-2xs font-medium">
                          {getNameInitials(holder.name)}
                        </AvatarFallback>
                      </Avatar>
                      <span className="flex min-w-0 flex-col leading-tight">
                        <span className="truncate text-sm font-medium">{holder.name}</span>
                        <span className="text-muted-foreground truncate text-xs">
                          {holder.detail || (holder.kind === "Worker" ? t("No terminal") : "")}
                        </span>
                      </span>
                    </Link>
                    <span className="flex items-center gap-2">
                      {holder.status !== "Active" ? (
                        <Badge variant="inactive">{holder.status}</Badge>
                      ) : null}
                      {canAssign ? (
                        <Button
                          size="icon-xs"
                          variant="ghost"
                          aria-label={`Take ${holder.name} off the position`}
                          disabled={assign.isPending}
                          onClick={() => assign.mutate({ holderId: holder.id, positionId: null })}
                        >
                          <XIcon className="size-3.5" />
                        </Button>
                      ) : null}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </section>
        </div>
      </SheetContent>
    </Sheet>
  );
}
