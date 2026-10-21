import { recordPath } from "@/config/record-links";
import { documentKindName } from "@/lib/document-kinds";
import { assistant } from "@/lib/queries/assistant";
import { useDeskStore } from "@/stores/desk-store";
import type { AssistantArtifact } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { useReducedMotion } from "@trenova/shared/hooks/use-reduced-motion";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useRef, useState } from "react";
import { Link } from "react-router";
import { SettlementStub } from "../../record-card/card-settlement";
import {
  CardFacts,
  CardHead,
  CardSection,
  CardStatus,
  RecordCard,
} from "../../record-card/record-card";
import { ExtractionActions } from "./extraction-actions";
import {
  checkCount,
  checkNames,
  checkQuestion,
  checkSummary,
  createQuestion,
} from "./extraction-asks";
import { ExtractionCheck } from "./extraction-check";
import { ExtractionFileDialog } from "./extraction-file-dialog";
import { groupExtraction, reviewTally, rowsInOrder, type FieldRow } from "./extraction-groups";
import { ExtractionLoad } from "./extraction-load";
import { extractionFrom, isImageFile } from "./extraction-payload";
import {
  documentMark,
  headFacts,
  headFigure,
  headStanding,
  headSubtitle,
  laneOf,
} from "./extraction-summary";
import { ExtractionViewer, type PageMark } from "./extraction-viewer";
import { stopTitle } from "./extraction-words";
import { isChecking, shipmentProposal, shipmentStage } from "./shipment-stage";

type BodyProps = {
  artifact: AssistantArtifact;
  /** Opens another artifact of the conversation, such as the approval of the drafted shipment. */
  onOpenArtifact?: (id: string) => void;
};

/** How sure the classifier must be of the kind before the head stops saying how sure it was. */
const SURE_OF_KIND = 0.9;

/**
 * What document intelligence read off a file, as the record it is about to become. The head
 * names the load by its lane and leads with the rate exactly as read; the readings a person
 * must settle come first, each beside the bit of page it came from; then the load as the
 * reading has it, route first; then the document itself, its pages with a mark over every
 * value read. Choosing a value lights its mark, turns to its page and scrolls it into view;
 * choosing a mark chooses its value. On a wide screen the document stands beside the reading
 * and stays in view while the reading scrolls. The shipment is drafted by the agent and waits
 * for approval, as every write does; the card only asks.
 */
