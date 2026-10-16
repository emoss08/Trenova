import { recordPath } from "@/config/record-links";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import type { DeskJournalEntryCard } from "@/lib/graphql/journal-cards";
import {
  approveJournalEntries,
  postJournalEntries,
  type JournalReviewResult,
} from "@/lib/graphql/journal-review";
import { journalEntry } from "@/lib/queries/journal-entry";
import { RequestReversalDialog } from "@/routes/journal-reversal/_components/request-reversal-dialog";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { Link } from "react-router";
import { CardActionButton } from "../../record-card/card-action";
import {
  CardEquation,
  CardJournal,
  JournalRefRow,
  JournalSeal,
  PeriodChip,
} from "../../record-card/card-journal";
import { SettlementSignoffs, SettlementStub } from "../../record-card/card-settlement";
import { CardTrack } from "../../record-card/card-track";
import {
  accountingDay,
  equationNote,
  journalEquation,
  journalTotals,
} from "../../record-card/journal-values";
import {
  CardBar,
  CardBarHint,
  CardFacts,
  CardHead,
  CardIcon,
  CardOpenLink,
  CardSection,
  CardStatus,
  CardStepper,
  CardUnreadable,
  RecordCard,
} from "../../record-card/record-card";
import { minorMoney, neighborsIn } from "../../record-card/record-card-values";
import { useListedRecords, type RecordArtifactProps } from "../registry";
import {
  entryActions,
  entryFigure,
  entryIsOff,
  entryKicker,
  entrySignoffs,
  entryStanding,
  entrySubtitle,
  entryTitle,
  entryTrack,
  entryTrackNote,
  postingBlocker,
  refusedReversal,
  reversalPair,
  reversalSlipTitle,
  reversalWhy,
  sourceRef,
  standingReversal,
  waitingHint,
} from "./journal-entry-card";
import {
  JournalReversalPair,
  JournalReversalSlip,
  JournalReversalWhy,
} from "./journal-entry-parts";
import { DeskJournalEntrySkeleton } from "./journal-entry-skeleton";

/**
 * What the card is given: the journal entry, and when it was opened from a
 * table or from a card that lists it, the rows in the order shown, how to step
 * to another, and the way back.
 */
export type JournalEntryCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the entry was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];

/**
 * A journal entry as the accountant who reviews the general ledger reads it,
 * opened from a table's row or looked up by the agent: what it booked and to
 * which accounts, written into the book with debits first and credits
 * indented; whether it balances; what it does to each side of the accounting
 * equation; where it stands on its way to the ledger and who moved it; the
 * record that made it, which opens in place; and, once it is reversed, the
 * entry that took it back.
 *
 * Its buttons are the review's own, taken by the person and never by the
 * agent, under the permission the server checks: approve an entry waiting on
 * approval, post an approved one, and ask for a posted entry to be reversed.
 */
