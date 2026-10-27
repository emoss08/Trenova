import { PERMISSION_CATALOG_STALE_TIME, permissionCatalog } from "@/lib/queries/permission-catalog";
import type { BundleGrant, CapabilityBundle } from "@/lib/role-api";
import { applyBundle, bundleState, removeBundles, type BundleState } from "@/lib/role-bundles";
import {
  CheckIcon,
  ChevronRightIcon,
  SearchLgIcon,
  SearchXIcon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Switch } from "@trenova/shared/components/ui/switch";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { AddPermission } from "@trenova/shared/types/role";
import { useQuery } from "@tanstack/react-query";
import { AnimatePresence, m } from "motion/react";
import { useId, useMemo, useState, type ReactNode } from "react";
import { RoleBundlePickerSkeleton } from "./role-bundle-picker-skeleton";

const moduleLabels: Record<string, string> = defineLabels({
  Desk: "Desk",
  Dispatch: "Dispatch",
  EDI: "EDI",
  Billing: "Billing",
  Accounting: "Accounting",
  Settlements: "Settlements",
  Safety: "Safety",
  Workforce: "Workforce",
  Customers: "Customers",
  Reporting: "Reporting",
  Administration: "Administration",
});

type ModuleGroup = {
  module: string;
  bundles: CapabilityBundle[];
};

type RoleBundlePickerProps = {
  permissions: AddPermission[];
  onPermissionsChange: (permissions: AddPermission[]) => void;
};

/**
 * The role editor's capabilities: the jobs a person does, each switched on as
 * a whole, grouped under the module it belongs to, with what it grants a click
 * away.
 */
