import { TextareaField } from "@/components/fields/textarea-field";
import {
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneDirtyTracker,
  WorkPaneFooter,
} from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  decideProfileChange,
  PROFILE_CHANGE_REQUESTS_KEY,
  type ProfileChangeRequestRow,
} from "@/lib/graphql/self-service";
import { worker as workerQueries } from "@/lib/queries/worker";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowRightIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Form } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatShiftDate } from "@trenova/shared/lib/scheduling";
import { changeRequestTone } from "@trenova/shared/lib/self-service";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  decideProfileChangeFormSchema,
  type DecideProfileChangeFormValues,
} from "@trenova/shared/types/self-service";
import { useEffect, useRef, type ReactNode } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { workerProfileChangesQuery } from "./profile-change-query";

type FieldChange = ProfileChangeRequestRow["changes"][number];

function shown(value: string, blank: string): string {
  return value.trim() === "" ? blank : value;
}

/** Each field as before → after, so the eye lands on what actually moves. */
export function ProfileChangeList({ changes }: { changes: readonly FieldChange[] }) {
  const t = useT();
  const blank = t("(blank)");
  return (
    <dl className="flex flex-col divide-y rounded-md border border-border-subtle">
      {changes.map((change) => (
        <div key={change.field} className="flex flex-col gap-0.5 px-2.5 py-2">
          <dt className="text-xs font-medium text-foreground-subtle">{t(change.label)}</dt>
          <dd className="flex min-w-0 flex-wrap items-center gap-2 text-sm tabular-nums">
            <del className="text-muted-foreground decoration-foreground-subtle">
              {shown(change.from, blank)}
            </del>
            <ArrowRightIcon aria-hidden className="size-3 shrink-0 text-foreground-subtle" />
            <ins className="font-medium no-underline">{shown(change.to, blank)}</ins>
          </dd>
        </div>
      ))}
    </dl>
  );
}

function PaneSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-2">
      <h4 className="text-xs font-medium text-foreground-subtle">{title}</h4>
      {children}
    </section>
  );
}

export type ProfileChangePaneProps = {
  workerId: string;
  workerFirstName: string;
  /** The request the page's `request` parameter names; the pane is open while it is set. */
  requestId: string | null;
  onClose: () => void;
};

/**
 * One profile change a driver asked for from Dash, beside whatever the page shows: each field old
 * → new, the driver's note, an office note, and Turn down or Approve. An approval writes exactly
 * the listed fields onto the record; turning it down needs a reason, which the driver sees.
 */
