import { HoverGlideGroup } from "@/components/hover-glide";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { VirtualRows, type VirtualRowsHandle } from "@/components/virtual-rows";
import { lazyWorkPane, preloadOn } from "@/components/work-pane";
import { usePermission } from "@/hooks/use-permission";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  fetchWorkerCredentialSummary,
  WORKER_CREDENTIAL_SUMMARY_KEY,
} from "@/lib/graphql/worker-credential";
import {
  deleteEmploymentVerification,
  DRIVER_QUALIFICATION_FILE_KEY,
  fetchDriverQualificationFile,
  markEmploymentVerificationRequested,
  recordEmploymentVerificationFollowUp,
  type DQFFile,
  type DQFItem,
  type EmploymentVerificationRow,
} from "@/lib/graphql/worker-dqf";
import type { WorkerTarget } from "@/routes/worker/use-worker-state";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { PlusIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { EmptyTableSketch } from "@trenova/shared/components/ui/empty-table";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { dqfItemStatusLabel, dqfItemTone, type DQFSectionValue } from "@trenova/shared/lib/dqf";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect, useRef, useState } from "react";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { QualificationSkeletonBody } from "../qualification-skeleton";
import { useQualificationRoute, type QualificationViewProps } from "../qualification-route";
import { ATTENTION_CLASS, revealElement, useRowFlash } from "../../use-row-flash";
import { useWorkerInvalidation } from "../../use-worker-invalidation";
import { WORKER_FORM_PANES } from "../../worker-form-panes";
import {
  dqNextSteps,
  dqSpine,
  itemTone,
  type DQStep,
  type FileCredential,
  type StepAbilities,
  type StepAction,
} from "./dq-file-model";
import { DQNextSteps } from "./dq-next-steps";
import { DQSpine, spineName } from "./dq-spine";
import { EmployerRow } from "./employer-row";

const VIEW = "dq-file" as const;

const EmployerFormPane = lazyWorkPane(
  WORKER_FORM_PANES["add-employer"],
  (module) => module.EmployerFormPane,
  {
    title: (props, t) =>
      props.verification ? t("Edit previous employer") : t("Add previous employer"),
  },
);

/**
 * The driver qualification file as a process: the verdict, the spine of its four sections, one
 * panel per section with the evidence, the previous employers and their requests, the next
 * steps in order and how long the file is kept. Employers are added and edited in the work pane.
 */
