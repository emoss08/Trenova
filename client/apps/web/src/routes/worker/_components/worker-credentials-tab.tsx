import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import {
  fetchWorkerCredentialSummary,
  fetchWorkerCredentials,
  verifyWorkerCredential,
  WORKER_CREDENTIAL_SUMMARY_KEY,
  WORKER_CREDENTIALS_KEY,
  type WorkerCredentialRow,
  type WorkerCredentialSummaryItem,
} from "@/lib/graphql/worker-credential";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { CredentialArchiveDialog } from "./credentials/credential-archive-dialog";
import {
  CredentialFormDialog,
  type CredentialFormMode,
} from "./credentials/credential-form-dialog";
import { CredentialHistory } from "./credentials/credential-history";
import { CredentialOverview } from "./credentials/credential-overview";
import {
  CredentialSlotRow,
  type CredentialSlotPermissions,
} from "./credentials/credential-slot-card";
import { useCredentialInvalidation } from "./credentials/use-credential-invalidation";

type FormState = {
  mode: CredentialFormMode;
  credential?: WorkerCredentialRow;
  credentialTypeId?: string | null;
};

export default function WorkerCredentialsTab({ workerId }: { workerId: string }) {
  const t = useT();

  const { allowed: canCreate } = usePermission(Resource.WorkerCredential, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.WorkerCredential, Operation.Update);
  const { allowed: canVerify } = usePermission(Resource.WorkerCredential, Operation.Approve);
  const { allowed: canArchive } = usePermission(Resource.WorkerCredential, Operation.Archive);
  const invalidate = useCredentialInvalidation(workerId);

  const [formState, setFormState] = useState<FormState | null>(null);
  const [archiving, setArchiving] = useState<WorkerCredentialRow | null>(null);

  const summaryQuery = useQuery({
    queryKey: [WORKER_CREDENTIAL_SUMMARY_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerCredentialSummary(workerId, { signal }),
  });
  const credentialsQuery = useQuery({
    queryKey: [WORKER_CREDENTIALS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerCredentials(workerId, true, { signal }),
  });

  const verify = useMutation({
    mutationFn: (credential: WorkerCredentialRow) =>
      verifyWorkerCredential(credential.id, credential.version),
    onSuccess: (saved) => {
      toast.success(t("Credential verified"), {
        description: `${saved.credentialType?.name ?? "The credential"} is marked as checked against its document.`,
      });
      void invalidate();
    },
    onError: (error: Error) => {
      toast.error(t("Could not verify credential"), { description: error.message });
    },
  });

  const permissions = useMemo<CredentialSlotPermissions>(
    () => ({ canCreate, canUpdate, canVerify, canArchive }),
    [canArchive, canCreate, canUpdate, canVerify],
  );

  const summary = summaryQuery.data;
  const { required, optional } = useMemo(() => {
    const items = summary?.items ?? [];
    return {
      required: items.filter((item) => item.required),
      optional: items.filter((item) => !item.required),
    };
  }, [summary]);
  const archived = useMemo(
    () => (credentialsQuery.data ?? []).filter((credential) => credential.status === "Archived"),
    [credentialsQuery.data],
  );

  const openForm = useCallback(
    (mode: CredentialFormMode, item?: WorkerCredentialSummaryItem, typeId?: string) => {
      setFormState({
        mode,
        credential: (item?.credential as WorkerCredentialRow | null | undefined) ?? undefined,
        credentialTypeId: typeId ?? item?.credentialType.id ?? null,
      });
    },
    [],
  );

  if (summaryQuery.isLoading || credentialsQuery.isLoading) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton className="h-20 w-full rounded-lg" />
        <Skeleton className="h-40 w-full rounded-lg" />
      </div>
    );
  }

  if (!summary) {
    return (
      <p className="text-muted-foreground text-sm">
        {t("Credentials could not be loaded. Try again in a moment.")}
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-5">
      <CredentialOverview
        summary={summary}
        canCreate={canCreate}
        onAdd={() => openForm("create", undefined, undefined)}
      />

      <CredentialSection
        title={t("Required")}
        hint={t("Every credential this worker must hold to be dispatched.")}
        items={required}
        permissions={permissions}
        verifyingId={verify.isPending ? verify.variables?.id : undefined}
        onAdd={(typeId) => openForm("create", undefined, typeId)}
        onRenew={(item) => openForm("renew", item)}
        onEdit={(item) => openForm("edit", item)}
        onVerify={(item) =>
          item.credential && verify.mutate(item.credential as WorkerCredentialRow)
        }
        onArchive={(item) => setArchiving((item.credential as WorkerCredentialRow) ?? null)}
        empty={t("No credential types are marked required for this driver type.")}
      />

      {optional.length > 0 ? (
        <CredentialSection
          title={t("Other credentials")}
          hint={t("Optional endorsements and certificates on file.")}
          items={optional}
          permissions={permissions}
          verifyingId={verify.isPending ? verify.variables?.id : undefined}
          onAdd={(typeId) => openForm("create", undefined, typeId)}
          onRenew={(item) => openForm("renew", item)}
          onEdit={(item) => openForm("edit", item)}
          onVerify={(item) =>
            item.credential && verify.mutate(item.credential as WorkerCredentialRow)
          }
          onArchive={(item) => setArchiving((item.credential as WorkerCredentialRow) ?? null)}
        />
      ) : null}

      <CredentialHistory archived={archived} />

      {formState ? (
        <CredentialFormDialog
          open
          onOpenChange={(open) => {
            if (!open) setFormState(null);
          }}
          workerId={workerId}
          mode={formState.mode}
          credential={formState.credential}
          credentialTypeId={formState.credentialTypeId}
        />
      ) : null}
      <CredentialArchiveDialog
        open={archiving !== null}
        onOpenChange={(open) => {
          if (!open) setArchiving(null);
        }}
        workerId={workerId}
        credential={archiving}
      />
    </div>
  );
}

type CredentialSectionProps = {
  title: string;
  hint: string;
  items: WorkerCredentialSummaryItem[];
  permissions: CredentialSlotPermissions;
  verifyingId?: string;
  empty?: string;
  onAdd: (typeId: string) => void;
  onRenew: (item: WorkerCredentialSummaryItem) => void;
  onEdit: (item: WorkerCredentialSummaryItem) => void;
  onVerify: (item: WorkerCredentialSummaryItem) => void;
  onArchive: (item: WorkerCredentialSummaryItem) => void;
};

function CredentialSection({
  title,
  hint,
  items,
  permissions,
  verifyingId,
  empty,
  onAdd,
  onRenew,
  onEdit,
  onVerify,
  onArchive,
}: CredentialSectionProps) {
  const t = useT();

  return (
    <section className="flex flex-col gap-2">
      <div className="flex items-baseline justify-between gap-3">
        <h4 className="text-muted-foreground text-[11px] font-semibold uppercase">{title}</h4>
        <p className="text-muted-foreground truncate text-xs">{hint}</p>
      </div>
      {items.length === 0 ? (
        <p className="text-muted-foreground rounded-lg border border-dashed px-3 py-4 text-center text-xs">
          {empty ?? t("Nothing on file.")}
        </p>
      ) : (
        <div className="divide-border divide-y rounded-lg border">
          {items.map((item) => (
            <CredentialSlotRow
              key={item.credentialType.id}
              item={item}
              permissions={permissions}
              verifying={Boolean(verifyingId) && item.credential?.id === verifyingId}
              onAdd={onAdd}
              onRenew={onRenew}
              onEdit={onEdit}
              onVerify={onVerify}
              onArchive={onArchive}
            />
          ))}
        </div>
      )}
    </section>
  );
}