export function ProfileChangePane({
  workerId,
  workerFirstName,
  requestId,
  onClose,
}: ProfileChangePaneProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canRead } = usePermission(Resource.ProfileChangeRequest, Operation.Read);
  const { allowed: canApprove } = usePermission(Resource.ProfileChangeRequest, Operation.Approve);
  const { allowed: canReject } = usePermission(Resource.ProfileChangeRequest, Operation.Reject);
  const open = requestId !== null;
  const requests = useQuery({
    ...workerProfileChangesQuery(workerId),
    enabled: open && canRead,
  });
  const request = requests.data?.find((row) => row.id === requestId) ?? null;
  const pending = request?.status === "Pending";

  const form = useForm<DecideProfileChangeFormValues>({
    resolver: zodResolver(decideProfileChangeFormSchema) as Resolver<DecideProfileChangeFormValues>,
    defaultValues: { approve: true, note: null },
  });
  const { control, handleSubmit, reset, setValue } = form;
  const approving = useWatch({ control, name: "approve" });
  const loadedFor = useRef(requestId);
  useEffect(() => {
    if (loadedFor.current === requestId) return;
    loadedFor.current = requestId;
    reset({ approve: true, note: null });
  }, [requestId, reset]);

  const decide = useApiMutation<
    { id: string },
    DecideProfileChangeFormValues,
    unknown,
    DecideProfileChangeFormValues
  >({
    form,
    resourceName: "Request",
    scope: { id: `profile-change-decide:${requestId ?? ""}` },
    mutationFn: (values) =>
      decideProfileChange({
        id: requestId ?? "",
        approve: values.approve,
        note: values.note?.trim() ? values.note.trim() : undefined,
      }),
    onSuccess: () => {
      reset({ approve: true, note: null });
      void queryClient.invalidateQueries({ queryKey: [PROFILE_CHANGE_REQUESTS_KEY, workerId] });
      void queryClient.invalidateQueries({ queryKey: workerQueries._def });
      void queryClient.invalidateQueries({ queryKey: ["worker-list"] });
      onClose();
    },
  });

  const submit = (approve: boolean) => {
    if (decide.isPending) return;
    setValue("approve", approve);
    void handleSubmit((values) => decide.mutateAsync(values))();
  };

  const description = request
    ? t(
        "Asked {0} from Dash. Approving writes the listed fields to the profile.",
        formatShiftDate(request.submittedAt),
      )
    : undefined;

  return (
    <WorkPane
      open={open}
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      title={t("Profile change from {0}", workerFirstName)}
      description={description}
      contentKey={requestId ?? undefined}
    >
      <FormProvider {...form}>
        <Form
          onSubmit={(event) => {
            event.preventDefault();
            submit(true);
          }}
        >
          <WorkPaneDirtyTracker form={form} />
          <WorkPaneBody>
            {!canRead ? (
              <p className="text-sm text-muted-foreground">
                {t("Your role does not include profile change requests.")}
              </p>
            ) : requests.isLoading ? (
              <div aria-busy aria-label={t("Loading the request")} className="flex flex-col gap-3">
                <div aria-hidden className="contents">
                  <span className="block h-3 w-28 ui-shimmer rounded-sm" />
                  <span className="block h-24 ui-shimmer rounded-md" />
                  <span className="block h-3 w-20 ui-shimmer rounded-sm" />
                  <span className="block h-10 ui-shimmer rounded-md" />
                </div>
              </div>
            ) : requests.error ? (
              <p className="text-sm text-danger-foreground">
                {t("The request could not be read.")}{" "}
                <button
                  type="button"
                  className="rounded-sm ui-focus-ring font-medium underline"
                  onClick={() => void requests.refetch()}
                >
                  {t("Try again")}
                </button>
              </p>
            ) : !request ? (
              <p className="text-sm text-muted-foreground">
                {t("This request is not among the worker's latest. It may have been withdrawn.")}
              </p>
            ) : (
              <>
                {!pending ? (
                  <div className="flex items-center gap-2">
                    <Badge variant={changeRequestTone(request.status).variant} dot>
                      {t(changeRequestTone(request.status).label)}
                    </Badge>
                    {request.decidedAt ? (
                      <span className="text-xs text-muted-foreground">
                        {formatShiftDate(request.decidedAt)}
                      </span>
                    ) : null}
                  </div>
                ) : null}
                <PaneSection title={t("What they changed")}>
                  <ProfileChangeList changes={request.changes} />
                </PaneSection>
                {request.note ? (
                  <PaneSection title={t("Their note")}>
                    <blockquote className="border-l-2 border-border-strong py-0.5 pl-2.5 text-sm">
                      {request.note}
                    </blockquote>
                  </PaneSection>
                ) : null}
                {pending ? (
                  <TextareaField<DecideProfileChangeFormValues>
                    control={control}
                    name="note"
                    label={t("Office note")}
                    placeholder={t("Sent to the driver with the decision")}
                    description={t("Needed to turn the request down; the driver sees it.")}
                    rows={3}
                  />
                ) : request.decisionNote ? (
                  <PaneSection title={t("Office note")}>
                    <p className="text-sm">{request.decisionNote}</p>
                  </PaneSection>
                ) : null}
              </>
            )}
          </WorkPaneBody>
          <WorkPaneFooter>
            {pending && (canApprove || canReject) ? (
              <>
                {canReject ? (
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    isLoading={decide.isPending && !approving}
                    disabled={decide.isPending}
                    onClick={() => submit(false)}
                  >
                    {t("Turn down")}
                  </Button>
                ) : null}
                {canApprove ? (
                  <Button
                    type="button"
                    size="sm"
                    isLoading={decide.isPending && approving}
                    disabled={decide.isPending}
                    onClick={() => submit(true)}
                  >
                    {t("Approve")}
                  </Button>
                ) : null}
              </>
            ) : (
              <WorkPaneCancel>{t("Close")}</WorkPaneCancel>
            )}
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </WorkPane>
  );
}
