import { ReasonDialog } from "@/components/reason-dialog";
import { recordPath } from "@/config/record-links";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import type { DeskManualJournalCard } from "@/lib/graphql/journal-cards";
import { journalEntry } from "@/lib/queries/journal-entry";
import { manualJournal } from "@/lib/queries/manual-journal";
import { apiService } from "@/services/api";
import type { ManualJournal } from "@/types/manual-journal";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useMemo, useState } from "react";
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
  CardSection,
  CardStatus,
  CardStepper,
  CardUnreadable,
  RecordCard,
} from "../../record-card/record-card";
import { cardDay, minorMoney, neighborsIn } from "../../record-card/record-card-values";
import { useListedRecords, type RecordArtifactProps } from "../registry";
import {
  journalActions,
  journalFigure,
  journalIsOff,
  journalKicker,
  journalSettled,
  journalSignoffs,
  journalStanding,
  journalSubtitle,
  journalTrack,
  journalTrackNote,
  postingPlace,
  submitBlocker,
} from "./manual-journal-card";
import { JournalPostingPlace, JournalSettledSlip, JournalWhy } from "./manual-journal-parts";
import { DeskManualJournalSkeleton } from "./manual-journal-skeleton";

/**
 * What the card is given: the manual journal, and when it was opened from a
 * table or from a card that lists it, the rows in the order shown, how to step
 * to another, and the way back.
 */
export type ManualJournalCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the journal was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];

type ReasonStep = "reject" | "cancel";

/**
 * A manual journal as the controller who approves it reads it, opened from a
 * table's row or looked up by the agent: what it books and why, in the
 * requester's own words; whether its lines balance, and by how much they are
 * off while they do not; what it does to each side of the accounting
 * equation; where it stands on its way to the ledger and who signed each
 * step; where it will post, moved to the first open day when its period is
 * shut; and, once posted, the journal entry it became, which opens in place.
 *
 * Its buttons are the request's own steps, taken by the person and never by
 * the agent, under the permission the server checks for each: submit a
 * balanced draft, approve and post a submitted one in one step, post one
 * approved already, reject one waiting with a reason, and cancel one not yet
 * posted with a reason. Editing the lines stays on its page.
 */
