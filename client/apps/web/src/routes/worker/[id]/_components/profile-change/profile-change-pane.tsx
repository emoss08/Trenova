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
import { decideProfileChange, type ProfileChangeRequestRow } from "@/lib/graphql/self-service";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { ArrowRightIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Form } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { changeRequestTone, isPhotoChange } from "@trenova/shared/lib/self-service";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  decideProfileChangeFormSchema,
  type DecideProfileChangeFormValues,
} from "@trenova/shared/types/self-service";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { PaneSection } from "../pane-section";
import { useWorkerInvalidation } from "../use-worker-invalidation";
import { PhotoComparison } from "./photo-comparison";
import { workerProfileChangesQuery } from "./profile-change-query";

type FieldChange = ProfileChangeRequestRow["changes"][number];

function shown(value: string, blank: string): string {
  return value.trim() === "" ? blank : value;
}

/** Each field as before → after, so the eye lands on what actually moves. */
function ChangedFields({ changes }: { changes: readonly FieldChange[] }) {
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

export type ProfileChangePaneProps = {
  workerId: string;
  workerFirstName: string;
  /** The request the page's `request` parameter names; the pane is open while it is set. */
  requestId: string | null;
  onClose: () => void;
};

/**
 * Mounted by `lazyWorkPane` (the page's `ProfileChange`), fresh for each request it opens on,
 * so the decision form starts from its defaults without a reset effect.
 *
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
  const invalidate = useWorkerInvalidation(workerId, "profile-change");
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
  const photoRequest = request ? isPhotoChange(request.changes) : false;

  const form = useForm<DecideProfileChangeFormValues>({
    resolver: zodResolver(decideProfileChangeFormSchema) as Resolver<DecideProfileChangeFormValues>,
    defaultValues: { approve: true, note: null },
  });
  const { control, handleSubmit, setValue } = form;
  const approving = useWatch({ control, name: "approve" });

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
    onSuccess: async () => {
      await invalidate();
      onClose();
    },
  });

  const submit = (approve: boolean) => {
    if (decide.isPending) return;
    setValue("approve", approve);
    void handleSubmit((values) => decide.mutateAsync(values))();
  };

  const description = request
    ? photoRequest
      ? t(
          "Sent {0} from Dash. Approving makes it their photo.",
          formatUnixDateMedium(request.submittedAt),
        )
      : t(
          "Asked {0} from Dash. Approving writes the listed fields to the profile.",
          formatUnixDateMedium(request.submittedAt),
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
              <ErrorState
                error={requests.error}
                layout="compact"
                title={t("The request could not be loaded")}
                onRetry={() => void requests.refetch()}
              />
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
                        {formatUnixDateMedium(request.decidedAt)}
                      </span>
                    ) : null}
                  </div>
                ) : null}
                {photoRequest ? (
                  <PaneSection title={t("Their new photo")}>
                    {request.proposedPhoto ? (
                      <PhotoComparison
                        firstName={request.worker?.firstName ?? workerFirstName}
                        lastName={request.worker?.lastName ?? ""}
                        current={request.worker?.photo}
                        proposed={request.proposedPhoto}
                      />
                    ) : (
                      <p className="text-sm text-muted-foreground">
                        {t(
                          "Once a photo request is decided, only the photo on the record is kept.",
                        )}
                      </p>
                    )}
                  </PaneSection>
                ) : (
                  <PaneSection title={t("What they changed")}>
                    <ChangedFields changes={request.changes} />
                  </PaneSection>
                )}
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
