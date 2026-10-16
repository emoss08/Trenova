import type { WatchtowerItem } from "@/lib/graphql/watchtower";
import type { AssistantPageContext } from "@/types/assistant";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { limitUtf8Bytes } from "@trenova/shared/lib/utils";
import { AgentSubjectRecordKind } from "@trenova/shared/types/generated/record-kinds";
import { formatFactTime } from "./lanes";

/** The bounds the server holds a page's title and figures to, in UTF-8 bytes (agent.PageContext). */
const MAX_KPIS = 12;
const MAX_KPI_LABEL = 80;
const MAX_KPI_VALUE = 60;
const MAX_TITLE = 200;

/**
 * The Watchtower item as the page a conversation is about: the record it is
 * about as the page's record, the item itself as the one row selected on the
 * watchtower, and its deadline and facts as the figures the person was
 * looking at. An agent asked "what should I do about this?" starts from
 * exactly what the card showed.
 */
export function watchtowerPageContext(
  t: TranslateFn,
  item: WatchtowerItem,
  now: number,
  timezone: string,
): AssistantPageContext {
  const kind =
    item.subjectType !== null && Object.hasOwn(AgentSubjectRecordKind, item.subjectType)
      ? AgentSubjectRecordKind[item.subjectType]
      : null;
  const withRecord = kind !== null && item.subjectId !== null;

  const figures = [
    ...(item.dueAt !== null
      ? [{ label: item.dueLabel ?? t("Due"), value: formatFactTime(t, item.dueAt, now, timezone) }]
      : []),
    ...item.facts.map((fact) => ({
      label: fact.label,
      value: fact.value ?? (fact.at !== null ? formatFactTime(t, fact.at, now, timezone) : ""),
    })),
    ...item.impact.map((stake) => ({ label: t("At stake"), value: stake })),
  ];

  return {
    path: `/desk/watchtower?item=${encodeURIComponent(item.id)}`,
    entityType: withRecord ? kind : "",
    entityId: withRecord ? (item.subjectId ?? "") : "",
    title: limitUtf8Bytes(item.title, MAX_TITLE),
    view: {
      resource: "watchtower",
      selection: { count: 1, ids: [item.id] },
      kpis: figures
        .filter((figure) => figure.value !== "")
        .slice(0, MAX_KPIS)
        .map((figure) => ({
          label: limitUtf8Bytes(figure.label, MAX_KPI_LABEL),
          value: limitUtf8Bytes(figure.value, MAX_KPI_VALUE),
        })),
    },
    draft: null,
  };
}
