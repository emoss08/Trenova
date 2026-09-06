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
  dqfSectionLabel,
  verificationMethodLabel,
  verificationSettled,
  verificationStatusLabel,
  verificationTone,
} from "@trenova/shared/lib/dqf";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { EmployerDialog } from "./employer-dialog";
import { useDqfInvalidation } from "./use-dqf-invalidation";

export default function WorkerDQFTab({ workerId }: { workerId: string }) {
  const { allowed: canCreate } = usePermission(Resource.Qualification, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.Qualification, Operation.Update);
  const { allowed: canDelete } = usePermission(Resource.Qualification, Operation.Delete);
  const invalidate = useDqfInvalidation(workerId);
  const [dialog, setDialog] = useState<{ verification: EmploymentVerificationRow | null } | null>(
    null,
  );

  const fileQuery = useQuery({
    queryKey: [DRIVER_QUALIFICATION_FILE_KEY, workerId],
    queryFn: ({ signal }) => fetchDriverQualificationFile(workerId, { signal }),
  });

  const requestedMutation = useMutation({
    mutationFn: (id: string) => markEmploymentVerificationRequested(id),
    onSuccess: () => {
      toast.success("Request recorded");
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error("Could not record the request", { description: error.message }),
  });

  const followUpMutation = useMutation({
    mutationFn: (id: string) => recordEmploymentVerificationFollowUp(id),
    onSuccess: (updated) => {
      toast.success("Follow-up recorded", {
        description: `${updated.followUpCount} chase${updated.followUpCount === 1 ? "" : "s"} on file as evidence of good-faith effort.`,
      });
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error("Could not record the follow-up", { description: error.message }),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteEmploymentVerification(id),
    onSuccess: () => {
      toast.success("Employer removed");
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error("Could not remove the employer", { description: error.message }),
  });

  const sections = useMemo(() => {
    const items = fileQuery.data?.items ?? [];
    return DQF_SECTION_ORDER.map((section) => ({
      section,
      items: items.filter((item) => item.section === section),
    })).filter((group) => group.items.length > 0);
  }, [fileQuery.data?.items]);

  if (fileQuery.isLoading) {
    return (
      <div className="flex flex-col gap-3 p-4">
        {[0, 1, 2].map((row) => (
          <Skeleton key={row} className="h-24 w-full" />
        ))}
      </div>
    );
  }

  const file = fileQuery.data;
  if (!file) return null;

  return (
    <div className="flex flex-col gap-4 p-4">
      <section
        className={cn(
          "rounded-lg border p-4",
          !file.complete && "border-amber-500/60 bg-amber-500/5",
        )}
      >
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0">
            <Badge variant={file.complete ? "active" : "inactive"}>
              {file.complete ? "Complete" : "Incomplete"}
            </Badge>
            <p className="text-muted-foreground mt-2 max-w-prose text-xs">
              The file is assembled on read from the credentials, documents, previous-employer
              investigations and testing record. Nothing here is a second copy, so it cannot drift
              from what those areas say.
            </p>
          </div>
          {canCreate ? (
            <Button size="sm" onClick={() => setDialog({ verification: null })}>
              Add previous employer
            </Button>
          ) : null}
        </div>

        <dl className="mt-4 grid grid-cols-2 gap-3 text-xs sm:grid-cols-4">
          <Figure
            label="Missing"
            value={String(file.missingRequired)}
            warn={file.missingRequired > 0}
          />
          <Figure label="Expired" value={String(file.expired)} warn={file.expired > 0} />
          <Figure
            label="Outstanding"
            value={String(file.outstanding)}
            warn={file.outstanding > 0}
          />
          <Figure
            label="Expiring soon"
            value={String(file.expiringSoon)}
            warn={file.expiringSoon > 0}
          />
        </dl>

        {file.safetyHistoryDueAt > 0 ? (
          <p
            className={cn(
              "mt-3 text-[11px]",
              file.safetyHistoryLate ? "text-red-600 dark:text-red-400" : "text-muted-foreground",
            )}
          >
            Previous-employer investigation was due {formatUnixDate(file.safetyHistoryDueAt)}
            {file.safetyHistoryLate ? " — it is late (49 CFR 391.23(c)(1))." : "."}
          </p>
        ) : null}

        {file.retentionExpiresAt ? (
          <p className="text-muted-foreground mt-1 text-[11px]">
            {file.purgeEligible
              ? `Held past its retention window; eligible for purge since ${formatUnixDate(file.retentionExpiresAt)}.`
              : `Hold until ${formatUnixDate(file.retentionExpiresAt)} (49 CFR 391.51(d)).`}
          </p>
        ) : null}
      </section>

      {sections.map((group) => (
        <section key={group.section}>
          <h3 className="cc-label text-foreground mb-2">{dqfSectionLabel(group.section)}</h3>
          <ul className="flex flex-col gap-1.5">
            {group.items.map((item) => (
              <ItemRow key={`${group.section}:${item.code}`} item={item} />
            ))}
          </ul>
        </section>
      ))}

      <section>
        <h3 className="cc-label text-foreground mb-2">Previous employers</h3>
        {file.verifications.length === 0 ? (
          <p className="text-muted-foreground rounded-md border border-dashed p-3 text-xs">
            No previous employer has been recorded. Until one is, the three-year investigation has
            not been made.
          </p>
        ) : (
          <ul className="flex flex-col gap-1.5">
            {file.verifications.map((verification) => (
              <li key={verification.id} className="rounded-md border px-3 py-2 text-xs">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="flex flex-wrap items-center gap-2">
                    <span className="font-medium">{verification.employerName}</span>
                    <Badge variant={verificationTone(verification.status)}>
                      {verificationStatusLabel(verification.status)}
                    </Badge>
                    {verification.wasDotRegulated ? null : (
                      <Badge variant="secondary">Non-DOT</Badge>
                    )}
                    {verification.followUpCount > 0 ? (
                      <span className="text-muted-foreground">
                        {verification.followUpCount} follow-up
                        {verification.followUpCount === 1 ? "" : "s"}
                      </span>
                    ) : null}
                  </span>
                  <span className="flex items-center gap-1">
                    {canUpdate && verification.status === "Pending" ? (
                      <Button
                        size="xs"
                        variant="outline"
                        isLoading={requestedMutation.isPending}
                        onClick={() => requestedMutation.mutate(verification.id)}
                      >
                        Mark requested
                      </Button>
                    ) : null}
                    {canUpdate && verification.status === "Requested" ? (
                      <Button
                        size="xs"
                        variant="outline"
                        isLoading={followUpMutation.isPending}
                        onClick={() => followUpMutation.mutate(verification.id)}
                      >
                        Follow up
                      </Button>
                    ) : null}
                    {canUpdate ? (
                      <Button size="xs" variant="ghost" onClick={() => setDialog({ verification })}>
                        Edit
                      </Button>
                    ) : null}
                    {canDelete && !verificationSettled(verification.status) ? (
                      <Button
                        size="xs"
                        variant="ghost"
                        isLoading={deleteMutation.isPending}
                        onClick={() => deleteMutation.mutate(verification.id)}
                      >
                        Remove
                      </Button>
                    ) : null}
                  </span>
                </div>
                <p className="text-muted-foreground mt-1">
                  {verification.employedFrom || verification.employedTo
                    ? `${verification.employedFrom ? formatUnixDate(verification.employedFrom) : "?"} – ${
                        verification.employedTo
                          ? formatUnixDate(verification.employedTo)
                          : "present"
                      } · `
                    : ""}
                  requested by {verificationMethodLabel(verification.method).toLowerCase()}
                  {verification.requestedAt
                    ? ` on ${formatUnixDate(verification.requestedAt)}`
                    : ""}
                </p>
                {verification.status === "Received" &&
                verification.wasDotRegulated &&
                !verification.drugAlcoholResponseReceivedAt ? (
                  <p className="mt-1 text-amber-600 dark:text-amber-400">
                    Answered without the drug and alcohol history (49 CFR 382.413).
                  </p>
                ) : null}
                {verification.hadAccidents || verification.hadDrugAlcoholViolations ? (
                  <p className="mt-1">
                    {verification.hadAccidents
                      ? `${verification.accidentCount} accident${verification.accidentCount === 1 ? "" : "s"} reported. `
                      : ""}
                    {verification.hadDrugAlcoholViolations
                      ? "Drug or alcohol violations reported."
                      : ""}
                  </p>
                ) : null}
                {verification.findings ? (
                  <p className="text-muted-foreground mt-1">{verification.findings}</p>
                ) : null}
              </li>
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
  return (
    <li className="flex flex-wrap items-center justify-between gap-2 rounded-md border px-3 py-2 text-xs">
      <span className="min-w-0">
        <span className="flex flex-wrap items-center gap-2">
          <span className="font-medium">{item.name}</span>
          <Badge variant={dqfItemTone(item.status)}>{dqfItemStatusLabel(item.status)}</Badge>
        </span>
        <span className="text-muted-foreground mt-0.5 block">{item.detail}</span>
      </span>
      <span className="text-muted-foreground shrink-0 tabular-nums">
        {item.expiresAt ? formatUnixDate(item.expiresAt) : ""}
      </span>
    </li>
  );
}

function Figure({ label, value, warn }: { label: string; value: string; warn?: boolean }) {
  return (
    <div>
      <dt className="text-muted-foreground text-[11px]">{label}</dt>
      <dd
        className={cn("font-semibold tabular-nums", warn && "text-amber-600 dark:text-amber-400")}
      >
        {value}
      </dd>
    </div>
  );
}
