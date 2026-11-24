import { Link } from "@trenova/shared/components/link";
import { recordPath } from "@/config/record-links";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import type { DeskIftaReturnCard } from "@/lib/graphql/ifta-return-desk-card";
import { iftaReturn } from "@/lib/queries/ifta-return";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  CardBar,
  CardBarHint,
  CardCheck,
  CardChecks,
  CardFacts,
  CardHead,
  CardIcon,
  CardSection,
  CardStats,
  CardStatus,
  CardUnreadable,
  RecordCard,
} from "../../record-card/record-card";
import type { RecordArtifactProps } from "../registry";
import {
  netFigure,
  returnChecks,
  returnFacts,
  returnStanding,
  returnStats,
  returnTitle,
  weightiestLines,
} from "./ifta-return-card";
import { DeskIftaReturnSkeleton } from "./ifta-return-skeleton";

export type IftaReturnCardProps = {
  recordId: string;
  onBack?: () => void;
  /** What the way back says; the table, unless the return was opened from another card. */
  backLabel?: string;
};

/**
 * One quarter's IFTA return as a fuel tax clerk reads it, opened from a
 * board's jurisdiction line or looked up by the agent: where it stands and when
 * it is due, what blocks finalizing it and what asks for a look, the quarter's
 * figures, and the jurisdictions that owe the most or earn the largest credit.
 * It reads; finalizing and filing stay on the IFTA returns page, one click away.
 */
export function DeskIftaReturn({ recordId, onBack, backLabel }: IftaReturnCardProps) {
  const t = useT();
  const query = useQuery({ ...iftaReturn.card(recordId), retry: false });
  const card = query.data;

  if (!card) {
    return query.isError ? (
      <CardUnreadable
        kind={t("IFTA return")}
        title={t("This IFTA return could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskIftaReturnSkeleton />
    );
  }

  return (
    <IftaReturnCardView
      key={card.id}
      recordId={recordId}
      card={card}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/** The return as an artifact of its own, when the agent looked it up. */
export function DeskIftaReturnArtifact({ recordId }: RecordArtifactProps) {
  return <DeskIftaReturn recordId={recordId} />;
}

function IftaReturnCardView({
  recordId,
  card,
  onBack,
  backLabel,
}: IftaReturnCardProps & { card: DeskIftaReturnCard }) {
  const t = useT();
  const now = useNowSeconds();
  const standing = returnStanding(card.status);
  const net = netFigure(card, t);
  const checks = returnChecks(card, now, t);
  const lines = weightiestLines(card, t);

  return (
    <RecordCard>
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={[t("IFTA return"), standing.label]}
        title={returnTitle(card, t)}
        figure={
          <>
            <b>{net.amount}</b>
            <span>{net.words}</span>
          </>
        }
        meta={
          <>
            <CardStatus key={card.status} tone={standing.tone}>
              {standing.label}
            </CardStatus>
            <CardFacts>{returnFacts(card, now, t)}</CardFacts>
          </>
        }
      />

      <CardSection title={t("Before it can be filed")}>
        <CardChecks>
          {checks.map((check) => (
            <CardCheck
              key={check.key}
              state={check.state}
              title={check.title}
              detail={check.detail}
            />
          ))}
        </CardChecks>
      </CardSection>

      <CardSection title={t("The quarter")}>
        <CardStats stats={returnStats(card, t)} />
      </CardSection>

      <CardSection title={t("Net tax by jurisdiction")}>
        {lines.rows.length === 0 ? (
          <p className="dk-rec-note">{t("The return has no jurisdiction lines yet.")}</p>
        ) : (
          <div className="dk-rec-list">
            {lines.rows.map((line) => (
              <div key={line.id} className="dk-rec-row">
                <span className="dk-rec-main">
                  <b>{line.name}</b>
                  <span className="dk-rec-sub">{line.sub}</span>
                </span>
                {line.state === "bad" ? (
                  <CardStatus tone="danger">{line.amount}</CardStatus>
                ) : (
                  <span className="font-mono text-sm tabular-nums">{line.amount}</span>
                )}
              </div>
            ))}
          </div>
        )}
        {lines.rest > 0 && (
          <p className="dk-rec-note">
            {t(
              "{0, plural, one {# more line on the return} other {# more lines on the return}}",
              lines.rest,
            )}
          </p>
        )}
      </CardSection>

      <CardBar>
        <CardBarHint>{t("Finalize or file it on the IFTA returns page")}</CardBarHint>
        <Link className="dk-ax-btn dk-ink" to={recordPath("ifta_return", card.id || recordId)}>
          <CardIcon name="ext" size={13} />
          {t("Open IFTA return")}
        </Link>
      </CardBar>
    </RecordCard>
  );
}
