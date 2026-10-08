import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  patchAIProvider,
  reorderAIProviders,
  type AIProviderPatch,
  type AIProviderRow,
  type AIProviderUsageDay,
} from "@/lib/graphql/ai-provider";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { AIProviderKind, AIProviderPreset, AITask } from "@/types/ai-provider";
import { useQueries, useQuery, useQueryClient } from "@tanstack/react-query";
import { formatList } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { useAddressedPanel, type AddressedPanel } from "../../use-addressed-panel";
import { Menu, Search, useSlashFocus, type MenuItem } from "../kit/controls";
import { Ic } from "../kit/ic";
import { SecH, Switch } from "../kit/layout";
import { Mark } from "../kit/marks";
import { ConfirmDialog } from "../kit/modal";
import { ReadSheet } from "../kit/read-sheet";
import { NovaSummary } from "../nova/nova-summary";
import type { NovaTarget } from "../nova/use-nova-segments";
import { useNovaTargets } from "../nova/use-nova-targets";
import { presetDisplayName } from "./preset-options";
import { Health, ProviderDetail, type ProviderTestState } from "./provider-detail";
import { ProviderEditor, type ProviderEditorTarget } from "./provider-editor";
import { ProviderLine, type ProviderWeek } from "./provider-line";
import {
  firstsOf,
  keyPlaceholder,
  liveState,
  moveBy,
  moveTo,
  needsKey,
  routeOf,
  sameOrder,
  taskMetas,
  toggleTask,
  uncovered,
} from "./provider-model";
import { ProviderRouting } from "./provider-routing";
import { ProvidersEmpty } from "./providers-empty";

const WEEK_DAYS = 7;
const SUMMARY_STALE_MS = 30_000;
const PENDING_REFRESH_MS = 3_000;
const USAGE_STALE_MS = 60_000;

/** The daily series alone, so the list of them keeps its identity while nothing changes. */
function dailyData(
  results: { data?: AIProviderUsageDay[] }[],
): (AIProviderUsageDay[] | undefined)[] {
  return results.map((result) => result.data);
}

/**
 * The providers in the order work is offered to them, what each takes first and how its
 * week went, and a grid of every task against every provider. A row opens the provider to
 * read; its editor holds the connection, key, tasks, access and limits.
 */