export default function WorkerDQFTab({ workerId, open, onOpen }: QualificationViewProps) {
  const t = useT();
  const { go, closeWhen } = useQualificationRoute(VIEW, onOpen);
  const { flash, flashOf } = useRowFlash();
  const sections = useRef(new Map<string, HTMLElement>());
  const employers = useRef<VirtualRowsHandle>(null);

  const { allowed: canCreate } = usePermission(Resource.Qualification, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.Qualification, Operation.Update);
  const { allowed: canDelete } = usePermission(Resource.Qualification, Operation.Delete);
  const { allowed: canReadCredentials } = usePermission(Resource.WorkerCredential, Operation.Read);
  const { allowed: canCreateCredential } = usePermission(
    Resource.WorkerCredential,
    Operation.Create,
  );
  const invalidate = useWorkerInvalidation(workerId, "dq-file");

  const fileQuery = useQuery({
    queryKey: [DRIVER_QUALIFICATION_FILE_KEY, workerId],
    queryFn: ({ signal }) => fetchDriverQualificationFile(workerId, { signal }),
  });
  const credentialQuery = useQuery({
    queryKey: [WORKER_CREDENTIAL_SUMMARY_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerCredentialSummary(workerId, { signal }),
    enabled: canReadCredentials,
  });

  const requested = useApiMutation<unknown, string>({
    scope: { id: `employment-verification:${workerId}` },
    resourceName: "Previous employer",
    mutationFn: (id) => markEmploymentVerificationRequested(id),
    onSuccess: (_data, id) => {
      flash(id);
      void invalidate();
    },
  });
  const followUp = useApiMutation<unknown, string>({
    scope: { id: `employment-verification:${workerId}` },
    resourceName: "Previous employer",
    mutationFn: (id) => recordEmploymentVerificationFollowUp(id),
    onSuccess: (_data, id) => {
      flash(id);
      void invalidate();
    },
  });
  const [removing, setRemoving] = useState<EmploymentVerificationRow | null>(null);
  const remove = useApiMutation<boolean, string>({
    scope: { id: `employment-verification:${workerId}` },
    resourceName: "Previous employer",
    mutationFn: (id) => deleteEmploymentVerification(id),
    onSuccess: () => {
      setRemoving(null);
      void invalidate();
    },
  });
  const busy = requested.isPending || followUp.isPending || remove.isPending;
  const run = (mutation: typeof requested, id: string) => {
    if (busy) return;
    void mutation.mutateAsync(id).catch(() => undefined);
  };

  const file = fileQuery.data;
  const lastShown = useRef<string | null>(null);
  useEffect(() => {
    const id = open.form == null ? open.employer : null;
    if (!id || id === lastShown.current || !file?.verifications.some((row) => row.id === id)) {
      return;
    }
    lastShown.current = id;
    flash(id);
    employers.current?.scrollToKey(id);
  });

  if (fileQuery.isPending) return <QualificationSkeletonBody view={VIEW} />;
  if (fileQuery.isError || !file) {
    return (
      <ErrorState
        layout="sheet"
        error={fileQuery.error ?? new Error("No qualification file")}
        title={t("The qualification file could not be loaded")}
        onRetry={() => void fileQuery.refetch()}
        sketch={<EmptyTableSketch columns={[{ label: t("Item") }, { label: t("Status") }]} />}
      />
    );
  }

  const fileCredentials = new Map<string, FileCredential>();
  for (const item of credentialQuery.data?.items ?? []) {
    fileCredentials.set(item.credentialType.code, {
      typeId: item.credentialType.id,
      credentialId: item.credential?.id ?? null,
      name: item.credentialType.name,
      required: item.required,
      verified: Boolean(item.credential?.verifiedAt),
    });
  }
  const can: StepAbilities = {
    createCredential: canCreateCredential,
    createEmployer: canCreate,
    updateEmployer: canUpdate,
  };
  const spine = dqSpine(file);
  const deadline = { dueAt: file.safetyHistoryDueAt, late: file.safetyHistoryLate };
  const editing =
    open.form === "edit-employer" && open.employer
      ? (file.verifications.find((row) => row.id === open.employer) ?? null)
      : null;
  const busyStep = requested.isPending
    ? `verification:${requested.variables}`
    : followUp.isPending
      ? `verification:${followUp.variables}`
      : null;

  const selectSection = (section: DQFSectionValue) => {
    revealElement(sections.current.get(section));
    flash(`section:${section}`);
  };
  const sectionRef = (section: string) => (element: HTMLElement | null) => {
    if (element) sections.current.set(section, element);
    else sections.current.delete(section);
  };
  const onStep = (_step: DQStep, action: StepAction) => {
    if (action.kind === "go") go(action.target);
    else run(action.kind === "request" ? requested : followUp, action.verificationId);
  };
  const credentialTarget = (item: DQFItem): WorkerTarget => {
    const credentialId = fileCredentials.get(item.code)?.credentialId;
    return credentialId
      ? { tab: "qualification", view: "credentials", credential: credentialId }
      : { tab: "qualification", view: "credentials" };
  };

  return (
    <div className="flex flex-col gap-4">
      <div
        data-testid="dqf-header"
        className="flex min-h-7 flex-wrap items-center gap-x-2.5 gap-y-2"
      >
        <h2 className="text-base font-semibold">{t("Driver qualification file")}</h2>
        <Badge variant={file.complete ? "success" : "danger"} dot>
          {file.complete ? t("Complete") : t("Incomplete")}
        </Badge>
        <span className="font-mono text-xs text-muted-foreground">{t("49 CFR 391.51")}</span>
        {canCreate ? (
          <Button
            size="sm"
            variant="secondary"
            className="ml-auto"
            {...preloadOn(EmployerFormPane)}
            onClick={() => go({ tab: "qualification", view: VIEW, form: "add-employer" })}
          >
            <PlusIcon className="size-3.5" />
            {t("Add previous employer")}
          </Button>
        ) : null}
      </div>

      <DQSpine blocks={spine} onSelect={selectSection} />

      <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_24rem]">
        <div className="flex min-w-0 flex-col gap-4">
          <ItemSection
            section="Credentials"
            file={file}
            sectionRef={sectionRef("Credentials")}
            flash={flashOf("section:Credentials")}
            link={{
              label: t("Credentials"),
              target: { tab: "qualification", view: "credentials" },
            }}
            itemTarget={credentialTarget}
            onGo={go}
          />
          <ItemSection
            section="Documents"
            file={file}
            sectionRef={sectionRef("Documents")}
            flash={flashOf("section:Documents")}
            link={{ label: t("Documents"), target: { tab: "files", view: "documents" } }}
            itemTarget={() => ({ tab: "files", view: "documents" })}
            onGo={go}
          />

          <div
            ref={sectionRef("SafetyHistory")}
            data-changed={flashOf("section:SafetyHistory")}
            className={cn("scroll-mt-4 rounded-lg", ATTENTION_CLASS)}
          >
            <SectionPanel
              title={t("Previous employers")}
              count={file.verifications.length}
              hint={t("Three-year lookback · 49 CFR 391.23")}
            >
              {file.verifications.length === 0 ? (
                <SectionPanelQuiet>
                  {t(
                    "No previous employer has been recorded. Until one is, the three-year investigation has not been made.",
                  )}
                </SectionPanelQuiet>
              ) : (
                <VirtualRows
                  aria-label={t("Previous employers")}
                  handleRef={employers}
                  rows={file.verifications.map((verification) => ({
                    key: verification.id,
                    render: () => (
                      <EmployerRow
                        verification={verification}
                        permissions={{ canUpdate, canDelete }}
                        deadline={deadline}
                        busy={busy}
                        flash={flashOf(verification.id)}
                        intent={preloadOn(EmployerFormPane)}
                        onRequest={(row) => run(requested, row.id)}
                        onFollowUp={(row) => run(followUp, row.id)}
                        onEdit={(row) =>
                          go({
                            tab: "qualification",
                            view: VIEW,
                            form: "edit-employer",
                            employer: row.id,
                          })
                        }
                        onRemove={setRemoving}
                      />
                    ),
                  }))}
                  mask="card"
                  className="max-h-120"
                  contentClassName="px-1.5 pt-1 pb-1.5"
                  hoverGlide
                  estimateSize={112}
                />
              )}
            </SectionPanel>
          </div>

          <ItemSection
            section="DrugAlcohol"
            file={file}
            sectionRef={sectionRef("DrugAlcohol")}
            flash={flashOf("section:DrugAlcohol")}
            link={{
              label: t("Drug and alcohol"),
              target: { tab: "qualification", view: "drug-and-alcohol" },
            }}
            itemTarget={() => ({ tab: "qualification", view: "drug-and-alcohol" })}
            onGo={go}
          />
        </div>

        <div className="flex min-w-0 flex-col gap-4">
          <FileNextSteps
            file={file}
            credentials={fileCredentials}
            can={can}
            busyId={busyStep}
            onAction={onStep}
          />
          <RetentionPanel file={file} />
        </div>
      </div>

      <EmployerFormPane.Component
        open={open.form === "add-employer" && canCreate}
        onOpenChange={closeWhen(open.form === "add-employer")}
        workerId={workerId}
        verification={null}
        onSaved={flash}
      />
      <EmployerFormPane.Component
        open={open.form === "edit-employer" && canUpdate && editing != null}
        onOpenChange={closeWhen(open.form === "edit-employer")}
        workerId={workerId}
        verification={editing}
        onSaved={flash}
      />
      <ConfirmDialog
        open={removing !== null}
        title={t("Remove this employer?")}
        description={
          removing
            ? t("{0} and its request history leave the file.", removing.employerName)
            : t("The employer and its request history leave the file.")
        }
        confirmLabel={t("Remove employer")}
        pending={remove.isPending}
        onConfirm={() => {
          if (removing && !remove.isPending) {
            void remove.mutateAsync(removing.id).catch(() => undefined);
          }
        }}
        onOpenChange={(next) => !next && setRemoving(null)}
      />
    </div>
  );
}

