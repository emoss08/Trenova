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
import { changeRequestTone, describeChanges } from "@trenova/shared/lib/self-service";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  decideProfileChangeFormSchema,
  type DecideProfileChangeFormValues,
} from "@trenova/shared/types/self-service";
import { ClipboardPenIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";

/**
 * What the driver has asked to change on their own record. The changes are
 * before-and-after pairs, so a manager sees exactly what is asked and an
 * approval applies exactly that.
 */
export function ProfileChangeRequests({ workerId }: { workerId: string }) {
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

  return (
    <div className="border-border rounded-lg border p-4">
      <div className="flex items-center gap-2">
        <ClipboardPenIcon className="text-muted-foreground size-4" />
        <p className="text-sm font-semibold">Profile changes</p>
      </div>
      <p className="text-muted-foreground mt-1 text-xs">
        Changes the driver asked for from Dash. An approval writes exactly what is listed onto the
        record — nothing else.
      </p>

      <ul className="mt-3 flex flex-col gap-2">
        {rows.map((request) => {
          const tone = changeRequestTone(request.status);
          const pending = request.status === "Pending";
          return (
            <li key={request.id} className="rounded-md border p-3 text-xs">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="flex items-center gap-2">
                  <span className="text-muted-foreground tabular-nums">
                    Asked {formatShiftDate(request.submittedAt)}
                  </span>
                  <Badge className={cn("border", tone.badge)}>{tone.label}</Badge>
                </span>
                {pending ? (
                  <span className="flex items-center gap-1.5">
                    {canReject ? (
                      <Button
                        size="xs"
                        variant="outline"
                        onClick={() => setDeciding({ request, approve: false })}
                      >
                        Turn down
                      </Button>
                    ) : null}
                    {canApprove ? (
                      <Button size="xs" onClick={() => setDeciding({ request, approve: true })}>
                        Approve
                      </Button>
                    ) : null}
                  </span>
                ) : null}
              </div>
              <ul className="mt-2 flex flex-col gap-0.5">
                {describeChanges(request.changes).map((line) => (
                  <li key={line} className="tabular-nums">
                    {line}
                  </li>
                ))}
              </ul>
              {request.note ? <p className="text-muted-foreground mt-1">“{request.note}”</p> : null}
              {request.decisionNote ? (
                <p className="text-muted-foreground mt-1">Office: {request.decisionNote}</p>
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

function DecideDialog({
  state,
  workerId,
  onOpenChange,
}: {
  state: { request: ProfileChangeRequestRow; approve: boolean } | null;
  workerId: string;
  onOpenChange: (open: boolean) => void;
}) {
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
          <ul className="flex flex-col gap-0.5 rounded-md border p-3 text-xs tabular-nums">
            {describeChanges(state.request.changes).map((line) => (
              <li key={line}>{line}</li>
            ))}
          </ul>
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
                  rules={{ required: !approve }}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
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
