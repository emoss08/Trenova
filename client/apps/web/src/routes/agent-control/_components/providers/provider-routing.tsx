import type { AIProviderRow } from "@/lib/graphql/ai-provider";
import type { AIProviderKind, AITask } from "@/types/ai-provider";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties } from "react";
import { Ic } from "../kit/ic";
import { SecH } from "../kit/layout";
import { Mark } from "../kit/marks";
import { routeOf, skipReason, type SkipReason, type TaskMeta } from "./provider-model";
import { TASK_FALLBACKS } from "./task-fallbacks";

type ProviderRoutingProps = {
  providers: readonly AIProviderRow[];
  metas: readonly TaskMeta[];
  keyRequired: (kind: AIProviderKind) => boolean;
  canAssign: boolean;
  onToggle: (provider: AIProviderRow, task: AITask) => void;
};

/**
 * Every task against every provider in order: who takes it, who is assigned but passed
 * over and why, and where it goes. A cell assigns or unassigns.
 */
export function ProviderRouting({
  providers,
  metas,
  keyRequired,
  canAssign,
  onToggle,
}: ProviderRoutingProps) {
  const t = useT();
  const rt = useRichT();
  const covered = metas.filter((meta) => routeOf(meta, providers, keyRequired).first).length;
  const skipLabel: Record<SkipReason | "backup", string> = {
    off: t("off"),
    needsKey: t("needs key"),
    untrusted: t("not trusted"),
    noEmbedding: t("no vector size"),
    backup: t("backup"),
  };

  return (
    <section className="sec" id="routing">
      <SecH t={t("Routing")} n={t("{0} of {1} covered", covered, metas.length)} />
      <p className="lead">
        {rt(
          "Each task goes to the first provider, in order, that's on, assigned and — where marked <shield/> — trusted. Click a cell to assign or unassign.",
          { shield: () => <Ic n="shield" s={11} /> },
        )}
      </p>
      <div className="mx-w">
        <div className="mx" style={{ "--cols": providers.length } as CSSProperties} role="table">
          <div className="mx-h" role="row">
            <span role="columnheader">{t("Task")}</span>
            {providers.map((provider, index) => (
              <span
                key={provider.id}
                role="columnheader"
                className={cn("mx-ph", !provider.enabled && "dim")}
              >
                <em className="mono">{index + 1}</em>
                <Mark provider={provider} s={16} />
                <b>{provider.name}</b>
              </span>
            ))}
            <span role="columnheader">{t("Goes to")}</span>
          </div>
          {metas.map((meta) => {
            const route = routeOf(meta, providers, keyRequired);
            return (
              <div key={meta.task} role="row" className={cn("mx-r", !route.first && "un")}>
                <span role="rowheader" className="mx-t">
                  {meta.label}
                  {meta.trust && <Ic n="shield" s={11} />}
                </span>
                {providers.map((provider) => {
                  const assigned = provider.tasks.includes(meta.task);
                  const first = route.first?.id === provider.id;
                  const why = skipLabel[skipReason(provider, meta, keyRequired) ?? "backup"];
                  const title = assigned
                    ? first
                      ? t("{0} takes it", provider.name)
                      : t("{0}: {1}", provider.name, why)
                    : t("Assign to {0}", provider.name);
                  return (
                    <button
                      key={provider.id}
                      type="button"
                      role="cell"
                      className={cn(
                        "mx-c",
                        assigned && "as",
                        first && "first",
                        assigned && !first && "skip",
                      )}
                      title={title}
                      aria-label={`${meta.label} · ${title}`}
                      aria-pressed={assigned}
                      disabled={!canAssign}
                      onClick={() => onToggle(provider, meta.task)}
                    >
                      <i />
                      {assigned && !first && <em>{why}</em>}
                    </button>
                  );
                })}
                <span role="cell" className="mx-g">
                  {route.first ? (
                    <>
                      <b>{route.first.name}</b>
                      {route.next && <em>{t("then {0}", route.next.name)}</em>}
                    </>
                  ) : (
                    <span className="t-w">{TASK_FALLBACKS[meta.task]}</span>
                  )}
                </span>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