/**
 * The next steps, worked out against the clock: a request falls due to be chased as the days
 * pass, so this leaf reads the minute and the view around it does not.
 */
function FileNextSteps({
  file,
  credentials,
  can,
  busyId,
  onAction,
}: {
  file: DQFFile;
  credentials: ReadonlyMap<string, FileCredential>;
  can: StepAbilities;
  busyId: string | null;
  onAction: (step: DQStep, action: StepAction) => void;
}) {
  const t = useT();
  const now = useNow("minute");
  const steps = dqNextSteps(t, { file, credentials, now, can });
  return <DQNextSteps steps={steps} busyId={busyId} onAction={onAction} />;
}

/** One section of the file as a panel of its items, with a link named for where it is fixed. */
function ItemSection({
  section,
  file,
  sectionRef,
  flash,
  link,
  itemTarget,
  onGo,
}: {
  section: DQFSectionValue;
  file: DQFFile;
  sectionRef: (element: HTMLElement | null) => void;
  flash: "a" | "b" | undefined;
  link: { label: string; target: WorkerTarget };
  itemTarget: (item: DQFItem) => WorkerTarget;
  onGo: (target: WorkerTarget) => void;
}) {
  const t = useT();
  const items = file.items.filter((item) => item.section === section);
  const title = spineName(t, section);

  return (
    <div
      ref={sectionRef}
      data-changed={flash}
      className={cn("scroll-mt-4 rounded-lg", ATTENTION_CLASS)}
    >
      <SectionPanel
        title={title}
        action={
          <button
            type="button"
            onClick={() => onGo(link.target)}
            className="rounded-sm ui-focus-ring text-xs text-muted-foreground transition-colors hover:text-foreground hover:underline"
          >
            {link.label}
          </button>
        }
      >
        {items.length === 0 ? (
          <SectionPanelQuiet>
            {t("Nothing in this section applies to this worker.")}
          </SectionPanelQuiet>
        ) : (
          <HoverGlideGroup className="bg-card px-1.5 pt-1 pb-1.5">
            <ul className="flex flex-col gap-px">
              {items.map((item) => (
                <li key={`${item.section}:${item.code}`}>
                  <button
                    type="button"
                    data-glide-target
                    data-state={itemTone(item.status) ?? "none"}
                    onClick={() => onGo(itemTarget(item))}
                    className="relative grid w-full grid-cols-[minmax(0,1fr)_auto] items-center gap-3 rounded-md ui-focus-ring p-2 text-left sm:grid-cols-[minmax(0,1fr)_7rem_7.5rem]"
                  >
                    <span className="min-w-0">
                      <span className="block truncate text-sm font-medium">{item.name}</span>
                      <span className="block truncate text-xs text-muted-foreground">
                        {[item.detail, item.regulation].filter(Boolean).join(" · ")}
                      </span>
                    </span>
                    <span
                      className={cn(
                        "hidden text-right text-xs whitespace-nowrap sm:block",
                        item.expiresAt ? "font-mono" : "text-muted-foreground",
                        item.status === "ExpiringSoon" && "text-warning-foreground",
                        item.status === "Expired" && "text-danger-foreground",
                      )}
                    >
                      {item.expiresAt ? formatUnixDateMedium(item.expiresAt) : t("No expiry")}
                    </span>
                    <Badge variant={dqfItemTone(item.status)} dot className="justify-self-start">
                      {dqfItemStatusLabel(item.status)}
                    </Badge>
                  </button>
                </li>
              ))}
            </ul>
          </HoverGlideGroup>
        )}
      </SectionPanel>
    </div>
  );
}

