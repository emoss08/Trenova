import { humanizeToolName } from "@/components/assistant/proposal-state";
import { toolRefusal, type ToolRefusal, type ToolStep } from "@/components/assistant/activity";
import type { AssistantProposal } from "@/types/assistant";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo, useState } from "react";
import { DeskErrorButton, DeskErrorCard } from "../desk-error-card";
import { DeskIcon, type DeskIconName } from "../desk-icons";

/** One record a write or a step did not get to, and why. */
export type BulkItem = { id: string; name: string; reason: string };

/**
 * A long list of records that did not go through, short by default: the
 * reasons as chips that filter it, the first few rows, and "Show all" for
 * the rest with a filter box once the list is long.
 */
export function DeskBulkList({
  items,
  limit = 3,
  noun,
}: {
  items: readonly BulkItem[];
  limit?: number;
  noun: string;
}) {
  const t = useT();
  const reasons = useMemo(() => {
    const counts = new Map<string, number>();
    for (const item of items) counts.set(item.reason, (counts.get(item.reason) ?? 0) + 1);
    return [...counts.entries()];
  }, [items]);
  const [reason, setReason] = useState<string | null>(null);
  const [all, setAll] = useState(false);
  const [query, setQuery] = useState("");

  let rows = reason ? items.filter((item) => item.reason === reason) : items;
  if (query.trim() !== "") {
    const needle = query.trim().toLowerCase();
    rows = rows.filter((item) => `${item.id} ${item.name}`.toLowerCase().includes(needle));
  }
  const shown = all ? rows : rows.slice(0, limit);

  return (
    <div className="dk-bl">
      {reasons.length > 1 && (
        <div className="dk-bl-rs">
          {reasons.map(([text, count]) => (
            <button
              key={text}
              type="button"
              className={cn(reason === text && "dk-on")}
              aria-pressed={reason === text}
              onClick={() => setReason((value) => (value === text ? null : text))}
            >
              <b>{count}</b>
              {text}
            </button>
          ))}
        </div>
      )}
      {all && items.length > 8 && (
        <label className="dk-bl-q">
          <DeskIcon name="search" size={12} />
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={t("Filter {0} {1}", rows.length, noun)}
            aria-label={t("Filter {0}", noun)}
          />
        </label>
      )}
      <div className={cn("dk-bl-l", all && "dk-all")}>
        {shown.map((item, index) => (
          <div key={`${item.id}-${index}`}>
            <span className="dk-ax-id">{item.id}</span>
            <span>{item.name}</span>
            {!reason && reasons.length > 1 ? <em>{item.reason}</em> : <em />}
          </div>
        ))}
        {shown.length === 0 && (
          <div className="dk-bl-none">{t("No {0} match “{1}”", noun, query)}</div>
        )}
      </div>
      {rows.length > limit && (
        <div className="dk-bl-f">
          <button type="button" className="dk-ec-link" onClick={() => setAll((value) => !value)}>
            {all ? t("Show fewer") : t("Show all {0} {1}", rows.length, noun)}
          </button>
        </div>
      )}
    </div>
  );
}

function capitalize(text: string): string {
  return text.charAt(0).toUpperCase() + text.slice(1);
}

/**
 * An approved change that did not all go through: how many of how many,
 * that the ones that went through are final and the rest were left as they
 * were, the list of the rest with why, and a way to ask the agent to fix
 * them. A change that failed outright says so with the reason it gave.
 */
export function DeskWriteResultCard({
  proposal,
  onAsk,
}: {
  proposal: Pick<AssistantProposal, "status" | "executionError" | "executionResult">;
  onAsk?: (text: string) => void;
}) {
  const t = useT();
  const result = proposal.executionResult;
  const failed = result?.failed ?? [];

  if (proposal.status === "ExecutionFailed") {
    return (
      <DeskErrorCard
        tone="err"
        title={t("The change didn't go through")}
        sub={proposal.executionError || t("Nothing was changed.")}
        actions={
          onAsk ? (
            <DeskErrorButton
              ink
              onClick={() => onAsk(t("The change didn't go through. Can you fix it?"))}
            >
              {t("Ask the agent to fix it")}
            </DeskErrorButton>
          ) : undefined
        }
      />
    );
  }
  if (!result || failed.length === 0) {
    return null;
  }

  const total = result.total ?? failed.length;
  const done = Math.max(0, total - failed.length);
  const noun = result.kind || t("records");
  const action = result.action || t("changed");
  // A failed record is named by its label, the way the app names it; the id
  // stands in only when the write could not name it.
  const items = failed.map((item) => ({
    id: item.label || item.id,
    name: "",
    reason: item.reason,
  }));

  return (
    <DeskErrorCard
      tone="err"
      title={t(
        "{0} {1} of {2} {3} · {4} didn't go through",
        capitalize(action),
        done,
        total,
        noun,
        failed.length,
      )}
      sub={t(
        "The {0} stayed as they were. The {1} that {2} are final.",
        failed.length,
        done,
        action,
      )}
    >
      <DeskBulkList items={items} noun={noun} />
      {onAsk && (
        <div className="dk-ec-acts">
          <DeskErrorButton
            ink
            onClick={() =>
              onAsk(
                t(
                  "{0} {1} didn't go through. Look at why and fix them: {2}",
                  failed.length,
                  noun,
                  failed
                    .slice(0, 20)
                    .map((item) => `${item.label || item.id} (${item.reason})`)
                    .join("; "),
                ),
              )
            }
          >
            {t("Ask the agent to fix these {0}", failed.length)}
          </DeskErrorButton>
        </div>
      )}
    </DeskErrorCard>
  );
}