export function DeskManualJournal({
  recordId,
  order: rowOrder = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: ManualJournalCardProps) {
  const t = useT();
  const query = useQuery({ ...manualJournal.card(recordId), retry: false });
  const card = query.data;

  if (!card) {
    return query.isError ? (
      <CardUnreadable
        kind={t("Manual journal")}
        title={t("This manual journal could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskManualJournalSkeleton stepper={onSelect !== undefined && rowOrder.length > 1} />
    );
  }

  return (
    <ManualJournalCard
      key={card.journal.id}
      card={card}
      rowOrder={rowOrder}
      onSelect={onSelect}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/**
 * The journal as an artifact of its own, when the agent looked it up: the
 * same card, with nothing to step through and nowhere to go back to.
 */
export function DeskManualJournalArtifact({ recordId }: RecordArtifactProps) {
  return <DeskManualJournal recordId={recordId} />;
}

function ManualJournalCard({
  card,
  rowOrder,
  onSelect,
  onBack,
  backLabel,
}: Omit<ManualJournalCardProps, "order" | "recordId"> & {
  card: DeskManualJournalCard;
  rowOrder: readonly string[];
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: maySubmit } = usePermission(Resource.ManualJournal, Operation.Submit);
  const { allowed: mayApprove } = usePermission(Resource.ManualJournal, Operation.Approve);
  const { allowed: mayReject } = usePermission(Resource.ManualJournal, Operation.Reject);
  const { allowed: mayCancel } = usePermission(Resource.ManualJournal, Operation.Cancel);
  const [asking, setAsking] = useState<ReasonStep | null>(null);
  const { journal } = card;
  const id = journal.id;
  const number = journal.requestNumber;
  const currency = journal.currencyCode;

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: manualJournal._def });
    void queryClient.invalidateQueries({ queryKey: ["manual-journal-list"] });
    void queryClient.invalidateQueries({ queryKey: journalEntry._def });
  };
  const submit = useApiMutation<ManualJournal, undefined>({
    mutationFn: () => apiService.manualJournalService.submit(id),
    onSuccess: () => {
      invalidate();
    },
    resourceName: "Manual Journal",
  });
  const approveAndPost = useApiMutation<ManualJournal, undefined>({
    mutationFn: async () => {
      await apiService.manualJournalService.approve(id);
      try {
        return await apiService.manualJournalService.post(id);
      } catch (error) {
        invalidate();
        throw error;
      }
    },
    onSuccess: () => {
      invalidate();
    },
    resourceName: "Manual Journal",
  });
  const post = useApiMutation<ManualJournal, undefined>({
    mutationFn: () => apiService.manualJournalService.post(id),
    onSuccess: () => {
      invalidate();
    },
    resourceName: "Manual Journal",
  });
  const reject = useApiMutation<ManualJournal, string>({
    mutationFn: (reason) => apiService.manualJournalService.reject(id, reason),
    onSuccess: () => {
      setAsking(null);
      invalidate();
    },
    resourceName: "Manual Journal",
  });
  const cancel = useApiMutation<ManualJournal, string>({
    mutationFn: (reason) => apiService.manualJournalService.cancel(id, reason),
    onSuccess: () => {
      setAsking(null);
      invalidate();
    },
    resourceName: "Manual Journal",
  });

  const entryIds = useMemo(
    () => (card.entries.length > 0 ? card.entries.map((entry) => entry.id) : NO_ROWS),
    [card.entries],
  );
  const listedEntries = useListedRecords({
    entity: "journal_entry",
    order: entryIds,
    backLabel: t("Back to {0}", number),
  });
  if (listedEntries.opened) {
    return listedEntries.opened;
  }

  const standing = journalStanding(journal, t);
  const totals = journalTotals(card.lines);
  const figure = journalFigure(card, totals, t);
  const equation = journalEquation(card.lines);
  const place = postingPlace(card);
  const settled = journalSettled(card, t);
  const actions = journalActions(journal);
  const refusal = card.posting?.refusal ? card.posting.refusal : null;
  const busy =
    submit.isPending ||
    approveAndPost.isPending ||
    post.isPending ||
    reject.isPending ||
    cancel.isPending;
  const leading =
    actions.submit && maySubmit
      ? "submit"
      : actions.approveAndPost && mayApprove
        ? "approveAndPost"
        : actions.post && mayApprove
          ? "post"
          : null;
  const offTrack = journalIsOff(journal);
  const reason = journal.reason?.trim() ?? "";

  return (
    <RecordCard className="dk-mj">
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={journalKicker(card, t)}
        stepper={
          onSelect &&
          rowOrder.length > 1 && (
            <CardStepper
              neighbors={neighborsIn(rowOrder, id)}
              onStep={onSelect}
              previousLabel={t("Previous journal")}
              nextLabel={t("Next journal")}
            />
          )
        }
        lead={<SettlementStub mark={t("MJ")} />}
        title={journal.description}
        subtitle={journalSubtitle(card, t)}
        figure={
          <>
            <b className={cn(figure.warn && "dk-warn", figure.struck && "dk-struck")}>
              {figure.struck ? <s>{figure.amount}</s> : figure.amount}
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
                t("Accounting date {0}", accountingDay(journal.accountingDate)),
                card.requestedPeriod ? (
                  <PeriodChip key="period" period={card.requestedPeriod} />
                ) : null,
              ]}
            </CardFacts>
          </>
        }
      />

      {settled && (
        <CardSection title={settled.title}>
          <JournalSettledSlip reason={settled.reason} by={settled.by} />
        </CardSection>
      )}

      <CardSection title={offTrack ? t("Where it stood") : t("Where it stands")}>
        <CardTrack
          steps={journalTrack(journal, t)}
          label={t("Manual journal lifecycle")}
          off={offTrack}
          note={journalTrackNote(card, t)}
        />
        <SettlementSignoffs rows={journalSignoffs(card, t)} />
      </CardSection>

      {card.entries.length > 0 && (
        <CardSection title={t("What it became")}>
          <div className="dk-rec-jrefs">
            {card.entries.map((entry) => (
              <JournalRefRow
                key={entry.id}
                mark={t("JE")}
                title={entry.entryNumber}
                sub={
                  entry.postedAt
                    ? t(
                        "Journal entry · posted {0} · {1}",
                        cardDay(entry.postedAt),
                        minorMoney(entry.totalDebit, currency),
                      )
                    : t("Journal entry · {0}", minorMoney(entry.totalDebit, currency))
                }
                to={recordPath("journal_entry", entry.id)}
                onOpen={listedEntries.open && (() => listedEntries.open?.(entry.id))}
              />
            ))}
          </div>
        </CardSection>
      )}

      {reason && (
        <CardSection title={t("Why")}>
          <JournalWhy text={reason} by={card.createdBy?.name ?? null} />
        </CardSection>
      )}

      <CardSection
        title={t("The journal")}
        aside={<JournalSeal totals={totals} currency={currency} />}
      >
        <CardJournal lines={card.lines} totals={totals} label={t("The journal's lines")} />
      </CardSection>

      {card.lines.length > 0 && (
        <CardSection title={t("What it does to the books")}>
          <CardEquation
            equation={equation}
            note={equationNote(totals, equation, t, currency)}
            currency={currency}
          />
        </CardSection>
      )}

      {place && (
        <CardSection title={t("Where it will post")}>
          <JournalPostingPlace place={place} asked={journal.accountingDate} />
        </CardSection>
      )}

      <CardBar>
        {actions.reject && mayReject && (
          <CardActionButton
            variant="quiet"
            className="dk-rec-step"
            blocker={null}
            busy={busy}
            onClick={() => setAsking("reject")}
          >
            <CardIcon name="x" size={12} stroke={2.4} />
            {t("Reject")}
          </CardActionButton>
        )}
        {actions.cancel && mayCancel && (
          <CardActionButton
            variant="quiet"
            className="dk-rec-step"
            blocker={null}
            busy={busy}
            onClick={() => setAsking("cancel")}
          >
            {t("Cancel")}
          </CardActionButton>
        )}
        <CardBarHint>
          {journal.status === "Draft"
            ? totals.balanced
              ? t("Edit the lines on its page")
              : t("Balance the lines on its page")
            : journal.status === "Posted"
              ? t("Reverse it from its journal entry")
              : t("Edit lines on its page")}
        </CardBarHint>
        <Link
          className={leading === null ? "dk-ax-btn dk-ink" : "dk-ax-btn"}
          to={recordPath("manual_journal", id)}
        >
          <CardIcon name="ext" size={13} />
          {t("Open journal")}
        </Link>
        {leading === "submit" && (
          <CardActionButton
            blocker={submitBlocker(card.lines.length, totals, t)}
            busy={busy}
            onClick={() => submit.mutate(undefined)}
          >
            {t("Submit for approval")}
          </CardActionButton>
        )}
        {leading === "approveAndPost" && (
          <CardActionButton
            blocker={refusal}
            busy={busy}
            onClick={() => approveAndPost.mutate(undefined)}
          >
            {t("Approve and post")}
          </CardActionButton>
        )}
        {leading === "post" && (
          <CardActionButton blocker={refusal} busy={busy} onClick={() => post.mutate(undefined)}>
            {t("Post")}
          </CardActionButton>
        )}
      </CardBar>

      <ReasonDialog
        open={asking === "reject"}
        title={t("Reject {0}", number)}
        description={t("The requester sees why. Nothing it books reaches the general ledger.")}
        placeholder={t("What needs to change")}
        confirmLabel={t("Reject")}
        destructive
        pending={reject.isPending}
        onConfirm={(text) => reject.mutate(text)}
        onOpenChange={(open) => setAsking(open ? "reject" : null)}
      />
      <ReasonDialog
        open={asking === "cancel"}
        title={t("Cancel {0}", number)}
        description={t(
          "A cancelled journal can't be submitted again; nothing it books reaches the general ledger.",
        )}
        placeholder={t("Why it is no longer needed")}
        confirmLabel={t("Cancel journal")}
        destructive
        pending={cancel.isPending}
        onConfirm={(text) => cancel.mutate(text)}
        onOpenChange={(open) => setAsking(open ? "cancel" : null)}
      />
    </RecordCard>
  );
}
