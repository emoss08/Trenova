import { ReasonDialog } from "@/components/reason-dialog";
import { lazyWorkPane, preloadOn } from "@/components/work-pane";
import { usePermission } from "@/hooks/use-permission";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  archiveWorkerCredential,
  fetchWorkerCredentialSummary,
  fetchWorkerCredentials,
  verifyWorkerCredential,
  WORKER_CREDENTIAL_SUMMARY_KEY,
  WORKER_CREDENTIALS_KEY,
  type WorkerCredentialRow,
  type WorkerCredentialSummaryItem,
} from "@/lib/graphql/worker-credential";
import { workerOverviewQuery } from "@/lib/queries/worker";
import { previewPendingRefetchMs } from "@/lib/resource-documents";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { EmptyTableSketch } from "@trenova/shared/components/ui/empty-table";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect, useRef, useState } from "react";
import type { BoundedTableHandle } from "../../bounded-table";
import { useRowFlash } from "../../use-row-flash";
import { useWorkerInvalidation } from "../../use-worker-invalidation";
import { WORKER_FORM_PANES } from "../../worker-form-panes";
import { useWorkerFormGate } from "../../use-worker-form-gate";
import { QualificationSkeletonBody } from "../qualification-skeleton";
import { useQualificationRoute, type QualificationViewProps } from "../qualification-route";
import { CredentialHistory } from "./credential-history";
import { credentialGroups, earlierVersions } from "./credential-model";
import { CredentialOverview } from "./credential-overview";
import { CredentialTable, type CredentialPermissions } from "./credential-table";

const VIEW = "credentials" as const;
const ARCHIVE_REASON_MAX = 255;

const CredentialReadPane = lazyWorkPane(
  () => import("./credential-read-pane"),
  (module) => module.CredentialReadPane,
  {
    title: (props, t) => props.credential?.credentialType?.name ?? t("Credential"),
    openKey: (props) => props.credentialId,
    fields: 3,
  },
);

const CredentialFormPane = lazyWorkPane(
  WORKER_FORM_PANES["add-credential"],
  (module) => module.CredentialFormPane,
  {
    title: (props, t) =>
      props.mode === "create"
        ? t("Add credential")
        : props.mode === "renew"
          ? t("Renew credential")
          : t("Edit credential"),
  },
);

/**
 * The credentials on file: the standing and its runway, one table of required and other
 * credentials, and History. A row opens its read pane; add, renew and edit open as forms in the
 * work pane, all from the address (`credential=`, `form=`), so the Overview and notifications
 * can open them too.
 */
