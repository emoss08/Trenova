import { FactoringNoaStatusBadge } from "@/components/factoring/factoring-badges";
import {
  LazySendFactoringNoaDialog,
  preloadFactoringNoaDialogs,
} from "@/components/factoring/factoring-noa-dialogs-lazy";
import { recordPath } from "@/config/record-links";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { draftFactoringNoa, type FactoringNoa } from "@/lib/graphql/factoring-noa";
import { factoringNoa as factoringNoaQueries } from "@/lib/queries/factoring-noa";
import { invalidateFactoringQueries } from "@/lib/queries/factoring-roots";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { LinkExternal01Icon, Send01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { dateToUnixTimestamp, formatUnixDate } from "@trenova/shared/lib/date";
import type { Customer } from "@trenova/shared/types/customer";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { Link } from "react-router";

/** The notice that governs a customer and factor now: the newest one not yet released. */
export function currentFactoringNoa(notices: readonly FactoringNoa[]): FactoringNoa | null {
  return notices.find((noa) => noa.status !== "Released") ?? null;
}

/**
 * Where this customer's notice of assignment to its factor stands, beside the
 * factor it is assigned to, with the action that moves it on: draft and send
 * a notice when there is none, send the one drafted or sent before.
 */
export function CustomerFactoringNoa() {
  const t = useT();
  const { control } = useFormContext<Customer>();
  const customerId = useWatch({ control, name: "id" });
  const factoringCompanyId = useWatch({ control, name: "billingProfile.factoringCompanyId" });
  const savedFactoringCompanyId =
    useFormContext<Customer>().formState.defaultValues?.billingProfile?.factoringCompanyId;
  const { allowed: canRead } = usePermission(Resource.FactoringNoa, Operation.Read);

  if (!canRead || !factoringCompanyId) {
    return null;
  }

  if (!customerId || factoringCompanyId !== savedFactoringCompanyId) {
    return (
      <p className="text-muted-foreground text-xs" data-testid="customer-factoring-noa">
        {t("Save the customer to send it a notice of assignment for this factor.")}
      </p>
    );
  }

  return <NoaStatus customerId={customerId} factoringCompanyId={factoringCompanyId} />;
}

function NoaStatus({
  customerId,
  factoringCompanyId,
}: {
  customerId: string;
  factoringCompanyId: string;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [nowUnix] = useState(() => dateToUnixTimestamp(new Date()));
  const [sending, setSending] = useState<FactoringNoa | null>(null);
  const { allowed: canCreate } = usePermission(Resource.FactoringNoa, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.FactoringNoa, Operation.Update);
  const { data, isLoading } = useQuery(
    factoringNoaQueries.list({ customerId, factoringCompanyId }),
  );
  const current = currentFactoringNoa(data ?? []);

  const draft = useMutation({
    mutationFn: () => draftFactoringNoa({ customerId, factoringCompanyId }),
    onSuccess: (noa) => {
      invalidateFactoringQueries(queryClient);
      setSending(noa);
    },
    onError: (error) => handleMutationError({ error, resourceName: "notice of assignment" }),
  });

  if (isLoading) {
    return <Skeleton className="h-8 w-full max-w-[400px]" />;
  }

  const canSend =
    canUpdate &&
    (current === null ? canCreate : current.status === "Draft" || current.status === "Sent");

  return (
    <div
      className="flex max-w-[400px] flex-wrap items-center justify-between gap-2 rounded-md border px-3 py-2"
      data-testid="customer-factoring-noa"
    >
      <div className="flex min-w-0 flex-col gap-0.5">
        <div className="flex items-center gap-2 text-sm">
          <span className="font-medium">{t("Notice of assignment")}</span>
          {current ? <FactoringNoaStatusBadge noa={current} nowUnix={nowUnix} /> : null}
        </div>
        <p className="text-muted-foreground text-xs">
          {current === null
            ? t("None yet. Invoices are not ready to factor until the customer acknowledges one.")
            : current.status === "Acknowledged"
              ? current.expiresAt
                ? t(
                    "Acknowledged {0}, expires {1}",
                    formatUnixDate(current.acknowledgedAt),
                    formatUnixDate(current.expiresAt),
                  )
                : t("Acknowledged {0}", formatUnixDate(current.acknowledgedAt))
              : current.status === "Sent"
                ? t("Sent {0}, waiting for the signed copy", formatUnixDate(current.sentAt))
                : t("Drafted, not sent yet")}
        </p>
        {current ? (
          <Link
            to={recordPath("factoring_noa", current.id)}
            className="text-brand inline-flex items-center gap-1 text-xs font-medium hover:underline"
          >
            {t("Open notice")}
            <LinkExternal01Icon className="size-3" />
          </Link>
        ) : null}
      </div>
      {canSend ? (
        <Button
          type="button"
          size="sm"
          variant="outline"
          isLoading={draft.isPending}
          onPointerEnter={preloadFactoringNoaDialogs}
          onFocus={preloadFactoringNoaDialogs}
          onClick={() => (current ? setSending(current) : draft.mutate())}
        >
          <Send01Icon />
          {t("Send NOA")}
        </Button>
      ) : null}
      <LazySendFactoringNoaDialog
        noa={sending}
        onOpenChange={(open) => {
          if (!open) setSending(null);
        }}
      />
    </div>
  );
}
