import type { AIProviderRow } from "@/lib/graphql/ai-provider";
import type { AITask } from "@/types/ai-provider";
import { formatList } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { InputField } from "@/components/fields/input-field";
import type { KeyboardEvent } from "react";
import { useForm, useWatch } from "react-hook-form";
import { aicFieldTrigger } from "../edit/field-trigger";
import { Ic } from "../kit/ic";
import { Switch } from "../kit/layout";
import type { ProviderWeek } from "./provider-line";
import { formatLatency, formatTokens, type TaskMeta } from "./provider-model";
import { Button } from "@trenova/shared/components/ui/button";

/** A test someone ran from this page, until the list reloads with its outcome. */
export type ProviderTestState =
  | { state: "run" }
  | { state: "ok" | "fail"; message: string; at: number };

type HealthProps = {
  provider: AIProviderRow;
  test: ProviderTestState | undefined;
  week: ProviderWeek | null;
};

/** How the endpoint last answered, and how its week went. */
export function Health({ provider, test, week }: HealthProps) {
  const t = useT();
  if (test?.state === "run") {
    return (
      <span className="hl run">
        <i className="spn" />
        {t("Testing connection…")}
      </span>
    );
  }
  const result =
    test ??
    (provider.lastTest
      ? {
          state: provider.lastTest.success ? ("ok" as const) : ("fail" as const),
          message:
            provider.lastTest.success && provider.lastTest.latencyMs > 0
              ? t("{0} · {1} ms", provider.lastTest.message, provider.lastTest.latencyMs)
              : provider.lastTest.message,
          at: provider.lastTest.testedAt,
        }
      : null);

  return (
    <span className="hl-w">
      {result ? (
        <span className={cn("hl", result.state === "ok" ? "ok" : "bad")} title={result.message}>
          <i />
          <span className="hl-m">{result.message}</span>
          <em>{formatUnixDateTimeShort(result.at)}</em>
        </span>
      ) : (
        <span className="hl nv">
          <i />
          {t("Never tested")}
        </span>
      )}
      {week && week.calls > 0 && (
        <span className="hl-s mono">
          {t(
            "{0} calls · {1}% failed",
            week.calls.toLocaleString(),
            ((week.failed / week.calls) * 100).toFixed(1),
          )}
        </span>
      )}
    </span>
  );
}

type ProviderDetailProps = {
  provider: AIProviderRow;
  index: number;
  count: number;
  metas: readonly TaskMeta[];
  /** The tasks this provider takes first. */
  firsts: ReadonlySet<AITask>;
  needsKey: boolean;
  keyPlaceholder: string;
  week: ProviderWeek | null;
  canUpdate: boolean;
  canDelete: boolean;
  savingKey: boolean;
  onSaveKey: (key: string) => void;
  onToggleTask: (task: AITask) => void;
  onAccess: (patch: { trusted?: boolean; allowPrivateNetwork?: boolean }) => void;
  onPrices: (prices: { input: string | null; output: string | null }) => void;
  onEdit: () => void;
  onMove: (delta: -1 | 1) => void;
  onRemove: () => void;
};

