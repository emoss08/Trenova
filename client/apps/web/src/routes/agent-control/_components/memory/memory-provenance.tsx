import { recordPath } from "@/config/record-links";
import type { AgentMemoryRow } from "@/lib/graphql/agent-memories";
import type { AgentMemoryStatus } from "@trenova/graphql/generated/graphql";
import { Badge } from "@trenova/shared/components/ui/badge";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { Link } from "react-router";
import {
  MemoryStatusBadge,
  OutsideContentBadge,
  memorySourceLabel,
} from "../activity/agent-badges";
import { reflectionSignalLabel } from "./reflection-signals";

type LinkedMemory = { id: string; content: string; status: AgentMemoryStatus };

/**
 * Where a memory came from, beside the form that edits it: who or what
 * recorded it and when, the conversation or run it was drawn from, why it was
 * kept and what was said, and the memory it replaced or that replaced it. An
 * agent keeps lessons on its own, so a person reviewing one needs all of it
 * to judge whether it should stay.
 */
export function MemoryProvenance({ memory }: { memory: AgentMemoryRow | null | undefined }) {
  const t = useT();
  if (!memory) {
    return null;
  }

  const evidence = memory.evidence;
  const signals = evidence?.signals ?? [];
  const quotes = evidence?.quotes ?? [];
  const ratings = evidence?.ratingCount ?? 0;

  return (
    <DescriptionList columns={2}>
      <DescriptionItem label={t("Recorded by")}>
        {memorySourceLabel(memory.source, t)}
      </DescriptionItem>
      <DescriptionItem label={t("Recorded")}>
        {formatUnixDateTimeMedium(memory.createdAt)}
      </DescriptionItem>

      {memory.sourceThreadId || memory.sourceRunId ? (
        <DescriptionItem label={t("Drawn from")} span="full">
          <span className="flex flex-wrap items-center gap-3">
            {memory.sourceThreadId ? (
              <Link
                to={recordPath("assistant_thread", memory.sourceThreadId)}
                className="text-brand ui-focus-ring rounded-sm hover:underline"
              >
                {t("Open the conversation")}
              </Link>
            ) : null}
            {memory.sourceRunId ? (
              <Link
                to={recordPath("agent_run", memory.sourceRunId)}
                className="text-brand ui-focus-ring rounded-sm hover:underline"
              >
                {t("Open the run")}
              </Link>
            ) : null}
          </span>
        </DescriptionItem>
      ) : null}

      {memory.tainted ? (
        <DescriptionItem label={t("Outside content")} span="full">
          <OutsideContentBadge
            t={t}
            title={t(
              "Written by a run that had read text from outside the organization; an agent that reads it is treated as having read that text too.",
            )}
          />
        </DescriptionItem>
      ) : null}

      {evidence?.reason ? (
        <DescriptionItem label={t("Why it was kept")} span="full">
          {evidence.reason}
        </DescriptionItem>
      ) : null}

      {signals.length > 0 ? (
        <DescriptionItem label={t("What made the agent look back")} span="full">
          <ul aria-label={t("What made the agent look back")} className="flex flex-wrap gap-1">
            {signals.map((signal) => (
              <li key={signal}>
                <Badge variant="neutral" appearance="outline">
                  {reflectionSignalLabel(signal, t)}
                </Badge>
              </li>
            ))}
          </ul>
        </DescriptionItem>
      ) : null}

      {ratings > 0 ? (
        <DescriptionItem label={t("Ratings")} span="full">
          {t("{0, plural, one {# rating} other {# ratings}}", ratings)} ·{" "}
          {t("{0, plural, one {# person} other {# people}}", evidence?.distinctUsers ?? 0)}
        </DescriptionItem>
      ) : null}

      {quotes.length > 0 ? (
        <DescriptionItem label={t("What was said")} span="full">
          <ul aria-label={t("What was said")} className="flex flex-col gap-1">
            {quotes.map((quote) => (
              <li
                key={quote}
                className="border-border-subtle text-foreground-muted border-l-2 pl-2 text-xs italic"
              >
                <q>{quote}</q>
              </li>
            ))}
          </ul>
        </DescriptionItem>
      ) : null}

      {memory.supersedes ? (
        <DescriptionItem label={t("Replaces")} span="full">
          <LinkedMemoryLine memory={memory.supersedes} />
        </DescriptionItem>
      ) : null}

      {memory.replacedBy ? (
        <DescriptionItem label={t("Replaced by")} span="full">
          <LinkedMemoryLine memory={memory.replacedBy} />
        </DescriptionItem>
      ) : null}
    </DescriptionList>
  );
}

function LinkedMemoryLine({ memory }: { memory: LinkedMemory }) {
  const t = useT();

  return (
    <span className="flex items-start gap-2">
      <MemoryStatusBadge value={memory.status} t={t} />
      <span className="text-sm leading-relaxed">{memory.content}</span>
    </span>
  );
}