export default function WorkerCredentialsTab({ workerId, open, onOpen }: QualificationViewProps) {
  const t = useT();
  const { go, closeWhen } = useQualificationRoute(VIEW, onOpen);
  const { flash, flashOf } = useRowFlash();
  const table = useRef<BoundedTableHandle>(null);

  const { allowed: canCreate } = usePermission(Resource.WorkerCredential, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.WorkerCredential, Operation.Update);
  const { allowed: canVerify } = usePermission(Resource.WorkerCredential, Operation.Approve);
  const { allowed: canArchive } = usePermission(Resource.WorkerCredential, Operation.Archive);
  const permissions: CredentialPermissions = { canCreate, canUpdate, canVerify, canArchive };
  const invalidate = useWorkerInvalidation(workerId, "credentials");

  const summaryQuery = useQuery({
    queryKey: [WORKER_CREDENTIAL_SUMMARY_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerCredentialSummary(workerId, { signal }),
  });
  const credentialsQuery = useQuery({
    queryKey: [WORKER_CREDENTIALS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerCredentials({ workerId }, { signal }),
    refetchInterval: (query) =>
      previewPendingRefetchMs(
        query,
        (query.state.data ?? []).map((row) => row.document),
      ),
  });
  const overviewQuery = useQuery(workerOverviewQuery(workerId));

  const verify = useApiMutation<WorkerCredentialRow, WorkerCredentialRow>({
    scope: { id: `worker-credential-verify:${workerId}` },
    resourceName: "Credential",
    mutationFn: (credential) => verifyWorkerCredential(credential.id, credential.version),
    onSuccess: (saved) => {
      flash(saved.id);
      void invalidate();
    },
  });
  const verifyCredential = async (credential: Pick<WorkerCredentialRow, "id" | "version">) => {
    if (verify.isPending) return;
    await verify.mutateAsync(credential as WorkerCredentialRow).catch(() => undefined);
  };

  const [archiving, setArchiving] = useState<WorkerCredentialRow | null>(null);
  const archive = useApiMutation<
    WorkerCredentialRow,
    { credential: WorkerCredentialRow; reason: string }
  >({
    scope: { id: `worker-credential-archive:${workerId}` },
    resourceName: "Credential",
    mutationFn: ({ credential, reason }) =>
      archiveWorkerCredential({ id: credential.id, version: credential.version, reason }),
    onSuccess: (_saved, { credential }) => {
      setArchiving(null);
      if (open.credential === credential.id) onOpen({ tab: "qualification", view: VIEW });
      void invalidate();
    },
  });

  const items = summaryQuery.data?.items ?? [];
  const credentials = credentialsQuery.data ?? [];
  const byId = new Map(credentials.map((row) => [row.id, row]));
  // A current credential is always in the summary, whatever the capped list holds.
  const credentialOf = (id: string | null | undefined): WorkerCredentialRow | null => {
    if (!id) return null;
    const held = items.find((item) => item.credential?.id === id)?.credential;
    return byId.get(id) ?? (held ? (held as WorkerCredentialRow) : null);
  };
  const rowKeyOf = (credentialId: string) =>
    items.find((item) => item.credential?.id === credentialId)?.credentialType.id ?? null;
  const readOpen = open.credential != null && open.form == null;
  const selected = credentialOf(open.credential);
  const formCredential =
    open.form === "renew-credential" || open.form === "edit-credential" ? selected : null;

  const lastShown = useRef<string | null>(null);
  useEffect(() => {
    const id = open.credential;
    const key = id ? rowKeyOf(id) : null;
    if (!id || !key || id === lastShown.current) return;
    lastShown.current = id;
    flash(id);
    table.current?.scrollToKey(key);
  });

  const loaded = summaryQuery.isSuccess && credentialsQuery.isSuccess;
  const opens = useWorkerFormGate(open.form, {
    "add-credential": canCreate,
    "renew-credential": loaded ? canCreate && formCredential !== null : "waiting",
    "edit-credential": loaded ? canUpdate && formCredential !== null : "waiting",
  });

  if (summaryQuery.isPending || credentialsQuery.isPending) {
    return <QualificationSkeletonBody view={VIEW} />;
  }
  if (summaryQuery.isError || credentialsQuery.isError) {
    return (
      <ErrorState
        layout="sheet"
        error={summaryQuery.error ?? credentialsQuery.error}
        title={t("The credentials could not be loaded")}
        onRetry={() => {
          void summaryQuery.refetch();
          void credentialsQuery.refetch();
        }}
        sketch={
          <EmptyTableSketch columns={[{ label: t("Credential") }, { label: t("Expires") }]} />
        }
      />
    );
  }

  const summary = summaryQuery.data;
  const { required, other } = credentialGroups(summary);
  const archived = credentials.filter((row) => row.status === "Archived");
  const itemCredential = (item: WorkerCredentialSummaryItem) => credentialOf(item.credential?.id);
  const target = (credentialId: string, form?: "renew-credential" | "edit-credential") => ({
    tab: "qualification" as const,
    view: VIEW,
    credential: credentialId,
    ...(form ? { form } : {}),
  });
  const addTarget = (typeId?: string) => ({
    tab: "qualification" as const,
    view: VIEW,
    form: "add-credential" as const,
    ...(typeId ? { kind: typeId } : {}),
  });

  return (
    <div className="flex flex-col gap-4">
      <CredentialOverview
        summary={summary}
        clearance={overviewQuery.data?.clearance}
        asOf={overviewQuery.data?.asOf ?? null}
        canCreate={canCreate}
        addIntent={preloadOn(CredentialFormPane)}
        onAdd={() => go(addTarget())}
        onOpen={go}
      />

      <section
        aria-label={t("Credentials on file")}
        className="overflow-hidden rounded-lg border bg-card"
      >
        <CredentialTable
          required={required}
          other={other}
          permissions={permissions}
          verifyingId={verify.isPending ? (verify.variables?.id ?? null) : null}
          flashOf={flashOf}
          handleRef={table}
          rowIntent={preloadOn(CredentialReadPane)}
          menuIntent={preloadOn(CredentialFormPane)}
          handlers={{
            onOpen: (item) => item.credential && go(target(item.credential.id)),
            onAdd: (item) => go(addTarget(item.credentialType.id)),
            onVerify: async (item) => {
              const credential = itemCredential(item);
              if (credential) await verifyCredential(credential);
            },
            onRenew: (item) =>
              item.credential && go(target(item.credential.id, "renew-credential")),
            onEdit: (item) => item.credential && go(target(item.credential.id, "edit-credential")),
            onArchive: (item) => setArchiving(itemCredential(item)),
          }}
        />
        <CredentialHistory
          archived={archived}
          count={summary.archivedCount}
          read={credentials.length}
        />
      </section>

      <CredentialReadPane.Component
        open={readOpen}
        onOpenChange={closeWhen(readOpen)}
        credentialId={readOpen ? (open.credential ?? null) : null}
        credential={readOpen ? selected : null}
        workerId={workerId}
        loadedEarlier={selected ? earlierVersions(credentials, selected) : []}
        permissions={permissions}
        verifying={verify.isPending}
        onVerify={verifyCredential}
        editIntent={preloadOn(CredentialFormPane)}
        onEdit={(credential) => go(target(credential.id, "edit-credential"))}
        onRenew={(credential) => go(target(credential.id, "renew-credential"))}
        onArchive={setArchiving}
      />
      <CredentialFormPane.Component
        open={opens("add-credential")}
        onOpenChange={closeWhen(open.form === "add-credential")}
        workerId={workerId}
        mode="create"
        credentialTypeId={open.form === "add-credential" ? open.kind : null}
      />
      <CredentialFormPane.Component
        open={opens("renew-credential")}
        onOpenChange={closeWhen(open.form === "renew-credential")}
        workerId={workerId}
        mode="renew"
        credential={open.form === "renew-credential" ? (formCredential ?? undefined) : undefined}
        credentialTypeId={
          open.form === "renew-credential" ? formCredential?.credentialTypeId : null
        }
        onSaved={(saved) => flash(saved.id)}
      />
      <CredentialFormPane.Component
        open={opens("edit-credential")}
        onOpenChange={closeWhen(open.form === "edit-credential")}
        workerId={workerId}
        mode="edit"
        credential={open.form === "edit-credential" ? (formCredential ?? undefined) : undefined}
        onSaved={(saved) => flash(saved.id)}
      />
      <ReasonDialog
        open={archiving !== null}
        title={t("Archive this credential?")}
        description={
          archiving?.credentialType?.name
            ? t("{0} leaves the file and moves to History.", archiving.credentialType.name)
            : t("It leaves the file and moves to History.")
        }
        placeholder={t("e.g. Endorsement surrendered")}
        confirmLabel={t("Archive")}
        cancelLabel={t("Cancel")}
        destructive
        maxLength={ARCHIVE_REASON_MAX}
        pending={archive.isPending}
        onConfirm={(reason) => {
          if (archiving && !archive.isPending) {
            void archive.mutateAsync({ credential: archiving, reason }).catch(() => undefined);
          }
        }}
        onOpenChange={(next) => !next && !archive.isPending && setArchiving(null)}
      />
    </div>
  );
}