/** A provider in its read sheet: its key, what it handles, access, price and week. */
export function ProviderDetail({
  provider,
  index,
  count,
  metas,
  firsts,
  needsKey,
  keyPlaceholder,
  week,
  canUpdate,
  canDelete,
  savingKey,
  onSaveKey,
  onToggleTask,
  onAccess,
  onPrices,
  onEdit,
  onMove,
  onRemove,
}: ProviderDetailProps) {
  const t = useT();
  const keyForm = useForm<{ key: string }>({ defaultValues: { key: "" } });
  const key = useWatch({ control: keyForm.control, name: "key" });
  const labels = new Map(metas.map((meta) => [meta.task, meta.label]));
  const handled = provider.tasks.map((task) => labels.get(task) ?? task);
  const local = !needsKey && !provider.hasApiKey;

  return (
    <div className="pd">
      {needsKey && (
        <div className="pd-key">
          <div>
            <b>{t("Add an API key to turn {0} on", provider.name)}</b>
            <span>
              {handled.length > 0
                ? t(
                    "Stored encrypted and never shown again. It will take {0}.",
                    formatList(handled, "conjunction"),
                  )
                : t("Stored encrypted and never shown again.")}
            </span>
          </div>
          <form
            className="flex items-start gap-2"
            onSubmit={keyForm.handleSubmit((values) => {
              if (values.key.trim()) onSaveKey(values.key.trim());
            })}
          >
            <InputField<{ key: string }>
              control={keyForm.control}
              name="key"
              rules={{ required: true }}
              type="password"
              autoFocus
              autoComplete="new-password"
              aria-label={t("API key")}
              placeholder={keyPlaceholder}
              disabled={!canUpdate || savingKey}
              leftElement={<Ic n="key" s={13} />}
              className="min-w-0 flex-1"
              inputClassProps={aicFieldTrigger}
            />
            <Button
              type="submit"
              variant="default"
              size="sm"
              disabled={!key.trim()}
              isLoading={savingKey}
              loadingText={t("Save and test")}
            >
              {t("Save and test")}
            </Button>
          </form>
        </div>
      )}
      <div className="pd-g">
        <div className="pd-s wide">
          <h4>
            {t("Handles")}
            <em className="mono">{provider.tasks.length}</em>
          </h4>
          <div className="tks" role="group" aria-label={t("Handles")}>
            {metas.map((meta) => {
              const on = provider.tasks.includes(meta.task);
              const first = firsts.has(meta.task);
              return (
                <button
                  key={meta.task}
                  type="button"
                  className={cn("tkb", on && "on", first && "first")}
                  aria-pressed={on}
                  disabled={!canUpdate}
                  title={
                    first
                      ? t("This provider takes it first")
                      : on
                        ? t("Assigned; an earlier provider takes it first")
                        : t("Not assigned")
                  }
                  onClick={() => onToggleTask(meta.task)}
                >
                  <Ic n={on ? "check" : "plus"} s={11} w={2.2} />
                  {meta.label}
                  {meta.trust && <Ic n="shield" s={10} />}
                </button>
              );
            })}
          </div>
        </div>
        <div className="pd-s">
          <h4>{t("Access")}</h4>
          <div className="sw-r">
            <span>
              <b>{t("Trusted")}</b>
              <em>{t("May take tasks that read sensitive records")}</em>
            </span>
            <Switch
              on={provider.trusted}
              label={t("Trusted")}
              disabled={!canUpdate}
              onChange={(trusted) => onAccess({ trusted })}
            />
          </div>
          <div className="sw-r">
            <span>
              <b>{t("Private network")}</b>
              <em>{t("May reach a server on your own network")}</em>
            </span>
            <Switch
              on={provider.allowPrivateNetwork}
              label={t("Private network")}
              disabled={!canUpdate}
              onChange={(allowPrivateNetwork) => onAccess({ allowPrivateNetwork })}
            />
          </div>
        </div>
        <div className="pd-s">
          <h4>{t("Price per million tokens")}</h4>
          <Prices
            key={`${provider.id}:${provider.version}`}
            provider={provider}
            disabled={!canUpdate}
            onSave={onPrices}
          />
          <p className="ad-h">
            {local
              ? t("Leave empty for your own hardware; spend shows as —.")
              : t("Used to show spend on the overview.")}
          </p>
        </div>
        <div className="pd-s">
          <h4>{t("Last 7 days")}</h4>
          {week && week.calls > 0 ? (
            <dl className="st4">
              <div>
                <dt>{t("Calls")}</dt>
                <dd className="mono">{week.calls.toLocaleString()}</dd>
              </div>
              <div>
                <dt>{t("Failed")}</dt>
                <dd className={cn("mono", week.failed > 0 && "t-d")}>
                  {week.failed.toLocaleString()}
                </dd>
              </div>
              <div>
                <dt>{t("Median")}</dt>
                <dd className="mono">{formatLatency(week.latencyP50Ms)}</dd>
              </div>
              <div>
                <dt>{t("Tokens")}</dt>
                <dd className="mono">{formatTokens(week.tokens)}</dd>
              </div>
            </dl>
          ) : (
            <p className="ad-h">{t("No calls yet.")}</p>
          )}
        </div>
      </div>
      <div className="ad-bar">
        {canUpdate && (
          <>
            <button type="button" className="xa" onClick={onEdit}>
              <Ic n="edit" s={13} />
              {t("Edit connection")}
            </button>
            <button type="button" className="xa" disabled={index === 0} onClick={() => onMove(-1)}>
              <Ic n="up" s={13} />
              {t("Move up")}
            </button>
            <button
              type="button"
              className="xa"
              disabled={index === count - 1}
              onClick={() => onMove(1)}
            >
              <Ic n="down" s={13} />
              {t("Move down")}
            </button>
          </>
        )}
        <span className="sp" />
        {canDelete && (
          <button type="button" className="xa d" onClick={onRemove}>
            <Ic n="trash" s={13} />
            {t("Remove")}
          </button>
        )}
      </div>
    </div>
  );
}

const PRICE = /^\d{0,6}(\.\d{0,6})?$/;
const DOLLAR = <span className="text-muted-foreground text-xs">$</span>;

type PriceValues = { input: string; output: string };

function normalizedPrice(text: string): string | null {
  const trimmed = text.trim();
  return trimmed === "" ? null : String(Number(trimmed));
}

/** The two prices, saved when a box is left with a different, valid amount. */
function Prices({
  provider,
  disabled,
  onSave,
}: {
  provider: AIProviderRow;
  disabled: boolean;
  onSave: (prices: { input: string | null; output: string | null }) => void;
}) {
  const t = useT();
  const form = useForm<PriceValues>({
    defaultValues: {
      input: provider.inputCostPerMillion ?? "",
      output: provider.outputCostPerMillion ?? "",
    },
  });

  const commit = () => {
    const { input, output } = form.getValues();
    if (!PRICE.test(input.trim()) || !PRICE.test(output.trim())) {
      form.reset({
        input: provider.inputCostPerMillion ?? "",
        output: provider.outputCostPerMillion ?? "",
      });
      return;
    }
    const next = { input: normalizedPrice(input), output: normalizedPrice(output) };
    const current = {
      input: normalizedPrice(provider.inputCostPerMillion ?? ""),
      output: normalizedPrice(provider.outputCostPerMillion ?? ""),
    };
    if (next.input !== current.input || next.output !== current.output) {
      onSave(next);
    }
  };
  const blurOnEnter = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter") event.currentTarget.blur();
  };

  return (
    <div className="grid grid-cols-2 gap-3">
      <InputField<PriceValues>
        control={form.control}
        name="input"
        label={t("Input")}
        inputMode="decimal"
        placeholder="—"
        disabled={disabled}
        leftElement={DOLLAR}
        inputClassProps={aicFieldTrigger}
        onBlur={commit}
        onKeyDown={blurOnEnter}
      />
      <InputField<PriceValues>
        control={form.control}
        name="output"
        label={t("Output")}
        inputMode="decimal"
        placeholder="—"
        disabled={disabled}
        leftElement={DOLLAR}
        inputClassProps={aicFieldTrigger}
        onBlur={commit}
        onKeyDown={blurOnEnter}
      />
    </div>
  );
}
