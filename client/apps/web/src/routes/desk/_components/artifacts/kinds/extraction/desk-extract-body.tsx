import { documentKindName } from "@/lib/document-kinds";
import { assistant } from "@/lib/queries/assistant";
import { useDeskStore } from "@/stores/desk-store";
import type { AssistantArtifact } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { LinkExternal01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useId, useState } from "react";
import { ExtractionActions, ShipmentSteps } from "./extraction-actions";
import {
  checkCount,
  checkNames,
  checkQuestion,
  checkSummary,
  createQuestion,
} from "./extraction-asks";
import { ExtractionFields } from "./extraction-fields";
import { ExtractionFileDialog } from "./extraction-file-dialog";
import {
  firstRowOf,
  groupExtraction,
  pagesToCheck,
  reviewTally,
  rowsInOrder,
  type FieldRow,
} from "./extraction-groups";
import { extractionFrom, isImageFile } from "./extraction-payload";
import { ExtractionViewer, type PageMark } from "./extraction-viewer";
import { stopTitle } from "./extraction-words";
import { isChecking, shipmentProposal, shipmentStage } from "./shipment-stage";

type BodyProps = {
  artifact: AssistantArtifact;
  /** Opens another artifact of the conversation, such as the approval of the drafted shipment. */
  onOpenArtifact?: (id: string) => void;
};

/**
 * What document intelligence read off a file, document first: its pages with a quiet box over
 * every value found, beside the values grouped the way a dispatcher reads a load. Choosing a
 * value (pointer or keyboard) lights its box, turns to its page and scrolls the box into view;
 * choosing a box chooses its value. The tally counts what needs a person, what the document
 * lacked and what was read. The shipment is drafted by the agent and waits for approval, as
 * every write does; the card only asks.
 */
export function DeskExtractBody({ artifact, onOpenArtifact }: BodyProps) {
  const t = useT();
  const listId = useId();
  const extraction = extractionFrom(artifact.payload);
  const { documentId, fileName, pageCount } = extraction;
  const threadId = artifact.threadId;
  const groups = groupExtraction(extraction);
  const rows = rowsInOrder(groups);
  const tally = reviewTally(groups);
  const names = checkNames(tally, t);
  const multiPage = pageCount > 1;

  const [selected, setSelected] = useState<string | null>(null);
  const [listHover, setListHover] = useState<string | null>(null);
  const [boxHover, setBoxHover] = useState<string | null>(null);
  const [page, setPage] = useState(() => {
    const first = extraction.fields.find((field) => field.box !== null);
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
  const checkPages = pagesToCheck(extraction.fields);
  const firstCheckPage = extraction.fields.find((field) => field.needsLook && field.box)?.page ?? 0;

  const select = (key: string, from: "pointer" | "keyboard" | "box") => {
    if (from === "pointer" && selected === key) {
      setSelected(null);
      return;
    }
    setSelected(key);
    const row = rowByKey.get(key);
    if (row?.kind === "read" && row.field.box && row.field.page > 0) setPage(row.field.page);
  };
  const jump = (row: FieldRow | undefined) => {
    if (row) select(row.key, "keyboard");
  };

  const file = { fileName: fileName || t("this document"), documentId };
  const onCheck = () => {
    noteDocumentAsk(documentId, { threadId, intent: "check", at: Date.now() });
    askAgent(threadId, checkQuestion(file, names, t));
  };
  const onCreate = () => {
    noteDocumentAsk(documentId, { threadId, intent: "create", at: Date.now() });
    askAgent(threadId, createQuestion(file, t));
  };
  const kind = documentKindName(extraction.documentKind) || extraction.kindLabel || t("Document");

  return (
    <div className="dk-ex">
      <div className="dk-ex-head">
        <div className="dk-ex-id">
          <span className="dk-ex-kind">{kind}</span>
          {extraction.kindConfidence !== null && (
            <span className="dk-ex-sure">
              {t("{0}% sure of the kind", Math.round(extraction.kindConfidence * 100))}
            </span>
          )}
          {documentId !== "" && (
            <Button
              variant="outline"
              size="xs"
              className="dk-ex-open"
              onClick={() => setFileOpen(true)}
            >
              {t("Open file")}
              <LinkExternal01Icon className="size-3" />
            </Button>
          )}
        </div>
        <div className="dk-ex-file">
          <span className="dk-ex-fname">{fileName || t("This document")}</span>
          <span>{t("{0, plural, one {# page} other {# pages}}", pageCount)}</span>
        </div>
        <ShipmentSteps stage={stage} />
        {rows.length > 0 && (
          <div className="dk-ex-tally" role="group" aria-label={t("What was read")}>
            {tally.check.length > 0 && (
              <Button
                variant="bare"
                size="bare"
                className="dk-ex-t dk-warn"
                onClick={() => jump(firstRowOf(tally.check[0]))}
              >
                <i aria-hidden className="dk-ex-dot" />
                {t("{0} to check", tally.check.length)}
              </Button>
            )}
            {tally.missing.length > 0 && (
              <Button
                variant="bare"
                size="bare"
                className="dk-ex-t dk-miss"
                onClick={() => jump(tally.missing[0])}
              >
                <i aria-hidden className="dk-ex-ring" />
                {t("{0} not on the document", tally.missing.length)}
              </Button>
            )}
            <Button
              variant="bare"
              size="bare"
              className="dk-ex-t"
              disabled={tally.read === 0}
              onClick={() =>
                jump(rows.find((row) => row.state === "settled" || row.state === "fair"))
              }
            >
              {t("{0} read", tally.read)}
            </Button>
          </div>
        )}
      </div>
      <div className={cn("dk-ex-main", documentId === "" && "dk-nofile")}>
        {documentId !== "" ? (
          <ExtractionViewer
            documentId={documentId}
            fileName={fileName || t("this document")}
            page={page}
            pageCount={pageCount}
            pages={extraction.pages}
            pagesToCheck={checkPages}
            readAhead={[page + 1, firstCheckPage]}
            onPage={setPage}
            marks={marks}
            hot={listHover ?? boxHover}
            selected={selected}
            onPoint={setBoxHover}
            onSelect={(key) => select(key, "box")}
            onOpenFile={() => setFileOpen(true)}
          />
        ) : (
          <section className="dk-ex-viewer" aria-label={t("Document")}>
            <p className="dk-ex-pfail" role="status">
              {t("The file this was read from is no longer on record.")}
            </p>
          </section>
        )}
        {rows.length > 0 ? (
          <ExtractionFields
            listId={listId}
            groups={groups}
            multiPage={multiPage}
            page={page}
            selected={selected}
            hot={boxHover ?? listHover}
            onSelect={select}
            onClear={() => setSelected(null)}
            onPoint={setListHover}
          />
        ) : (
          <p className="dk-ex-fields dk-ex-empty">{t("Nothing was read off this document.")}</p>
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
    </div>
  );
}
