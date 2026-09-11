import { useT } from "@trenova/shared/i18n/use-t";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  decideProfileChange,
  fetchProfileChangeRequests,
  PROFILE_CHANGE_REQUESTS_KEY,
  type ProfileChangeRequestRow,
} from "@/lib/graphql/self-service";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatShiftDate } from "@trenova/shared/lib/scheduling";
import { changeRequestTone } from "@trenova/shared/lib/self-service";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  decideProfileChangeFormSchema,
  type DecideProfileChangeFormValues,
} from "@trenova/shared/types/self-service";
import { ArrowRightIcon, CheckIcon, ClipboardPenIcon, XIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";

type FieldChange = ProfileChangeRequestRow["changes"][number];

/**
 * What the driver has asked to change on their own record. The changes are
 * before-and-after pairs, so a manager sees exactly what is asked and an
 * approval applies exactly that.
 */
export function ProfileChangeRequests({ workerId }: { workerId: string }) {
  const t = useT();

  const { allowed: canRead } = usePermission(Resource.ProfileChangeRequest, Operation.Read);
  const { allowed: canApprove } = usePermission(Resource.ProfileChangeRequest, Operation.Approve);
  const { allowed: canReject } = usePermission(Resource.ProfileChangeRequest, Operation.Reject);
  const [deciding, setDeciding] = useState<{
    request: ProfileChangeRequestRow;
    approve: boolean;
  } | null>(null);

  const requests = useQuery({
    queryKey: [PROFILE_CHANGE_REQUESTS_KEY, workerId],
    queryFn: ({ signal }) => fetchProfileChangeRequests({ workerId, limit: 20 }, { signal }),
    enabled: canRead && workerId.length > 0,
  });

  if (!canRead) return null;
  if (requests.isLoading) return <Skeleton className="h-24 w-full rounded-lg" />;

  const rows = requests.data ?? [];
  if (rows.length === 0) return null;

  const pendingCount = rows.filter((row) => row.status === "Pending").length;

  return (
    <div className="rounded-lg border p-4">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <ClipboardPenIcon className="text-muted-foreground size-4" />
          <h3 className="text-sm font-semibold">{t("Profile changes")}</h3>
        </div>
        {pendingCount > 0 ? <Badge variant="warning">{t("{0} waiting on you", pendingCount)}</Badge> : null}
      </div>
      <p className="text-muted-foreground mt-1 text-xs">
        {t("Changes the driver asked for from Dash. An approval writes exactly what is listed onto the record — nothing else.")}
      </p>

      <ul className="mt-3 flex flex-col gap-2">
        {rows.map((request) => {
          const tone = changeRequestTone(request.status);
          const pending = request.status === "Pending";
          return (
            <li key={request.id} className="rounded-lg border p-3 text-xs">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="flex items-center gap-2">
                  <span className="text-muted-foreground tabular-nums">
                    {t("Asked {0}", formatShiftDate(request.submittedAt))}
                  </span>
                  <Badge variant={tone.variant}>{tone.label}</Badge>
                </span>
                {pending ? (
                  <span className="flex items-center gap-1.5">
                    {canReject ? (
                      <Button
                        size="xs"
                        variant="outline"
                        onClick={() => setDeciding({ request, approve: false })}
                      >
                        <XIcon className="size-3" />
                        {t("Turn down")}
                      </Button>
                    ) : null}
                    {canApprove ? (
                      <Button size="xs" onClick={() => setDeciding({ request, approve: true })}>
                        <CheckIcon className="size-3" />
                        {t("Approve")}
                      </Button>
                    ) : null}
                  </span>
                ) : null}
              </div>
              <ChangeList changes={request.changes} className="mt-2" />
              {request.note ? <p className="text-muted-foreground mt-2">“{request.note}”</p> : null}
              {request.decisionNote ? (
                <p className="text-muted-foreground mt-1">{t("Office: {0}", request.decisionNote)}</p>
              ) : null}
            </li>
          );
        })}
      </ul>

      <DecideDialog
        state={deciding}
        workerId={workerId}
        onOpenChange={(open) => !open && setDeciding(null)}
      />
    </div>
  );
}

function shown(value: string): string {
  return value.trim() === "" ? "(blank)" : value;
}

/** Each field as before → after, so the eye lands on what actually moves. */
function ChangeList({
  changes,
  className,
}: {
  changes: readonly FieldChange[];
  className?: string;
}) {
  return (
    <dl className={cn("grid grid-cols-[auto_1fr] gap-x-3 gap-y-1", className)}>
      {changes.map((change) => (
        <div key={change.field} className="contents">
          <dt className="text-muted-foreground">{change.label}</dt>
          <dd className="flex min-w-0 flex-wrap items-center gap-1.5 tabular-nums">
            <span className="text-muted-foreground line-through decoration-muted-foreground/50">
              {shown(change.from)}
            </span>
            <ArrowRightIcon className="text-muted-foreground size-3 shrink-0" />
            <span className="font-medium">{shown(change.to)}</span>
          </dd>
        </div>
      ))}
    </dl>
  );
}

function DecideDialog({
  state,
  workerId,
  onOpenChange,
}: {
  state: { request: ProfileChangeRequestRow; approve: boolean } | null;
  workerId: string;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();

  const queryClient = useQueryClient();
  const form = useForm<DecideProfileChangeFormValues>({
    resolver: zodResolver(decideProfileChangeFormSchema) as Resolver<DecideProfileChangeFormValues>,
    defaultValues: { approve: true, note: null },
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (state) reset({ approve: state.approve, note: null });
  }, [state, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    { id: string },
    DecideProfileChangeFormValues,
    unknown,
    DecideProfileChangeFormValues
  >({
    form,
    resourceName: "Request",
    mutationFn: (values) =>
      decideProfileChange({
        id: state?.request.id ?? "",
        approve: values.approve,
        note: values.note ?? undefined,
      }),
    onSuccess: (_result, values) => {
      toast.success(values.approve ? "Change applied to the record" : "Request turned down");
      void queryClient.invalidateQueries({ queryKey: [PROFILE_CHANGE_REQUESTS_KEY, workerId] });
      void queryClient.invalidateQueries({ queryKey: ["worker"] });
      onOpenChange(false);
    },
  });

  const approve = state?.approve ?? true;

  return (
    <Dialog open={state !== null} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{approve ? "Apply this change" : "Turn this request down"}</DialogTitle>
          <DialogDescription>
            {approve
              ? "The listed fields are written onto the record and the driver is told."
              : "The record stays as it is. The driver is told, with your reason."}
          </DialogDescription>
        </DialogHeader>
        {state ? (
          <ChangeList
            changes={state.request.changes}
            className="bg-muted/30 rounded-lg border p-3 text-xs"
          />
        ) : null}
        <FormProvider {...form}>
          <Form
            onSubmit={(submitEvent) => {
              submitEvent.preventDefault();
              submitEvent.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(submitEvent);
            }}
          >
            <FormGroup className="pb-2" cols={1}>
              <FormControl cols="full">
                <TextareaField<DecideProfileChangeFormValues>
                  control={control}
                  name="note"
                  label={approve ? "Note (optional)" : "Why"}
                  placeholder={
                    approve
                      ? "Anything the driver should know"
                      : "e.g. The address needs a unit number"
                  }
                  description={t("Sent to the driver with the decision and kept on the request.")}
                  rules={{ required: !approve }}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button
                type="submit"
                variant={approve ? "default" : "destructive"}
                isLoading={isPending}
              >
                {approve ? "Apply" : "Turn down"}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