export function DeskExtractBody({ artifact, onOpenArtifact }: BodyProps) {
  const t = useT();
  const reduced = useReducedMotion();
  const extraction = extractionFrom(artifact.payload);
  const { documentId, fileName, pageCount } = extraction;
  const threadId = artifact.threadId;
  const groups = groupExtraction(extraction);
  const rows = rowsInOrder(groups);
  const tally = reviewTally(groups);
  const names = checkNames(tally, t);
  const multiPage = pageCount > 1;
  const docRef = useRef<HTMLElement>(null);

  const [selected, setSelected] = useState<string | null>(null);
  const [listHover, setListHover] = useState<string | null>(null);
  const [boxHover, setBoxHover] = useState<string | null>(null);
  const [page, setPage] = useState(() => {
    const first =
      extraction.fields.find((field) => field.needsLook && field.box !== null) ??
      extraction.fields.find((field) => field.box !== null);
    return first ? first.page : 1;
  });
  const [fileOpen, setFileOpen] = useState(false);

  const askAgent = useDeskStore((state) => state.askAgent);
  const noteDocumentAsk = useDeskStore((state) => state.noteDocumentAsk);
  const askWaiting = useDeskStore((state) => state.asks[threadId] !== undefined);
  const ask = useDeskStore((state) => state.documentAsks[documentId] ?? null);
  const proposalsQuery = useQuery(assistant.proposals(threadId));
  const turnsQuery = useQuery(assistant.activeTurns());
  const replying = (turnsQuery.data?.items ?? []).some((turn) => turn.threadId === threadId);
  const stageInput = {
    documentId,
    attachedShipmentId: extraction.attachedShipmentId,
    proposals: proposalsQuery.data?.results ?? [],
    replying,
    ask: ask?.threadId === threadId ? ask : null,
    askWaiting,
  };
  const stage = shipmentStage(stageInput);
  const checking = isChecking(stageInput);
  const proposal = shipmentProposal(stageInput.proposals, documentId);
  const decisionsQuery = useQuery({
    ...assistant.artifacts(threadId)._ctx.summary,
    enabled: stage.stage === "waiting" && onOpenArtifact !== undefined,
  });
  const decision = (decisionsQuery.data?.results ?? []).find(
    (candidate) =>
      candidate.kind === "decision_request" &&
      proposal !== null &&
      candidate.proposalId === proposal.id,
  );

  const rowByKey = new Map(rows.map((row) => [row.key, row]));
  const markLabel = (row: FieldRow) =>
    row.kind === "read" && row.field.stop
      ? t("{0} · {1}", stopTitle(row.field.stop, t), row.label)
      : row.label;
  const marks: PageMark[] = extraction.fields.flatMap((field) => {
    const row = rowByKey.get(field.key);
    if (!field.box || field.page !== page || !row) return [];
    return [
      {
        key: field.key,
        label: markLabel(row),
        value: field.value,
        doubtful: field.needsLook,
        box: field.box,
      },
    ];
  });
  const checksByPage = new Map<number, number>();
  for (const field of extraction.fields) {
    if (field.needsLook && field.box) {
      checksByPage.set(field.page, (checksByPage.get(field.page) ?? 0) + 1);
    }
  }
  const firstCheckPage = extraction.fields.find((field) => field.needsLook && field.box)?.page ?? 0;

  const focus = (key: string) => {
    setSelected(key);
    const row = rowByKey.get(key);
    if (row?.kind === "read" && row.field.box && row.field.page > 0) setPage(row.field.page);
  };
  const select = (key: string) => {
    if (selected === key) {
      setSelected(null);
      return;
    }
    focus(key);
  };
  const show = (key: string) => {
    focus(key);
    docRef.current?.scrollIntoView?.({ block: "nearest", behavior: reduced ? "auto" : "smooth" });
  };

  const file = { fileName: fileName || t("this document"), documentId };
  const onCheck = () => {
    noteDocumentAsk(documentId, { threadId, intent: "check" });
    askAgent(threadId, checkQuestion(file, names, t));
  };
  const onCreate = () => {
    noteDocumentAsk(documentId, { threadId, intent: "create" });
    askAgent(threadId, createQuestion(file, t));
  };

  const kind = documentKindName(extraction.documentKind) || extraction.kindLabel || t("Document");
  const lane = laneOf(groups);
  const figure = headFigure(extraction, tally, t);
  const settled = stage.stage === "created" || stage.stage === "attached";
  const standing = headStanding(stage, tally.check.length, t);
  const shipmentId = settled ? stage.shipmentId : "";
  const unsure =
    extraction.kindConfidence !== null && extraction.kindConfidence < SURE_OF_KIND
      ? t("{0}% sure of the kind", Math.round(extraction.kindConfidence * 100))
      : "";

  return (
    <RecordCard className="dk-ex">
      <CardHead
        kicker={[
          kind,
          fileName || t("This document"),
          t("{0, plural, one {# page} other {# pages}}", pageCount),
        ]}
        lead={<SettlementStub mark={documentMark(extraction.documentKind, t)} />}
        title={lane ? t("{0} → {1}", lane.from, lane.to) : kind}
        subtitle={headSubtitle(extraction, lane, t) || undefined}
        figure={
          figure && (
            <>
              <b className={cn(figure.warn && "dk-warn")}>{figure.amount}</b>
              <span>{figure.caption}</span>
            </>
          )
        }
        meta={
          <>
            <CardStatus key={standing.label} tone={standing.tone} mark={standing.mark}>
              {standing.label}
            </CardStatus>
            <CardFacts>
              {[
                shipmentId !== "" ? (
                  <Link
                    key="shipment"
                    className="dk-ex-link ui-focus-ring"
                    to={recordPath("shipment", shipmentId)}
                  >
                    {stage.stage === "created" && stage.name !== ""
                      ? stage.name
                      : t("Open the shipment")}
                  </Link>
                ) : null,
                ...(settled ? [] : headFacts(tally, t)),
                unsure,
              ]}
            </CardFacts>
          </>
        }
      />
      <div className={cn("dk-ex-body", documentId === "" && "dk-nofile")}>
        <div className="dk-ex-reads">
          {!settled && (
            <ExtractionCheck
              documentId={documentId}
              pages={extraction.pages}
              tally={tally}
              selected={selected}
              onSelect={select}
              onShow={show}
            />
          )}
          {rows.length > 0 ? (
            <ExtractionLoad
              groups={groups}
              selected={selected}
              hot={boxHover ?? listHover}
              multiPage={multiPage}
              onSelect={select}
              onPoint={setListHover}
            />
          ) : (
            <CardSection title={t("The load as read")}>
              <p className="dk-ex-empty">{t("Nothing was read off this document.")}</p>
            </CardSection>
          )}
        </div>
        {documentId !== "" ? (
          <ExtractionViewer
            sectionRef={docRef}
            documentId={documentId}
            fileName={fileName || t("this document")}
            page={page}
            pageCount={pageCount}
            pages={extraction.pages}
            checksByPage={checksByPage}
            readAhead={[page + 1, firstCheckPage]}
            onPage={setPage}
            marks={marks}
            hot={listHover ?? boxHover}
            selected={selected}
            onPoint={setBoxHover}
            onSelect={select}
            onOpenFile={() => setFileOpen(true)}
          />
        ) : (
          <CardSection title={t("The document")}>
            <p className="dk-ex-empty" role="status">
              {t("The file this was read from is no longer on record.")}
            </p>
          </CardSection>
        )}
      </div>
      <ExtractionActions
        stage={stage}
        checking={checking}
        checkCount={checkCount(tally)}
        checkSummary={checkCount(tally) > 0 ? checkSummary(names, t) : ""}
        onCheck={onCheck}
        onCreate={onCreate}
        onReview={decision && onOpenArtifact ? () => onOpenArtifact(decision.id) : null}
      />
      {documentId !== "" && (
        <ExtractionFileDialog
          documentId={documentId}
          fileName={fileName}
          image={isImageFile(extraction)}
          page={page}
          pageCount={pageCount}
          open={fileOpen}
          onOpenChange={setFileOpen}
        />
      )}
    </RecordCard>
  );
}