export default function ProvidersTab() {
  const t = useT();
  const queryClient = useQueryClient();
  const novaTargets = useNovaTargets();
  const searchRef = useRef<HTMLInputElement>(null);
  useSlashFocus(searchRef);
  const timezone = useMemo(() => resolveUserTimezone(), []);

  const { allowed: canCreate } = usePermission(Resource.AIProvider, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.AIProvider, Operation.Update);
  const { allowed: canDelete } = usePermission(Resource.AIProvider, Operation.Delete);
  const { allowed: canManage } = usePermission(Resource.AIProvider, Operation.Manage);

  const listQuery = useQuery(queries.aiProvider.list());
  const catalogQuery = useQuery(queries.aiProvider.catalog());
  const limitsQuery = useQuery(queries.aiProvider.limits());
  const summaryQuery = useQuery({
    ...queries.aiControl.summary("Providers"),
    staleTime: SUMMARY_STALE_MS,
    refetchInterval: (state) => (state.state.data?.pending ? PENDING_REFRESH_MS : false),
  });
  const usageQuery = useQuery({
    ...queries.aiProvider.usage(WEEK_DAYS),
    staleTime: USAGE_STALE_MS,
  });

  const saved = useMemo(() => listQuery.data ?? [], [listQuery.data]);
  const catalog = catalogQuery.data;
  const metas = useMemo(() => taskMetas(catalog?.tasks ?? []), [catalog?.tasks]);
  const keyRequired = useCallback(
    (kind: AIProviderKind) =>
      catalog?.kinds.find((entry) => entry.kind === kind)?.requiresApiKey ?? false,
    [catalog?.kinds],
  );

  const [order, setOrder] = useState<string[] | null>(null);
  const [dragging, setDragging] = useState<string | null>(null);
  const providers = useMemo(() => {
    if (!order) return saved;
    const byId = new Map(saved.map((provider) => [provider.id, provider]));
    const ordered = order.flatMap((id) => byId.get(id) ?? []);
    return ordered.length === saved.length ? ordered : saved;
  }, [order, saved]);

  const [query, setQuery] = useState("");
  const [openId, setOpenId] = useState<string | null>(null);
  const [menuOpen, setMenuOpen] = useState(false);
  const [showOff, setShowOff] = useState(false);
  const [editor, setEditor] = useState<ProviderEditorTarget | null>(null);
  const [removing, setRemoving] = useState<AIProviderRow | null>(null);
  const [tests, setTests] = useState<Record<string, ProviderTestState>>({});

  const dailies = useQueries({
    queries: saved.map((provider) => ({
      ...queries.aiProvider.daily(provider.id, WEEK_DAYS, timezone),
      staleTime: USAGE_STALE_MS,
    })),
    combine: dailyData,
  });
  const weeks = useMemo(() => {
    const slices = new Map(
      (usageQuery.data?.byProvider ?? []).map((slice) => [slice.providerId, slice]),
    );
    return new Map(
      saved.map((provider, index): [string, ProviderWeek | null] => {
        const slice = slices.get(provider.id);
        const days = dailies[index] ?? [];
        return [
          provider.id,
          slice
            ? {
                calls: slice.calls,
                failed: slice.failed,
                latencyP50Ms: slice.latencyP50Ms,
                tokens: slice.inputTokens + slice.outputTokens,
                days,
              }
            : null,
        ];
      }),
    );
  }, [dailies, saved, usageQuery.data?.byProvider]);

  const failing = useMemo(
    () =>
      new Set((summaryQuery.data?.facts.failing ?? []).map((entry) => String(entry.providerId))),
    [summaryQuery.data?.facts.failing],
  );
  const liveOf = useCallback(
    (provider: AIProviderRow) => {
      const test = tests[provider.id];
      if (test && test.state !== "run") return test.state;
      return liveState(provider.lastTest, test?.state === "run", failing.has(provider.id));
    },
    [failing, tests],
  );
  const needsKeyOf = useCallback(
    (provider: AIProviderRow) => needsKey(provider, keyRequired),
    [keyRequired],
  );

  const refresh = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queries.aiProvider._def }),
      queryClient.invalidateQueries({ queryKey: queries.aiControl._def }),
    ]);
  }, [queryClient]);

  const test = useApiMutation({
    mutationFn: (provider: AIProviderRow) => apiService.aiProviderService.test(provider.id),
    onMutate: (provider) =>
      setTests((current) => ({ ...current, [provider.id]: { state: "run" } })),
    onSuccess: async (result, provider) => {
      setTests((current) => ({
        ...current,
        [provider.id]: {
          state: result.success ? "ok" : "fail",
          message: result.message,
          at: Math.floor(Date.now() / 1000),
        },
      }));
      await refresh();
    },
    onError: (_error, provider) =>
      setTests((current) =>
        Object.fromEntries(Object.entries(current).filter(([id]) => id !== provider.id)),
      ),
    resourceName: t("AI provider"),
  });

  const patch = useApiMutation({
    mutationFn: ({
      provider,
      input,
    }: {
      provider: AIProviderRow;
      input: AIProviderPatch;
      message?: string;
    }) => patchAIProvider(provider, input),
    onSuccess: async (_saved, { message }) => {
      if (message) toast.success(message);
      await refresh();
    },
    onError: () => void refresh(),
    resourceName: t("AI provider"),
  });

  const reorder = useApiMutation({
    mutationFn: (ids: string[]) => reorderAIProviders(ids),
    onSuccess: async () => {
      await refresh();
      setOrder(null);
    },
    onError: () => {
      setOrder(null);
      void refresh();
    },
    resourceName: t("AI provider"),
  });

  const remove = useApiMutation({
    mutationFn: (provider: AIProviderRow) => apiService.aiProviderService.remove(provider.id),
    onSuccess: async (_result, provider) => {
      toast.success(t("{0} removed", provider.name));
      setRemoving(null);
      setOpenId((current) => (current === provider.id ? null : current));
      await refresh();
    },
    resourceName: t("AI provider"),
  });

  const saveKey = useApiMutation({
    mutationFn: async ({ provider, key }: { provider: AIProviderRow; key: string }) => {
      const keyed = await patchAIProvider(provider, { apiKey: key });
      setTests((current) => ({ ...current, [provider.id]: { state: "run" } }));
      const result = await apiService.aiProviderService.test(provider.id);
      if (!result.success) {
        return { provider: keyed, result, enabled: false };
      }
      const before = uncovered(metas, providers, keyRequired).map((meta) => meta.task);
      const enabled = await patchAIProvider(keyed, { enabled: true });
      return { provider: enabled, result, enabled: true, before };
    },
    onSuccess: async (outcome) => {
      setTests((current) => ({
        ...current,
        [outcome.provider.id]: {
          state: outcome.result.success ? "ok" : "fail",
          message: outcome.result.message,
          at: Math.floor(Date.now() / 1000),
        },
      }));
      if (outcome.enabled) {
        const gained = metas.filter(
          (meta) =>
            outcome.before?.includes(meta.task) && outcome.provider.tasks.includes(meta.task),
        );
        toast.success(
          gained.length > 0
            ? t(
                "{0} is on · {1} now covered",
                outcome.provider.name,
                formatList(gained.map((meta) => meta.label)),
              )
            : t("{0} is on", outcome.provider.name),
        );
      } else {
        toast.error(t("Key saved, but the test failed"), { description: outcome.result.message });
      }
      await refresh();
    },
    resourceName: t("AI provider"),
  });

  const toggleProviderTask = useCallback(
    (provider: AIProviderRow, task: AITask) =>
      patch.mutate({ provider, input: { tasks: toggleTask(provider.tasks, task) } }),
    [patch],
  );
  const setEnabled = useCallback(
    (provider: AIProviderRow, enabled: boolean) =>
      patch.mutate({
        provider,
        input: { enabled },
        message: enabled ? t("{0} is on", provider.name) : t("{0} is off", provider.name),
      }),
    [patch, t],
  );
  const commitOrder = useCallback(
    (ids: string[]) => {
      if (
        sameOrder(
          ids,
          saved.map((provider) => provider.id),
        )
      ) {
        setOrder(null);
        return;
      }
      setOrder(ids);
      reorder.mutate(ids);
    },
    [reorder, saved],
  );

  const addPreset = useCallback((preset: AIProviderPreset) => {
    setMenuOpen(false);
    setEditor({ kind: "create", preset });
  }, []);
  const editProvider = useCallback(
    (provider: AIProviderRow) => {
      const limits = limitsQuery.data?.get(provider.id);
      if (!limits) {
        toast.error(t("{0} cannot be edited yet", provider.name), {
          description:
            limitsQuery.error instanceof Error
              ? limitsQuery.error.message
              : t("Its limits and key are still loading."),
        });
        return;
      }
      setOpenId(null);
      setEditor({ kind: "edit", provider: { ...provider, ...limits } });
    },
    [limitsQuery.data, limitsQuery.error, t],
  );
  const scrollTo = useCallback((id: string) => {
    window.setTimeout(
      () => document.getElementById(id)?.scrollIntoView({ behavior: "smooth", block: "start" }),
      60,
    );
  }, []);

  useAddressedPanel(
    useCallback(
      (request: AddressedPanel) => {
        if (request.mode === "create") {
          if (!catalog) return false;
          const preset = catalog.presets.find((entry) => entry.key === request.preset);
          if (preset && canCreate) {
            setEditor({ kind: "create", preset });
          } else if (canCreate) {
            setMenuOpen(true);
          }
          return true;
        }
        if (listQuery.isLoading) return false;
        if (saved.some((provider) => provider.id === request.entityId)) {
          setOpenId(request.entityId);
          scrollTo(`pv-${request.entityId}`);
        }
        return true;
      },
      [canCreate, catalog, listQuery.isLoading, saved, scrollTo],
    ),
  );

  const onTarget = useCallback(
    (target: NovaTarget) => {
      if (target.kind === "routing") {
        scrollTo("routing");
        return;
      }
      if (target.kind === "provider") {
        setOpenId(target.providerId);
        scrollTo(`pv-${target.providerId}`);
        return;
      }
      if (target.kind === "providers") return;
      novaTargets(target);
    },
    [novaTargets, scrollTo],
  );

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      if (
        event.metaKey ||
        event.ctrlKey ||
        event.altKey ||
        editor ||
        openId ||
        (target && (target.isContentEditable || /INPUT|TEXTAREA|SELECT/.test(target.tagName)))
      ) {
        return;
      }
      if (event.key.toLowerCase() === "n" && canCreate) {
        event.preventDefault();
        setMenuOpen(true);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [canCreate, editor, openId]);

  const presets = catalog?.presets ?? [];
  const menuItems: MenuItem[] = [
    { kind: "heading", label: t("Hosted") },
    ...presets
      .filter((preset) => !preset.selfHosted)
      .map((preset): MenuItem => ({
        kind: "item",
        icon: <Mark provider={{ name: presetDisplayName(preset) }} preset={preset} s={18} />,
        label: presetDisplayName(preset),
        note: preset.exampleModel,
        onSelect: () => addPreset(preset),
      })),
    { kind: "separator" },
    { kind: "heading", label: t("On your network") },
    ...presets
      .filter((preset) => preset.selfHosted)
      .map((preset): MenuItem => ({
        kind: "item",
        icon: <Mark provider={{ name: presetDisplayName(preset) }} preset={preset} s={18} />,
        label: presetDisplayName(preset),
        note: preset.baseUrl,
        onSelect: () => addPreset(preset),
      })),
  ];

  const editorSheet =
    editor && catalog ? (
      <ProviderEditor
        key={editor.kind === "edit" ? editor.provider.id : editor.preset.key}
        target={editor}
        providers={saved}
        catalog={catalog}
        metas={metas}
        keyRequired={keyRequired}
        canDelete={canDelete}
        onClose={() => setEditor(null)}
        onRemove={(provider) => remove.mutateAsync(provider)}
      />
    ) : null;

  if (listQuery.isLoading || catalogQuery.isLoading) {
    return (
      <div className="tabp" aria-busy>
        <NovaSummary context={t("Providers")} segments={undefined} loading onTarget={onTarget} />
      </div>
    );
  }

  if (saved.length === 0) {
    return (
      <>
        <ProvidersEmpty presets={presets} canCreate={canCreate} onPick={addPreset} />
        {editorSheet}
      </>
    );
  }

  const needle = query.trim().toLowerCase();
  const kindLabel = (kind: AIProviderKind) =>
    catalog?.kinds.find((entry) => entry.kind === kind)?.label ?? kind;
  const matches = providers.filter((provider) =>
    `${provider.name} ${provider.model} ${kindLabel(provider.kind)}`.toLowerCase().includes(needle),
  );
  const off = providers.filter((provider) => !provider.enabled);
  const visible = matches.filter((provider) => provider.enabled || showOff || needle !== "");
  const onCount = providers.length - off.length;
  const bad = providers.find((provider) => provider.enabled && liveOf(provider) === "fail");
  const waitingKey = providers.find(needsKeyOf);
  const labelOf = new Map(metas.map((meta) => [meta.task, meta.label]));
  const open = providers.find((provider) => provider.id === openId) ?? null;
  const openIndex = open ? providers.indexOf(open) : -1;
  const ids = providers.map((provider) => provider.id);

  return (
    <div className="tabp">
      <NovaSummary
        context={t("Providers")}
        segments={summaryQuery.data?.segments}
        loading={summaryQuery.isLoading}
        onTarget={onTarget}
        control={
          bad && canManage ? (
            <>
              <button type="button" className="btn ink lg" onClick={() => test.mutate(bad)}>
                <Ic n="plug" s={13} />
                {t("Test {0}", bad.name)}
              </button>
              <span>{bad.baseUrl || kindLabel(bad.kind)}</span>
            </>
          ) : waitingKey && canUpdate ? (
            <>
              <button type="button" className="btn ink lg" onClick={() => setOpenId(waitingKey.id)}>
                <Ic n="key" s={13} />
                {t("Add {0} key", waitingKey.name)}
              </button>
              {waitingKey.tasks.length > 0 && (
                <span>
                  {t(
                    "Takes {0}",
                    formatList(waitingKey.tasks.map((task) => labelOf.get(task) ?? task)),
                  )}
                </span>
              )}
            </>
          ) : undefined
        }
      />
      <div className="tb">
        <Search
          value={query}
          onChange={setQuery}
          placeholder={t("Search providers")}
          inputRef={searchRef}
        />
        <span className="sp" />
        <span className="tb-ct mono">{t("{0} of {1} on", onCount, providers.length)}</span>
        {canCreate && (
          <div className="rel">
            <button
              type="button"
              className="btn ink"
              onClick={() => setMenuOpen((value) => !value)}
            >
              <Ic n="plus" s={13} />
              {t("New provider")}
              <span className="kbd">N</span>
            </button>
            {menuOpen && (
              <Menu
                right
                label={t("New provider")}
                items={menuItems}
                onClose={() => setMenuOpen(false)}
              />
            )}
          </div>
        )}
      </div>
      <section className="sec chain-s">
        <SecH
          t={t("The chain")}
          n={t("{0} providers · top to bottom", providers.length)}
          r={canUpdate ? <span className="sh2-n">{t("Drag to reorder")}</span> : undefined}
        />
        <ol className="pls">
          {visible.map((provider) => (
            <ProviderLine
              key={provider.id}
              provider={provider}
              index={providers.indexOf(provider)}
              open={openId === provider.id}
              live={liveOf(provider)}
              needsKey={needsKeyOf(provider)}
              firsts={firstsOf(provider, metas, providers, keyRequired).length}
              week={weeks.get(provider.id) ?? null}
              dragging={dragging === provider.id}
              canReorder={canUpdate && !reorder.isPending}
              canTest={canManage}
              canToggle={canUpdate}
              onOpen={(focusKey) =>
                setOpenId((current) => (focusKey || current !== provider.id ? provider.id : null))
              }
              onTest={() => test.mutate(provider)}
              onToggle={(enabled) => setEnabled(provider, enabled)}
              onDragStart={() => {
                setOrder(ids);
                setDragging(provider.id);
              }}
              onDragOver={() => {
                if (dragging && dragging !== provider.id) {
                  setOrder((current) => moveTo(current ?? ids, dragging, provider.id));
                }
              }}
              onDragEnd={() => {
                setDragging(null);
                commitOrder(order ?? ids);
              }}
            />
          ))}
          {needle === "" && off.length > 0 && (
            <li
              className="pl-off"
              role="button"
              tabIndex={0}
              aria-expanded={showOff}
              onClick={() => setShowOff((value) => !value)}
              onKeyDown={(event) => {
                if (event.key === "Enter" || event.key === " ") {
                  event.preventDefault();
                  setShowOff((value) => !value);
                }
              }}
            >
              <Ic n={showOff ? "chevD" : "chevR"} s={12} />
              <span>{showOff ? t("Hide {0} off", off.length) : t("Show {0} off", off.length)}</span>
              <span className="pl-om">
                {off.map((provider) => (
                  <Mark key={provider.id} provider={provider} s={18} />
                ))}
              </span>
              <em>{t("They keep their place in line and are skipped until turned on.")}</em>
            </li>
          )}
          {canCreate && (
            <li
              className="pl-add"
              role="button"
              tabIndex={0}
              onClick={() => setMenuOpen(true)}
              onKeyDown={(event) => {
                if (event.key === "Enter" || event.key === " ") {
                  event.preventDefault();
                  setMenuOpen(true);
                }
              }}
            >
              <Ic n="plus" s={13} />
              {t("Add a provider")}
            </li>
          )}
        </ol>
      </section>
      <ProviderRouting
        providers={providers}
        metas={metas}
        keyRequired={keyRequired}
        canAssign={canUpdate}
        onToggle={toggleProviderTask}
      />
      <ReadSheet
        open={open !== null}
        onClose={() => setOpenId(null)}
        label={open?.name ?? t("Provider")}
        head={
          open && (
            <>
              <Mark provider={open} s={36} />
              <div className="sh-t">
                <b>{open.name}</b>
                <span className="mono">{open.model}</span>
              </div>
              <Switch
                on={open.enabled}
                disabled={needsKeyOf(open) || !canUpdate}
                label={open.enabled ? t("Turn off") : t("Turn on")}
                onChange={(enabled) => setEnabled(open, enabled)}
              />
            </>
          )
        }
      >
        {open && (
          <>
            {canUpdate && (
              <div className="sh-act">
                <button type="button" className="btn sm" onClick={() => editProvider(open)}>
                  <Ic n="edit" s={12} />
                  {t("Edit connection")}
                </button>
              </div>
            )}
            <div className="sh-m">
              <span>{t("Priority {0}", openIndex + 1)}</span>
              <span>{kindLabel(open.kind)}</span>
              <span className="mono">{open.baseUrl || t("Provider default")}</span>
            </div>
            <div className="sh-hl">
              <Health provider={open} test={tests[open.id]} week={weeks.get(open.id) ?? null} />
              {!needsKeyOf(open) && canManage && (
                <button
                  type="button"
                  className="btn sm"
                  disabled={tests[open.id]?.state === "run"}
                  onClick={() => test.mutate(open)}
                >
                  <Ic n="plug" s={12} />
                  {t("Test")}
                </button>
              )}
            </div>
            <ProviderDetail
              key={open.id}
              provider={open}
              index={openIndex}
              count={providers.length}
              metas={metas}
              firsts={
                new Set(
                  metas
                    .filter((meta) => routeOf(meta, providers, keyRequired).first?.id === open.id)
                    .map((meta) => meta.task),
                )
              }
              needsKey={needsKeyOf(open)}
              keyPlaceholder={keyPlaceholder(open.kind, open.baseUrl) ?? t("API key")}
              week={weeks.get(open.id) ?? null}
              canUpdate={canUpdate}
              canDelete={canDelete}
              savingKey={saveKey.isPending}
              onSaveKey={(key) => saveKey.mutate({ provider: open, key })}
              onToggleTask={(task) => toggleProviderTask(open, task)}
              onAccess={(input) => patch.mutate({ provider: open, input })}
              onPrices={(prices) =>
                patch.mutate({
                  provider: open,
                  input: {
                    inputCostPerMillion: prices.input,
                    outputCostPerMillion: prices.output,
                  },
                  message: t("Prices saved"),
                })
              }
              onEdit={() => editProvider(open)}
              onMove={(delta) => commitOrder(moveBy(ids, open.id, delta))}
              onRemove={() => setRemoving(open)}
            />
          </>
        )}
      </ReadSheet>
      <ConfirmDialog
        open={removing !== null}
        onClose={() => setRemoving(null)}
        title={t("Remove {0}?", removing?.name ?? "")}
        description={
          removing && removing.tasks.length > 0
            ? t("Its tasks fall to the next provider in line. The stored key is deleted.")
            : t("The stored key is deleted.")
        }
        confirmLabel={t("Remove provider")}
        danger
        busy={remove.isPending}
        onConfirm={() => removing && remove.mutate(removing)}
      />
      {editorSheet}
      <span className={cn("sr-only")} aria-live="polite">
        {reorder.isPending ? t("Saving the new order") : ""}
      </span>
    </div>
  );
}
