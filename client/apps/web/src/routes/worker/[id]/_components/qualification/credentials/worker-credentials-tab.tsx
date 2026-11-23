import { ReasonDialog } from "@/components/reason-dialog";
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
import { worker as workerQueries } from "@/lib/queries/worker";
import { previewPendingRefetchMs } from "@/lib/resource-documents";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { EmptyTableSketch } from "@trenova/shared/components/ui/empty-table";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect, useRef, useState } from "react";
import { QualificationSkeletonBody } from "../qualification-skeleton";
import { useQualificationRoute, type QualificationViewProps } from "../qualification-route";
import { revealElement, useRowFlash } from "../use-row-flash";
import { CredentialFormPane } from "./credential-form-pane";
import { CredentialHistory } from "./credential-history";
import { earlierVersions, credentialGroups } from "./credential-model";
import { CredentialOverview } from "./credential-overview";
import { CredentialReadPane } from "./credential-read-pane";
import { CredentialTable, type CredentialPermissions } from "./credential-table";
import { useCredentialInvalidation } from "./use-credential-invalidation";

const VIEW = "credentials" as const;
const ARCHIVE_REASON_MAX = 255;

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
  const rows = useRef(new Map<string, HTMLTableRowElement>());

  const { allowed: canCreate } = usePermission(Resource.WorkerCredential, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.WorkerCredential, Operation.Update);
  const { allowed: canVerify } = usePermission(Resource.WorkerCredential, Operation.Approve);
  const { allowed: canArchive } = usePermission(Resource.WorkerCredential, Operation.Archive);
  const permissions: CredentialPermissions = { canCreate, canUpdate, canVerify, canArchive };
  const invalidate = useCredentialInvalidation(workerId);

  const summaryQuery = useQuery({
    queryKey: [WORKER_CREDENTIAL_SUMMARY_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerCredentialSummary(workerId, { signal }),
  });
  const credentialsQuery = useQuery({
    queryKey: [WORKER_CREDENTIALS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerCredentials(workerId, true, { signal }),
    refetchInterval: (query) =>
      previewPendingRefetchMs((query.state.data ?? []).map((row) => row.document)),
  });
  const overviewQuery = useQuery(workerQueries.overview(workerId));

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

  const credentials = credentialsQuery.data ?? [];
  const byId = new Map(credentials.map((row) => [row.id, row]));
  const readOpen = open.credential != null && open.form == null;
  const selected = open.credential ? (byId.get(open.credential) ?? null) : null;
  const formCredential =
    open.form === "renew-credential" || open.form === "edit-credential" ? selected : null;

  const lastShown = useRef<string | null>(null);
  useEffect(() => {
    const id = open.credential;
    if (!id || id === lastShown.current || !byId.has(id)) return;
    lastShown.current = id;
    flash(id);
    revealElement(rows.current.get(id));
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
        description={t("This isn't an empty file. Try again in a moment.")}
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
  const credentialOf = (item: WorkerCredentialSummaryItem) =>
    item.credential
      ? (byId.get(item.credential.id) ?? (item.credential as WorkerCredentialRow))
      : null;
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
          rowRef={(key, element) => {
            if (element) rows.current.set(key, element);
            else rows.current.delete(key);
          }}
          handlers={{
            onOpen: (item) => item.credential && go(target(item.credential.id)),
            onAdd: (item) => go(addTarget(item.credentialType.id)),
            onVerify: async (item) => {
              const credential = credentialOf(item);
              if (credential) await verifyCredential(credential);
            },
            onRenew: (item) =>
              item.credential && go(target(item.credential.id, "renew-credential")),
            onEdit: (item) => item.credential && go(target(item.credential.id, "edit-credential")),
            onArchive: (item) => setArchiving(credentialOf(item)),
          }}
        />
        <CredentialHistory archived={archived} />
      </section>

      <CredentialReadPane
        open={readOpen}
        onOpenChange={closeWhen(readOpen)}
        credential={readOpen ? selected : null}
        loading={credentialsQuery.isFetching}
        earlier={selected ? earlierVersions(credentials, selected) : []}
        permissions={permissions}
        verifying={verify.isPending}
        onVerify={verifyCredential}
        onEdit={(credential) => go(target(credential.id, "edit-credential"))}
        onRenew={(credential) => go(target(credential.id, "renew-credential"))}
        onArchive={setArchiving}
      />
      <CredentialFormPane
        open={open.form === "add-credential" && canCreate}
        onOpenChange={closeWhen(open.form === "add-credential")}
        workerId={workerId}
        mode="create"
        credentialTypeId={open.form === "add-credential" ? open.kind : null}
      />
      <CredentialFormPane
        open={open.form === "renew-credential" && canCreate && formCredential != null}
        onOpenChange={closeWhen(open.form === "renew-credential")}
        workerId={workerId}
        mode="renew"
        credential={open.form === "renew-credential" ? (formCredential ?? undefined) : undefined}
        credentialTypeId={
          open.form === "renew-credential" ? formCredential?.credentialTypeId : null
        }
        onSaved={(saved) => flash(saved.id)}
      />
      <CredentialFormPane
        open={open.form === "edit-credential" && canUpdate && formCredential != null}
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