type StepKind = "failed" | ToolRefusal;

const KIND_ORDER: StepKind[] = ["failed", "denied", "over_budget", "invalid", "duplicate"];

const KIND_ICON: Record<StepKind, DeskIconName> = {
  failed: "x",
  denied: "lock",
  over_budget: "alert",
  invalid: "info",
  duplicate: "undo",
};

const KIND_CLASS: Record<StepKind, string> = {
  failed: "dk-s-failed",
  denied: "dk-s-denied",
  over_budget: "dk-s-budget",
  invalid: "dk-s-invalid",
  duplicate: "dk-s-dup",
};

function kindLabel(kind: StepKind, count: number, t: TranslateFn): string {
  switch (kind) {
    case "failed":
      return t("{0, plural, one {# didn't work} other {# didn't work}}", count);
    case "denied":
      return t("{0, plural, one {# not permitted} other {# not permitted}}", count);
    case "over_budget":
      return t("{0, plural, one {# out of budget} other {# out of budget}}", count);
    case "invalid":
      return t("{0, plural, one {# not accepted} other {# not accepted}}", count);
    case "duplicate":
      return t("{0, plural, one {# skipped} other {# skipped}}", count);
  }
}

/**
 * Why a step did not go through, from the first line of its result, without
 * the runtime's stock opening ("Tool \"x\" was not run:"), which the row
 * already says by naming the step.
 */
function stepReason(step: ToolStep): string {
  const line = step.content.split("\n").find((part) => part.trim() !== "") ?? "";
  const reason = line.replace(/^Tool\s+"[^"]+"\s+(was not run|failed)\s*[:.]\s*/u, "").trim();
  const text = reason.charAt(0).toUpperCase() + reason.slice(1);
  return text.length > 140 ? `${text.slice(0, 139)}…` : text;
}

/**
 * The steps of a reply that did not go through, folded into one line under
 * it: how many of how many, a dot for each kind. Opened, each kind is a row
 * with what the steps were and why, and a kind with several steps opens to
 * list them.
 */
export function DeskStepFailures({ steps }: { steps: readonly ToolStep[] }) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const [selected, setSelected] = useState<StepKind | null>(null);

  const groups = useMemo(() => {
    const byKind = new Map<StepKind, ToolStep[]>();
    for (const step of steps) {
      if (step.status !== "failed") continue;
      const kind: StepKind = toolRefusal(step) ?? "failed";
      byKind.set(kind, [...(byKind.get(kind) ?? []), step]);
    }
    return KIND_ORDER.flatMap((kind) => {
      const list = byKind.get(kind);
      return list ? [{ kind, steps: list }] : [];
    });
  }, [steps]);

  const failedCount = groups.reduce((sum, group) => sum + group.steps.length, 0);
  if (failedCount === 0) {
    return null;
  }

  return (
    <div className={cn("dk-sf", open && "dk-open")}>
      <button
        type="button"
        className="dk-sf-h"
        aria-expanded={open}
        onClick={() => {
          setOpen((value) => !value);
          setSelected(null);
        }}
      >
        <span className="dk-sf-ic">
          <DeskIcon name="alert" size={12} stroke={2.2} />
        </span>
        <span>
          <b>{t("{0} of {1} steps didn't go through", failedCount, steps.length)}</b>
        </span>
        <span className="dk-sf-dots">
          {groups.map((group) => (
            <i key={group.kind} className={KIND_CLASS[group.kind]} />
          ))}
        </span>
        <span className="dk-sf-cv">
          <DeskIcon name="chevR" size={11} stroke={2.2} />
        </span>
      </button>
      {open && (
        <div className="dk-sf-l">
          {groups.map((group) => {
            const phrases = [...new Set(group.steps.map((step) => humanizeToolName(step.name)))];
            const reasons = [...new Set(group.steps.map(stepReason).filter(Boolean))];
            const many = group.steps.length > 1;
            return (
              <div
                key={group.kind}
                className={cn(
                  "dk-sf-r",
                  KIND_CLASS[group.kind],
                  selected === group.kind && "dk-on",
                )}
              >
                <button
                  type="button"
                  className="dk-sf-rh"
                  style={{ cursor: many ? "pointer" : "default" }}
                  onClick={() =>
                    many && setSelected((value) => (value === group.kind ? null : group.kind))
                  }
                >
                  <span className="dk-ec-sic">
                    <DeskIcon name={KIND_ICON[group.kind]} size={10} stroke={2.4} />
                  </span>
                  <b>{kindLabel(group.kind, group.steps.length, t)}</b>
                  <span>{[phrases.join(", "), reasons[0]].filter(Boolean).join(" · ")}</span>
                  {many && (
                    <span className="dk-sf-cv">
                      <DeskIcon name="chevR" size={10} stroke={2.2} />
                    </span>
                  )}
                </button>
                {selected === group.kind && (
                  <div className="dk-sf-items">
                    <DeskBulkList
                      limit={5}
                      noun={t("steps")}
                      items={group.steps.map((step) => ({
                        id: step.name,
                        name: humanizeToolName(step.name),
                        reason: stepReason(step),
                      }))}
                    />
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
