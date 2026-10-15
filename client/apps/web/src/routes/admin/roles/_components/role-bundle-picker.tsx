import { applyBundle, bundleState, removeBundle, type BundleState } from "@/lib/role-bundles";
import { getPermissionBundles, type CapabilityBundle } from "@/lib/role-api";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { AddPermission } from "@trenova/shared/types/role";
import { useQuery } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import { RoleBundlePickerSkeleton } from "./role-bundle-picker-skeleton";

const moduleLabels: Record<string, string> = defineLabels({
  Dispatch: "Dispatch",
  Billing: "Billing",
  Accounting: "Accounting",
  Settlements: "Settlements",
  Safety: "Safety",
  Workforce: "Workforce",
  Customers: "Customers",
  Reporting: "Reporting",
  Platform: "Platform",
  Administration: "Administration",
});

export const permissionBundlesQuery = {
  queryKey: ["permission-bundles"] as const,
  queryFn: getPermissionBundles,
  staleTime: 1000 * 60 * 30,
};

type RoleBundlePickerProps = {
  permissions: AddPermission[];
  onPermissionsChange: (permissions: AddPermission[]) => void;
};

export function RoleBundlePicker({ permissions, onPermissionsChange }: RoleBundlePickerProps) {
  const t = useT();
  const { data, isLoading, isError } = useQuery(permissionBundlesQuery);

  const bundles = useMemo(() => data?.bundles ?? [], [data?.bundles]);
  const modules = useMemo(() => {
    const grouped = new Map<string, CapabilityBundle[]>();
    for (const bundle of bundles) {
      const list = grouped.get(bundle.module) ?? [];
      list.push(bundle);
      grouped.set(bundle.module, list);
    }
    return [...grouped.entries()];
  }, [bundles]);

  const toggle = useCallback(
    (bundle: CapabilityBundle, state: BundleState) => {
      onPermissionsChange(
        state === "full"
          ? removeBundle(permissions, bundle, bundles)
          : applyBundle(permissions, bundle),
      );
    },
    [bundles, onPermissionsChange, permissions],
  );

  if (isLoading) {
    return <RoleBundlePickerSkeleton />;
  }
  if (isError) {
    return (
      <p className="text-muted-foreground py-8 text-center text-sm">
        {t("Capabilities could not be loaded. Use the advanced view to choose permissions.")}
      </p>
    );
  }

  return (
    <div className="space-y-5">
      {modules.map(([module, moduleBundles]) => (
        <section key={module} className="space-y-2">
          <h3 className="text-muted-foreground text-xs font-medium">
            {moduleLabels[module] ?? module}
          </h3>
          <div className="grid gap-2 md:grid-cols-2">
            {moduleBundles.map((bundle) => (
              <BundleOption
                key={bundle.id}
                bundle={bundle}
                state={bundleState(bundle, permissions)}
                onToggle={toggle}
              />
            ))}
          </div>
        </section>
      ))}
    </div>
  );
}

function BundleOption({
  bundle,
  state,
  onToggle,
}: {
  bundle: CapabilityBundle;
  state: BundleState;
  onToggle: (bundle: CapabilityBundle, state: BundleState) => void;
}) {
  const t = useT();
  const summary = useMemo(
    () => bundle.grants.map((grant) => grant.displayName).join(", "),
    [bundle.grants],
  );

  return (
    <label
      className={cn(
        "flex cursor-pointer items-start gap-3 rounded-md border p-3 transition-colors",
        state === "none" ? "hover:bg-muted/50" : "border-foreground/20 bg-muted/40",
      )}
    >
      <Checkbox
        checked={state === "full"}
        indeterminate={state === "partial"}
        onCheckedChange={() => onToggle(bundle, state)}
        className="mt-0.5"
      />
      <span className="min-w-0 space-y-0.5">
        <span className="block text-sm font-medium">{t(bundle.label)}</span>
        <span className="text-muted-foreground block text-xs">{t(bundle.description)}</span>
        <span className="text-muted-foreground/80 block truncate text-xs" title={summary}>
          {state === "partial" ? t("Partly granted: {0}", summary) : summary}
        </span>
      </span>
    </label>
  );
}
