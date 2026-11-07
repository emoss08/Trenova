import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { updateAuditAlertPolicy, type AuditAlertPolicy } from "@/lib/graphql/audit-log-admin";
import { audit } from "@/lib/queries/audit";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Input } from "@trenova/shared/components/ui/input";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { Switch } from "@trenova/shared/components/ui/switch";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { AuditAlertPoliciesSkeleton } from "./audit-alert-policies-skeleton";

function PolicyRow({ policy, canManage }: { policy: AuditAlertPolicy; canManage: boolean }) {
  const t = useT();
  const queryClient = useQueryClient();
  const [threshold, setThreshold] = useState(String(policy.threshold));
  const [windowMinutes, setWindowMinutes] = useState(String(policy.windowMinutes));

  const save = useApiMutation({
    scope: { id: `audit-alert-policy:${policy.kind}` },
    resourceName: "Security alert",
    mutationFn: (enabled: boolean) =>
      updateAuditAlertPolicy({
        kind: policy.kind,
        enabled,
        threshold: policy.counted ? Number(threshold) : null,
        windowMinutes: policy.counted ? Number(windowMinutes) : null,
      }),
    onSuccess: () =>
      void queryClient.invalidateQueries({ queryKey: audit.alertPolicies().queryKey }),
  });

  const commit = (enabled: boolean) => {
    if (save.isPending) {
      return;
    }
    void save.mutateAsync(enabled).catch(() => undefined);
  };

  return (
    <div className="flex flex-col gap-3 rounded-md border p-3 sm:flex-row sm:items-center">
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium">{policy.title}</p>
        <p className="text-muted-foreground text-xs">{policy.description}</p>
      </div>
      {policy.counted ? (
        <div className="flex items-center gap-2 text-xs">
          <label className="flex items-center gap-1.5">
            <span className="text-muted-foreground">{t("After")}</span>
            <Input
              aria-label={t("Entries before an alert")}
              type="number"
              min={2}
              className="h-7 w-20"
              value={threshold}
              disabled={!canManage || save.isPending}
              onChange={(event) => setThreshold(event.target.value)}
              onBlur={() => threshold !== String(policy.threshold) && commit(policy.enabled)}
            />
          </label>
          <label className="flex items-center gap-1.5">
            <span className="text-muted-foreground">{t("in")}</span>
            <Input
              aria-label={t("Minutes to count over")}
              type="number"
              min={1}
              className="h-7 w-20"
              value={windowMinutes}
              disabled={!canManage || save.isPending}
              onChange={(event) => setWindowMinutes(event.target.value)}
              onBlur={() =>
                windowMinutes !== String(policy.windowMinutes) && commit(policy.enabled)
              }
            />
            <span className="text-muted-foreground">{t("minutes")}</span>
          </label>
        </div>
      ) : null}
      <div className="flex size-9 items-center justify-center">
        {save.isPending ? (
          <Spinner className="size-4" />
        ) : (
          <Switch
            aria-label={t("Alert on {0}", policy.title)}
            checked={policy.enabled}
            disabled={!canManage}
            onCheckedChange={(checked) => commit(checked)}
          />
        )}
      </div>
    </div>
  );
}

export default function AuditAlertPolicies() {
  const t = useT();
  const { allowed: canManage } = usePermission(Resource.SecurityAuditLog, Operation.Update);
  const { data, isPending, isError } = useQuery(audit.alertPolicies());

  if (isPending) {
    return <AuditAlertPoliciesSkeleton />;
  }
  if (isError || !data) {
    return (
      <p className="text-muted-foreground text-sm">{t("Security alerts could not be loaded.")}</p>
    );
  }

  return (
    <section aria-label={t("Security alerts")} className="flex flex-col gap-2">
      <p className="text-muted-foreground text-xs">
        {t(
          "Everyone who can read security entries is notified when one of these happens. Turn off the ones your team does not need.",
        )}
      </p>
      {data.map((policy) => (
        <PolicyRow key={policy.kind} policy={policy} canManage={canManage} />
      ))}
    </section>
  );
}