export function DeskJournalEntry({
  recordId,
  order: rowOrder = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: JournalEntryCardProps) {
  const t = useT();
  const query = useQuery({ ...journalEntry.card(recordId), retry: false });
  const card = query.data;

  if (!card) {
    return query.isError ? (
      <CardUnreadable
        kind={t("Journal entry")}
        title={t("This journal entry could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskJournalEntrySkeleton stepper={onSelect !== undefined && rowOrder.length > 1} />
    );
  }

  return (
    <JournalEntryCard
      key={card.entry.id}
      card={card}
      rowOrder={rowOrder}
      onSelect={onSelect}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/**
 * The entry as an artifact of its own, when the agent looked it up: the same
 * card, with nothing to step through and nowhere to go back to.
 */
export function DeskJournalEntryArtifact({ recordId }: RecordArtifactProps) {
  return <DeskJournalEntry recordId={recordId} />;
}

/** The review's one-entry result as the card reads it: changed, or refused with the server's words. */
function reviewed(result: JournalReviewResult, refused: string): JournalReviewResult {
  const outcome = result.outcomes[0];
  if (!outcome?.changed) {
    throw new Error(outcome?.error || refused);
  }
  return result;
}

function JournalEntryCard({
  card,
  rowOrder,
  onSelect,
  onBack,
  backLabel,
}: Omit<JournalEntryCardProps, "order" | "recordId"> & {
  card: DeskJournalEntryCard;
  rowOrder: readonly string[];
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: mayReview } = usePermission(Resource.JournalEntry, Operation.Approve);
  const { allowed: mayReverse } = usePermission(Resource.JournalReversal, Operation.Create);
  const [reversing, setReversing] = useState(false);
  const { entry } = card;
  const id = entry.id;

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: journalEntry._def });
    void queryClient.invalidateQueries({ queryKey: ["journal-reversal-list"] });
  };
  const approve = useApiMutation<JournalReviewResult, undefined>({
    mutationFn: async () =>
      reviewed(await approveJournalEntries([id]), t("The entry could not be approved")),
    onSuccess: () => {
      invalidate();
    },
    resourceName: "Journal Entry",
  });
  const post = useApiMutation<JournalReviewResult, undefined>({
    mutationFn: async () =>
      reviewed(await postJournalEntries([id]), t("The entry could not be posted")),
    onSuccess: () => {
      invalidate();
    },
    resourceName: "Journal Entry",
  });

  const source = card.source ? sourceRef(card.source, t) : null;
  const backToEntry = t("Back to {0}", entry.entryNumber);
  const listedSource = useListedRecords({
    entity: source?.entity ?? "journal_entry",
    order: source?.entity ? [source.id] : NO_ROWS,
    backLabel: backToEntry,
  });
  const pair = reversalPair(card);
  const other = pair?.find((row) => !row.mine);
  const listedPair = useListedRecords({
    entity: "journal_entry",
    order: other ? [other.entry.id] : NO_ROWS,
    backLabel: backToEntry,
  });
  const opened = listedSource.opened ?? listedPair.opened;
  if (opened) {
    return opened;
  }

  const standing = entryStanding(entry, t);
  const figure = entryFigure(entry, t);
  const totals = journalTotals(card.lines);
  const equation = journalEquation(card.lines);
  const why = reversalWhy(card);
  const pending = standingReversal(card);
  const refused = refusedReversal(card);
  const actions = entryActions(card);
  const blocker = postingBlocker(card.posting);
  const hint = waitingHint(card, t);
  const busy = approve.isPending || post.isPending;
  const leading = mayReview ? (actions.approve ? "approve" : actions.post ? "post" : null) : null;
  const signoffs = entrySignoffs(card, t);
  const amount = minorMoney(entry.totalDebit);
  const sourceLink = source?.entity ? recordPath(source.entity, source.id) : null;

  return (
    <RecordCard className="dk-jen">
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={entryKicker(card, t)}
        stepper={
          onSelect &&
          rowOrder.length > 1 && (
            <CardStepper
              neighbors={neighborsIn(rowOrder, id)}
              onStep={onSelect}
              previousLabel={t("Previous entry")}
              nextLabel={t("Next entry")}
            />
          )
        }
        lead={<SettlementStub mark={t("JE")} />}
        title={entryTitle(entry, t)}
        subtitle={entrySubtitle(card, t)}
        figure={
          <>
            <b className={figure.struck ? "dk-struck" : undefined}>
              {figure.struck ? <s>{amount}</s> : amount}
            </b>
            <span>{figure.caption}</span>
          </>
        }
        meta={
          <>
            <CardStatus key={standing.label} tone={standing.tone} mark={standing.mark}>
              {standing.label}
            </CardStatus>
            <CardFacts>
              {[
                t("Accounting date {0}", accountingDay(entry.accountingDate)),
                card.period ? <PeriodChip key="period" period={card.period} /> : null,
              ]}
            </CardFacts>
          </>
        }
      />

      {(pair || pending || refused) && (
        <CardSection
          title={t("Its reversal")}
          aside={pair ? <span className="dk-ok">{t("Nets to zero")}</span> : undefined}
        >
          {pair && <JournalReversalPair rows={pair} onOpen={listedPair.open} />}
          {pair && why && <JournalReversalWhy code={why.code} text={why.text} />}
          {pending && (
            <JournalReversalSlip
              reversal={pending}
              title={reversalSlipTitle(pending, card.reversalRequestedBy?.name ?? null, t)}
              refused={false}
            />
          )}
          {refused && (
            <JournalReversalSlip
              reversal={refused}
              title={reversalSlipTitle(refused, card.reversalRequestedBy?.name ?? null, t)}
              refused
            />
          )}
        </CardSection>
      )}

      <CardSection
        title={entryIsOff(entry) ? t("Where it stood") : t("Where it stands")}
        aside={
          source && sourceLink ? (
            <CardOpenLink
              to={sourceLink}
              onOpen={listedSource.open && (() => listedSource.open?.(source.id))}
            >
              {t("Open {0}", source.number)}
            </CardOpenLink>
          ) : undefined
        }
      >
        <CardTrack
          steps={entryTrack(entry, t)}
          label={t("Journal entry lifecycle")}
          off={entryIsOff(entry)}
          note={entryTrackNote(card, t)}
        />
        <SettlementSignoffs rows={signoffs} />
      </CardSection>

      <CardSection title={t("The journal")} aside={<JournalSeal totals={totals} />}>
        <CardJournal lines={card.lines} totals={totals} label={t("The entry's lines")} />
      </CardSection>

      {card.lines.length > 0 && (
        <CardSection title={t("What it does to the books")}>
          <CardEquation equation={equation} note={equationNote(totals, equation, t)} />
        </CardSection>
      )}

      {source && (
        <CardSection title={t("Where it came from")}>
          <JournalRefRow
            mark={source.mark}
            title={source.number}
            sub={source.kind}
            to={sourceLink}
            onOpen={listedSource.open && (() => listedSource.open?.(source.id))}
          />
          <p className="dk-rec-jref-note">
            {entry.isAutoGenerated
              ? t("Trenova wrote this entry from its source when the source posted.")
              : t("Someone recorded this entry by hand.")}
          </p>
        </CardSection>
      )}

      <CardBar>
        {mayReverse && actions.reverse && (
          <CardActionButton
            variant="quiet"
            className="dk-rec-step"
            blocker={null}
            busy={busy}
            onClick={() => setReversing(true)}
          >
            <CardIcon name="undo" size={12} stroke={2.2} />
            {t("Request reversal")}
          </CardActionButton>
        )}
        <CardBarHint>
          {hint ??
            (entry.status === "Reversed"
              ? t("A reversed entry stays in the ledger as it was posted")
              : actions.reverse
                ? t("Reversing posts a mirror entry after someone approves it")
                : t("Every line and its source are on the entry's page"))}
        </CardBarHint>
        <Link
          className={leading === null ? "dk-ax-btn dk-ink" : "dk-ax-btn"}
          to={recordPath("journal_entry", id)}
        >
          <CardIcon name="ext" size={13} />
          {t("Open entry")}
        </Link>
        {leading === "approve" && (
          <CardActionButton blocker={null} busy={busy} onClick={() => approve.mutate(undefined)}>
            {t("Approve")}
          </CardActionButton>
        )}
        {leading === "post" && (
          <CardActionButton blocker={blocker} busy={busy} onClick={() => post.mutate(undefined)}>
            {card.posting?.period ? t("Post to {0}", card.posting.period.name) : t("Post")}
          </CardActionButton>
        )}
      </CardBar>

      {mayReverse && actions.reverse && (
        <RequestReversalDialog
          entryId={id}
          entryNumber={entry.entryNumber}
          open={reversing}
          onOpenChange={setReversing}
          onDone={invalidate}
        />
      )}
    </RecordCard>
  );
}
