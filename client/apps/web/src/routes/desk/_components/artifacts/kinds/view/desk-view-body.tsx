import type { AssistantArtifact } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo } from "react";
import { Link } from "react-router";
import { composedViewFrom, navigationFrom } from "../../artifact-payloads";
import { ArtIcon } from "../../desk-art-kinds";
import { Cell } from "../table/desk-table-body";
import { useRichT } from "@trenova/shared/i18n/rich";
import { Notice, entityName } from "../artifact-body-parts";

/** A view described in words and opened live: what it filters, what it left out, and the way in. */
export function DeskViewBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const rt = useRichT();
  const view = useMemo(() => composedViewFrom(artifact), [artifact]);
  const navigation = useMemo(() => navigationFrom(artifact), [artifact]);
  if (view) {
    const parts = splitTerms(view.explanation, view.terms);
    return (
      <div className="dk-ax-pad dk-ax-view">
        <p className="dk-ax-vx">
          {parts.map((part, index) =>
            part.term ? (
              <span key={index} className="dk-ax-term">
                {part.text}
              </span>
            ) : (
              part.text
            ),
          )}
        </p>
        <div className="dk-ax-vprev">
          <div className="dk-ax-vbar">
            <span>{entityName(view.entity, t)}</span>
            <em>{t("{0, plural, one {# filter} other {# filters}}", view.filterCount)}</em>
            {view.count !== null && (
              <b>
                {view.countCapped
                  ? t("{0}+ results", view.count)
                  : t("{0, plural, one {# result} other {# results}}", view.count)}
              </b>
            )}
          </div>
          {view.preview.map((row) => (
            <div key={row.key} className="dk-ax-vrow">
              {view.columns.map((column, index) => (
                <span key={column.key}>
                  <Cell column={column} value={row.values[column.key]} first={index === 0} />
                </span>
              ))}
            </div>
          ))}
        </div>
        {view.unresolved.map((entry) => (
          <div key={entry.phrase} className="dk-ax-warn">
            <ArtIcon name="warn" size={13} />
            <span>
              {rt("Left out <b>“{0}”</b>.", { b: (c) => <b>{c}</b> }, entry.phrase)}{" "}
              {asSentence(entry.reason)}
            </span>
          </div>
        ))}
        <Link className="dk-ax-btn dk-ink dk-wide" to={view.path}>
          <ArtIcon name="ext" size={13} />
          {t("Open in {0}", entityName(view.entity, t))}
        </Link>
      </div>
    );
  }
  if (navigation) {
    return (
      <div className="dk-ax-pad dk-ax-view">
        <p className="dk-ax-vx">{navigation.location || navigation.name}</p>
        <Link className="dk-ax-btn dk-ink dk-wide" to={navigation.path}>
          <ArtIcon name="ext" size={13} />
          {t("Open {0}", navigation.name)}
        </Link>
      </div>
    );
  }
  return <Notice>{t("This view no longer names a page.")}</Notice>;
}

/** A reason as the warning reads it: one sentence, closed with a full stop. */
export function asSentence(text: string): string {
  const trimmed = text.trim();
  return trimmed === "" || /[.!?]$/.test(trimmed) ? trimmed : `${trimmed}.`;
}

function splitTerms(
  text: string,
  terms: readonly string[],
): Array<{ text: string; term: boolean }> {
  if (terms.length === 0) {
    return [{ text, term: false }];
  }
  const escaped = terms.map((term) => term.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"));
  const pattern = new RegExp(`(${escaped.join("|")})`, "g");
  return text
    .split(pattern)
    .filter((part) => part !== "")
    .map((part) => ({ text: part, term: terms.includes(part) }));
}
