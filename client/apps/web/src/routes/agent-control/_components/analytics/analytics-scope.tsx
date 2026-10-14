import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { useT } from "@trenova/shared/i18n/use-t";
import type {
  AiAnalyticsCompareBy,
  AiAnalyticsWindow,
  AiTask,
} from "@trenova/graphql/generated/graphql";
import { Seg } from "../kit/layout";
import { WINDOW_OPTIONS } from "./analytics-model";
import { LabeledSelect } from "./labeled-select";

const EVERY = "__every__";
const HIDDEN_TASKS: readonly string[] = ["Embedding", "EvaluationJudge"];

type AnalyticsScopeProps = {
  range: AiAnalyticsWindow;
  onRange: (range: AiAnalyticsWindow) => void;
  onPrefetch: (range: AiAnalyticsWindow) => void;
  compareBy: AiAnalyticsCompareBy;
  onCompareBy: (compareBy: AiAnalyticsCompareBy) => void;
  agentId: string | null;
  onAgent: (agentId: string | null) => void;
  agents: readonly Pick<AgentDefinitionRow, "id" | "name">[];
  task: AiTask | null;
  onTask: (task: AiTask | null) => void;
  taskLabels: ReadonlyMap<string, string>;
};

/** What the page measures: the window, what is compared, and the agent and task it is narrowed to. */
export function AnalyticsScope({
  range,
  onRange,
  onPrefetch,
  compareBy,
  onCompareBy,
  agentId,
  onAgent,
  agents,
  task,
  onTask,
  taskLabels,
}: AnalyticsScopeProps) {
  const t = useT();
  const agentItems = [
    { value: EVERY, label: t("Every agent") },
    ...agents.map((agent) => ({ value: agent.id, label: agent.name })),
  ];
  const taskItems = [
    { value: EVERY, label: t("Every task") },
    ...[...taskLabels.entries()]
      .filter(([value]) => !HIDDEN_TASKS.includes(value))
      .map(([value, label]) => ({ value, label: t(label) })),
  ];

  return (
    <div className="an-scope">
      <Seg
        v={range}
        opts={WINDOW_OPTIONS.map((option) => [option.value, t(option.label)] as const)}
        onChange={onRange}
        onHover={(value) => value !== range && onPrefetch(value)}
        label={t("Window")}
      />
      <span className="an-by">
        <span>{t("Compare by")}</span>
        <Seg
          v={compareBy}
          opts={[
            ["Model", t("Model")],
            ["Provider", t("Provider")],
          ]}
          onChange={onCompareBy}
          label={t("Compare by")}
        />
      </span>
      <span className="sp" />
      {agents.length > 0 && (
        <LabeledSelect
          label={t("Agent")}
          items={agentItems}
          value={agentId ?? EVERY}
          active={agentId !== null}
          onChange={(value) => onAgent(value === EVERY ? null : value)}
        />
      )}
      <LabeledSelect
        label={t("Task")}
        items={taskItems}
        value={task ?? EVERY}
        active={task !== null}
        onChange={(value) => onTask(value === EVERY ? null : (value as AiTask))}
      />
      {(agentId !== null || task !== null) && (
        <button
          type="button"
          className="lnk ui-focus-ring"
          onClick={() => {
            onAgent(null);
            onTask(null);
          }}
        >
          {t("Clear")}
        </button>
      )}
    </div>
  );
}
