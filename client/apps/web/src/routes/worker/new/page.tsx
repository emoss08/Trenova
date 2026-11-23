import { PageLayout } from "@/components/navigation/sidebar-layout";
import { UnsavedChangesGuard } from "@/components/unsaved-changes-guard";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { workerService } from "@/services/worker";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { Link } from "@trenova/shared/components/link";
import { Button } from "@trenova/shared/components/ui/button";
import { Form } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { workerFormSchema, type Worker } from "@trenova/shared/types/worker";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { NEW_WORKER_VALUES } from "../[id]/_components/profile/worker-form-values";
import {
  NewWorkerNote,
  WorkerProfileCards,
} from "../[id]/_components/profile/worker-profile-cards";
import { WORKERS_PATH, workerPath } from "../use-worker-state";

const FORM_ID = "new-worker-form";

/**
 * A new worker: the Profile's cards, empty, with the whole record required, since whoever creates
 * a worker records their personal details. Creating one starts onboarding, assigns required
 * training and applies the default time off policy, and then opens the worker's own page.
 */
export function WorkerCreatePage() {
  const t = useT();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { isPopout, closePopout } = usePopoutWindow();
  const form = useForm<Worker>({
    resolver: zodResolver(workerFormSchema(true)) as Resolver<Worker>,
    defaultValues: NEW_WORKER_VALUES,
  });
  const {
    handleSubmit,
    reset,
    formState: { isDirty },
  } = form;

  const create = useApiMutation<Worker, Worker, unknown, Worker>({
    form,
    resourceName: "Worker",
    scope: { id: "worker-create" },
    mutationFn: (values) => workerService.create(values),
    onSuccess: (created, values) => {
      reset(values);
      void queryClient.invalidateQueries({ queryKey: ["worker-list"] });
      if (isPopout) {
        closePopout();
        return;
      }
      if (created.id) void navigate({ to: workerPath(created.id), replace: true });
    },
  });

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("New worker"),
        description: t(
          "Everything a dispatcher and the safety file need from day one. The rest of the record fills in as you go.",
        ),
      }}
    >
      <FormProvider {...form}>
        <Form id={FORM_ID} onSubmit={handleSubmit((values) => create.mutateAsync(values))}>
          <WorkerProfileCards mode="create" personalWithheld={false} note={<NewWorkerNote />} />
          <div className="mt-4 flex justify-end gap-2 border-t border-border-subtle pt-4">
            {isPopout ? (
              <Button type="button" variant="outline" size="sm" onClick={() => closePopout()}>
                {t("Cancel")}
              </Button>
            ) : (
              <Button
                variant="outline"
                size="sm"
                nativeButton={false}
                render={<Link to={WORKERS_PATH} />}
              >
                {t("Cancel")}
              </Button>
            )}
            <Button type="submit" size="sm" form={FORM_ID} isLoading={create.isPending}>
              {t("Create worker")}
            </Button>
          </div>
        </Form>
      </FormProvider>
      <UnsavedChangesGuard when={isDirty && !create.isPending && !create.isSuccess} />
    </PageLayout>
  );
}