/** How long the file is held: while the worker is employed and the retention after they leave. */
function RetentionPanel({ file }: { file: DQFFile }) {
  const t = useT();
  const sentence = file.purgeEligible
    ? t(
        "Held past its retention since {0}. Review it for purge; that is never automatic.",
        formatUnixDateMedium(file.retentionExpiresAt),
      )
    : file.retentionExpiresAt
      ? t("Hold this file until {0}.", formatUnixDateMedium(file.retentionExpiresAt))
      : t("Hold this file while they work here and for at least three years after they leave.");
  return (
    <SectionPanel title={t("Retention")} hint={t("391.51(d)")}>
      <p className="px-3 pt-3 pb-2.5 text-sm">{sentence}</p>
      <DescriptionList layout="split" className="border-t border-border-subtle px-3">
        <DescriptionItem label={t("Hired")} numeric>
          {file.hireDate ? formatUnixDateMedium(file.hireDate) : t("Not recorded")}
        </DescriptionItem>
        <DescriptionItem label={t("Can be destroyed")} numeric>
          {file.retentionExpiresAt ? (
            formatUnixDateMedium(file.retentionExpiresAt)
          ) : (
            <span className="text-muted-foreground">
              {t("At least three years after they leave")}
            </span>
          )}
        </DescriptionItem>
      </DescriptionList>
    </SectionPanel>
  );
}