export function RoleBundlePicker({ permissions, onPermissionsChange }: RoleBundlePickerProps) {
  const t = useT();
  const [searchQuery, setSearchQuery] = useState("");
  const [wantsOperationNames, setWantsOperationNames] = useState(false);

  const { data, isLoading, isError } = useQuery({
    ...permissionCatalog.bundles(),
    staleTime: PERMISSION_CATALOG_STALE_TIME,
  });
  const { data: categories } = useQuery({
    ...permissionCatalog.resources(),
    staleTime: PERMISSION_CATALOG_STALE_TIME,
    enabled: wantsOperationNames,
  });

  const bundles = useMemo(() => data?.bundles ?? [], [data?.bundles]);

  // Each resource's own name for each operation, such as "Commit" for an
  // invoice run's approve, read only once someone opens a capability.
  const operationNames = useMemo(() => {
    const names = new Map<string, Map<string, string>>();
    for (const category of categories ?? []) {
      for (const resource of category.resources) {
        names.set(
          resource.resource,
          new Map(resource.operations.map((op) => [op.operation, op.displayName])),
        );
      }
    }
    return names;
  }, [categories]);

  const permissionMap = useMemo(
    () => new Map(permissions.map((permission) => [permission.resource, permission])),
    [permissions],
  );

  const states = new Map(bundles.map((bundle) => [bundle.id, bundleState(bundle, permissions)]));
  const onCount = bundles.filter((bundle) => states.get(bundle.id) === "full").length;

  const query = searchQuery.trim().toLowerCase();
  const matches = (bundle: CapabilityBundle) =>
    query.length === 0 ||
    bundle.label.toLowerCase().includes(query) ||
    bundle.description.toLowerCase().includes(query) ||
    bundle.grants.some((grant) => grant.displayName.toLowerCase().includes(query));

  const modules: ModuleGroup[] = [];
  for (const bundle of bundles) {
    if (!matches(bundle)) {
      continue;
    }
    const group = modules.find((candidate) => candidate.module === bundle.module);
    if (group) {
      group.bundles.push(bundle);
    } else {
      modules.push({ module: bundle.module, bundles: [bundle] });
    }
  }

  const toggle = (bundle: CapabilityBundle, state: BundleState) => {
    onPermissionsChange(
      state === "full"
        ? removeBundles(permissions, [bundle], bundles)
        : applyBundle(permissions, bundle),
    );
  };

  const setModule = (group: ModuleGroup, on: boolean) => {
    onPermissionsChange(
      on
        ? group.bundles.reduce<AddPermission[]>(applyBundle, permissions)
        : removeBundles(permissions, group.bundles, bundles),
    );
  };

  if (isLoading) {
    return <RoleBundlePickerSkeleton label={t("Loading capabilities")} />;
  }
  if (isError) {
    return (
      <p className="text-foreground-muted py-8 text-center text-sm">
        {t("Capabilities could not be loaded. Use the advanced view to choose permissions.")}
      </p>
    );
  }
  if (bundles.length === 0) {
    return (
      <p className="text-foreground-muted py-8 text-center text-sm">
        {t("No capabilities are available.")}
      </p>
    );
  }

  return (
    <div className="flex h-160 flex-col gap-3">
      <div className="flex shrink-0 items-center gap-3">
        <Input
          aria-label={t("Search capabilities")}
          leftElement={<SearchLgIcon className="text-foreground-subtle size-3.5" />}
          placeholder={t("Search capabilities...")}
          value={searchQuery}
          onChange={(event) => setSearchQuery(event.target.value)}
          className="max-w-xs"
          rightElement={
            searchQuery ? (
              <Button
                type="button"
                variant="ghost"
                size="icon-xs"
                aria-label={t("Clear search")}
                onClick={() => setSearchQuery("")}
                className="text-foreground-muted hover:text-foreground"
              >
                <XCloseIcon className="size-3.5" />
              </Button>
            ) : null
          }
        />
        <span className="text-foreground-muted ml-auto text-xs tabular-nums">
          {t("{0} of {1} granted", onCount, bundles.length)}
        </span>
      </div>

      {modules.length === 0 ? (
        <div className="flex flex-1 flex-col items-center justify-center text-center">
          <SearchXIcon className="text-foreground-subtle size-6" />
          <p className="mt-3 text-sm font-medium">{t("No capabilities found")}</p>
          <p className="text-foreground-muted mt-1 text-xs">
            {t("No capabilities match “{0}”. Try a different search term.", searchQuery)}
          </p>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="mt-4"
            onClick={() => setSearchQuery("")}
          >
            {t("Clear search")}
          </Button>
        </div>
      ) : (
        <ScrollArea maskVariant="card" className="min-h-0 flex-1">
          <div className="flex flex-col gap-6 pt-1 pb-4">
            {modules.map((group) => (
              <ModuleSection
                key={group.module}
                label={moduleLabels[group.module] ?? group.module}
                granted={group.bundles.filter((bundle) => states.get(bundle.id) === "full").length}
                total={group.bundles.length}
                onSetAll={(on) => setModule(group, on)}
              >
                {group.bundles.map((bundle) => (
                  <CapabilityRow
                    key={bundle.id}
                    bundle={bundle}
                    state={states.get(bundle.id) ?? "none"}
                    permissionMap={permissionMap}
                    operationNames={categories ? operationNames : undefined}
                    onToggle={toggle}
                    onOpen={() => setWantsOperationNames(true)}
                  />
                ))}
              </ModuleSection>
            ))}
          </div>
        </ScrollArea>
      )}
    </div>
  );
}

function ModuleSection({
  label,
  granted,
  total,
  onSetAll,
  children,
}: {
  label: string;
  granted: number;
  total: number;
  onSetAll: (on: boolean) => void;
  children: ReactNode;
}) {
  const t = useT();
  const headingId = useId();
  const allOn = granted === total;

  return (
    <section aria-labelledby={headingId}>
      <div className="mb-2 flex items-baseline gap-2 px-1">
        <h3 id={headingId} className="text-sm font-semibold">
          {label}
        </h3>
        <span className="text-foreground-subtle text-xs tabular-nums">
          {t("{0} of {1} granted", granted, total)}
        </span>
        <Button
          type="button"
          variant="ghost"
          size="xs"
          className="text-foreground-muted ml-auto"
          onClick={() => onSetAll(!allOn)}
        >
          {allOn ? t("Turn off all") : t("Turn on all")}
        </Button>
      </div>
      <div className="bg-card divide-border-subtle divide-y overflow-hidden rounded-lg border">
        {children}
      </div>
    </section>
  );
}

