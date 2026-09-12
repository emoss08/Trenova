import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import {
  deleteEmploymentVerification,
  DRIVER_QUALIFICATION_FILE_KEY,
  fetchDriverQualificationFile,
  markEmploymentVerificationRequested,
  recordEmploymentVerificationFollowUp,
  type DQFItem,
  type EmploymentVerificationRow,
} from "@/lib/graphql/worker-dqf";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDate } from "@trenova/shared/lib/date";
import {
  DQF_SECTION_ORDER,
  dqfItemStatusLabel,
  dqfItemTone,
  dqfNextSteps,
  dqfSectionLabel,
  dqfSectionTab,
  type DQFNextStep,
} from "@trenova/shared/lib/dqf";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { ArrowUpRightIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { DQFFileHeader } from "./dqf-file-header";
import { DQFNextSteps } from "./dqf-next-steps";
import { EmployerDialog } from "./employer-dialog";
import { EmployerRow } from "./employer-row";
import { useDqfInvalidation } from "./use-dqf-invalidation";

type WorkerDQFTabProps = {
  workerId: string;
  /** Hands the reader to the tab that fixes a gap. */
  onOpenTab: (tab: string) => void;
};

/**
 * The qualification file as a process. The header says whether the file
 * stands, the spine says which part is short, the next steps say what to do
 * about it in order, and the sections underneath are the evidence. Sending a
 * request or chasing an employer happens from the step itself.
 */
export default function WorkerDQFTab({ workerId, onOpenTab }: WorkerDQFTabProps) {
  const t = useT();

  const { allowed: canCreate } = usePermission(Resource.Qualification, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.Qualification, Operation.Update);
  const { allowed: canDelete } = usePermission(Resource.Qualification, Operation.Delete);
  const invalidate = useDqfInvalidation(workerId);
  const [dialog, setDialog] = useState<{ verification: EmploymentVerificationRow | null } | null>(
    null,
  );
  const [now] = useState(() => Math.floor(Date.now() / 1000));

  const fileQuery = useQuery({
    queryKey: [DRIVER_QUALIFICATION_FILE_KEY, workerId],
    queryFn: ({ signal }) => fetchDriverQualificationFile(workerId, { signal }),
  });

  const requested = useMutation({
    mutationFn: (id: string) => markEmploymentVerificationRequested(id),
    onSuccess: () => {
      toast.success(t("Request recorded"));
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error(t("Could not record the request"), { description: error.message }),
  });
  const followUp = useMutation({
    mutationFn: (id: string) => recordEmploymentVerificationFollowUp(id),
    onSuccess: (updated) => {
      toast.success(t("Follow-up recorded"), {
        description: `${updated.followUpCount} chase${updated.followUpCount === 1 ? "" : "s"} on file as evidence of good-faith effort.`,
      });
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error(t("Could not record the follow-up"), { description: error.message }),
  });
  const remove = useMutation({
    mutationFn: (id: string) => deleteEmploymentVerification(id),
    onSuccess: () => {
      toast.success(t("Employer removed"));
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error(t("Could not remove the employer"), { description: error.message }),
  });

  const file = fileQuery.data;
  const steps = useMemo(() => (file ? dqfNextSteps(file, now) : []), [file, now]);
  const sections = useMemo(() => {
    const items = file?.items ?? [];
    return DQF_SECTION_ORDER.map((section) => ({
      section,
      items: items.filter((item) => item.section === section),
    })).filter((group) => group.items.length > 0);
  }, [file?.items]);
  const verificationsById = useMemo(
    () => new Map((file?.verifications ?? []).map((row) => [row.id, row])),
    [file?.verifications],
  );

  if (fileQuery.isLoading) {
    return (
      <div className="flex flex-col gap-4 p-4">
        <Skeleton className="h-16 w-full rounded-lg" />
        <div className="grid grid-cols-4 gap-3">
          {[0, 1, 2, 3].map((n) => (
            <Skeleton key={n} className="h-16 rounded-lg" />
          ))}
        </div>
        <Skeleton className="h-40 w-full rounded-lg" />
      </div>
    );
  }
  if (!file) return null;

  const busy = requested.isPending || followUp.isPending || remove.isPending;
  const busyStepId = requested.isPending
    ? `verification:${requested.variables}`
    : followUp.isPending
      ? `verification:${followUp.variables}`
      : null;

  function runStep(step: DQFNextStep) {
    if (step.kind === "item" && step.tab) {
      onOpenTab(step.tab);
      return;
    }
    if (step.kind === "employers") {
      setDialog({ verification: null });
      return;
    }
    if (step.kind === "verification" && step.verificationId) {
      const verification = verificationsById.get(step.verificationId);
      if (!verification) return;
      if (step.action === "request") {
        requested.mutate(verification.id);
      } else if (step.action === "chase") {
        followUp.mutate(verification.id);
      } else {
        setDialog({ verification });
      }
    }
  }

  return (
    <div className="flex flex-col gap-5 p-4">
      <DQFFileHeader
        file={file}
        canCreate={canCreate}
        onAddEmployer={() => setDialog({ verification: null })}
        onOpenTab={onOpenTab}
      />

      <DQFNextSteps steps={steps} busyId={busyStepId} onStep={runStep} />

      {sections.map((group) => {
        const tab = dqfSectionTab(group.section);
        return (
          <section key={group.section} className="flex flex-col gap-2">
            <div className="flex items-center justify-between gap-3">
              <h4 className="text-muted-foreground text-[11px] font-semibold uppercase">
                {dqfSectionLabel(group.section)}
              </h4>
              {tab ? (
                <Button
                  size="xs"
                  variant="ghost"
                  className="text-muted-foreground"
                  onClick={() => onOpenTab(tab)}
                >
                  {t("Open {0}", tab)}
                  <ArrowUpRightIcon className="size-3" />
                </Button>
              ) : null}
            </div>
            <ul className="divide-border divide-y rounded-lg border">
              {group.items.map((item) => (
                <ItemRow key={`${group.section}:${item.code}`} item={item} />
              ))}
            </ul>
          </section>
        );
      })}

      <section className="flex flex-col gap-2">
        <div className="flex items-baseline justify-between gap-3">
          <h4 className="text-muted-foreground text-[11px] font-semibold uppercase">
            {t("Previous employers")}
          </h4>
          <p className="text-muted-foreground truncate text-xs">
            {t("Three-year lookback · 49 CFR 391.23")}
          </p>
        </div>
        {file.verifications.length === 0 ? (
          <p className="text-muted-foreground rounded-lg border border-dashed px-3 py-4 text-center text-xs">
            {t(
              "No previous employer has been recorded. Until one is, the three-year investigation has not been made.",
            )}
          </p>
        ) : (
          <ul className="divide-border divide-y rounded-lg border">
            {file.verifications.map((verification) => (
              <EmployerRow
                key={verification.id}
                verification={verification}
                permissions={{ canUpdate, canDelete }}
                now={now}
                busy={busy}
                onRequest={(row) => requested.mutate(row.id)}
                onFollowUp={(row) => followUp.mutate(row.id)}
                onEdit={(row) => setDialog({ verification: row })}
                onDelete={(row) => remove.mutate(row.id)}
              />
            ))}
          </ul>
        )}
      </section>

      <EmployerDialog
        open={dialog !== null}
        onOpenChange={(open) => !open && setDialog(null)}
        workerId={workerId}
        verification={dialog?.verification ?? null}
      />
    </div>
  );
}

function ItemRow({ item }: { item: DQFItem }) {
  const settled = item.status === "Satisfied" || item.status === "NotApplicable";
  return (
    <li className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-4 gap-y-0.5 px-3 py-2.5 text-xs">
      <div className="min-w-0">
        <p className={cn("truncate text-sm font-medium", settled && "text-muted-foreground")}>
          {item.name}
        </p>
        <p className="text-muted-foreground truncate">
          {[item.detail, item.regulation].filter(Boolean).join(" · ")}
        </p>
      </div>
      <div className="flex items-center gap-2">
        {item.expiresAt ? (
          <span className="text-muted-foreground tabular-nums">
            {formatUnixDate(item.expiresAt)}
          </span>
        ) : null}
        <Badge variant={dqfItemTone(item.status)}>{dqfItemStatusLabel(item.status)}</Badge>
      </div>
    </li>
  );
}
