import { useWorkPane } from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { useProtectedFieldAccess } from "@/hooks/use-protected-fields";
import { workerOverviewQuery } from "@/lib/queries/worker";
import {
  workerFormTarget,
  type WorkerRecord,
  type WorkerTabProps,
} from "@/routes/worker/use-worker-state";
import { useWorkerAreas } from "@/routes/worker/worker-areas";
import { workerService } from "@/services/worker";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import { Form } from "@trenova/shared/components/ui/form";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { workerFormSchema, type Worker } from "@trenova/shared/types/worker";
import { useEffect, useRef } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { ProfileChangeAlert } from "./profile-change-alert";
import { ProfileDockReporter, profileFormId, type ProfileDock } from "./profile-dock";
import { workerFormValues, workerSaveValues } from "./worker-form-values";
import { WorkerProfileCards } from "./worker-profile-cards";
import { useWorkerInvalidation } from "../use-worker-invalidation";

export type ProfileTabProps = WorkerTabProps & {
  /**
   * Hands the page the form for its save dock. The page draws the dock beside the work pane host,
   * so it stays in view on any tab, and tells unsaved edits from the form itself, since this tab's
   * effects stop while it is hidden. Not called for a reader who may not update the worker.
   */
  onDock: (dock: ProfileDock) => void;
};

function versionOf(record: WorkerRecord): string {
  return `${record.id}:${record.version ?? ""}:${record.profile?.version ?? ""}`;
}

/**
 * The worker's own record as one form: the person, their licence and clearance. Saving goes
 * through the dock with the full record, as the server replaces it whole. A change another tab
 * writes while this form holds edits (an employment event moves the fleet code) comes in under
 * the edits instead of over them.
 */
export default function ProfileTab({ workerId, worker, onOpen, shown, onDock }: ProfileTabProps) {
  const invalidate = useWorkerInvalidation(workerId, "profile");
  const { closeThen } = useWorkPane();
  const areas = useWorkerAreas();
  const personalVisible = useProtectedFieldAccess("Personal");
  const { allowed: canUpdate } = usePermission(Resource.Worker, Operation.Update);
  const { allowed: canRecordEvent } = usePermission(
    Resource.WorkerEmploymentEvent,
    Operation.Create,
  );
  const overview = useQuery(workerOverviewQuery(workerId));

  const form = useForm<Worker>({
    resolver: zodResolver(workerFormSchema(personalVisible)) as Resolver<Worker>,
    defaultValues: workerFormValues(worker),
  });
  const { handleSubmit, reset } = form;

  // The form follows the stored record only when it moves to a new version, so neither a refetch
  // of the same version nor the tab being shown again (Activity re-runs effects) touches it. A new
  // version keeps every field the person has changed and takes the rest from the record.
  const loadedVersion = useRef(versionOf(worker));
  useEffect(() => {
    const version = versionOf(worker);
    if (loadedVersion.current === version) return;
    loadedVersion.current = version;
    reset(workerFormValues(worker), { keepDirtyValues: true });
  }, [worker, reset]);

  const save = useApiMutation<Worker, Worker, unknown, Worker>({
    form,
    resourceName: "Worker",
    scope: { id: `worker-profile-save:${workerId}` },
    mutationFn: (values) => workerService.update(workerId, workerSaveValues(values)),
    onSuccess: (_saved, values) => {
      reset(values);
      void invalidate();
    },
  });

  // A save started from the dock on another tab must not fail out of sight: a refusal (a field
  // the form or the server turned down) brings the person here, where its alert and fields are.
  const showRefusal = () => {
    if (!shown) closeThen(() => onOpen({ tab: "profile" }));
  };
  const submit = handleSubmit((values) => save.mutateAsync(values).catch(showRefusal), showRefusal);
  const formId = profileFormId(workerId);

  const mayRecordEvent = canRecordEvent && areas.isDrawable({ tab: "employment", view: "history" });

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <ProfileChangeAlert
        workerId={workerId}
        firstName={worker.firstName}
        waiting={overview.data?.portal?.pendingProfileChangeCount ?? null}
        onReview={(request) => closeThen(() => onOpen({ tab: "profile", request }))}
      />
      <FormProvider {...form}>
        <Form
          id={formId}
          onSubmit={(event) => void submit(event)}
          className="flex min-w-0 flex-col"
        >
          {canUpdate ? <ProfileDockReporter formId={formId} onDock={onDock} /> : null}
          <fieldset disabled={!canUpdate} className="contents">
            <WorkerProfileCards
              mode="edit"
              personalWithheld={!personalVisible}
              stored={{
                status: worker.status,
                terminationDate: worker.profile?.terminationDate ?? null,
                complianceStatus: worker.profile?.complianceStatus ?? null,
                licenseStateId: worker.profile?.licenseStateId ?? null,
                licenseStateAbbreviation: worker.profile?.licenseState?.abbreviation ?? null,
              }}
              onRecordEvent={
                mayRecordEvent
                  ? () => closeThen(() => onOpen(workerFormTarget("record-event")))
                  : undefined
              }
            />
          </fieldset>
        </Form>
      </FormProvider>
    </div>
  );
}