function CapabilityRow({
  bundle,
  state,
  permissionMap,
  operationNames,
  onToggle,
  onOpen,
}: {
  bundle: CapabilityBundle;
  state: BundleState;
  permissionMap: ReadonlyMap<string, AddPermission>;
  /** Each resource's operation names; undefined until they load. */
  operationNames?: ReadonlyMap<string, ReadonlyMap<string, string>>;
  onToggle: (bundle: CapabilityBundle, state: BundleState) => void;
  onOpen: () => void;
}) {
  const t = useT();
  const labelId = useId();
  const descriptionId = useId();
  const includedId = useId();
  const [open, setOpen] = useState(false);

  return (
    <div data-bundle={bundle.id}>
      <div className="flex items-center gap-4 px-4 py-3">
        <div className="min-w-0 flex-1">
          <p id={labelId} className="text-sm font-medium">
            {bundle.label}
          </p>
          <p id={descriptionId} className="text-foreground-muted text-xs">
            {bundle.description}
          </p>
          <button
            type="button"
            aria-expanded={open}
            aria-controls={open ? includedId : undefined}
            onClick={() => {
              setOpen(!open);
              onOpen();
            }}
            className="text-foreground-subtle hover:text-foreground ui-focus-ring mt-1 inline-flex items-center gap-0.5 rounded-sm text-xs transition-colors"
          >
            {t("{0, plural, one {# resource} other {# resources}}", bundle.grants.length)}
            <ChevronRightIcon
              className={cn("size-3 transition-transform duration-200", open && "rotate-90")}
            />
          </button>
        </div>
        {state === "partial" ? (
          <span className="text-foreground-muted shrink-0 text-xs">{t("Partly granted")}</span>
        ) : null}
        <Switch
          checked={state === "full"}
          onCheckedChange={() => onToggle(bundle, state)}
          aria-labelledby={labelId}
          aria-describedby={descriptionId}
        />
      </div>
      <AnimatePresence initial={false}>
        {open ? (
          <m.div
            key="included"
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.2, ease: [0.2, 0, 0, 1] }}
            className="overflow-hidden"
          >
            <ul
              id={includedId}
              aria-label={t("What {0} grants", bundle.label)}
              className="bg-sunken border-border-subtle space-y-1.5 border-t px-4 py-3"
            >
              {bundle.grants.map((grant) => (
                <IncludedLine
                  key={grant.resource}
                  grant={grant}
                  held={grant.operations.every((operation) =>
                    permissionMap.get(grant.resource)?.operations.includes(operation),
                  )}
                  names={operationNames?.get(grant.resource)}
                  loading={operationNames === undefined}
                />
              ))}
            </ul>
          </m.div>
        ) : null}
      </AnimatePresence>
    </div>
  );
}

function IncludedLine({
  grant,
  held,
  names,
  loading,
}: {
  grant: BundleGrant;
  /** Whether the role holds every operation this grant gives. */
  held: boolean;
  names?: ReadonlyMap<string, string>;
  loading: boolean;
}) {
  const t = useT();

  return (
    <li className="flex items-center gap-2 text-xs">
      <CheckIcon
        aria-hidden
        className={cn(
          "size-3.5 shrink-0 transition-colors",
          held ? "text-foreground" : "text-foreground-subtle opacity-40",
        )}
      />
      <span className="sr-only">{held ? t("Granted") : t("Not granted")}</span>
      <span className={cn("shrink-0", !held && "text-foreground-muted")}>{grant.displayName}</span>
      {loading ? (
        <Skeleton className="h-3 w-24" />
      ) : (
        <span className="text-foreground-subtle min-w-0 truncate">
          {grant.operations.map((operation) => names?.get(operation) ?? operation).join(", ")}
        </span>
      )}
    </li>
  );
}
