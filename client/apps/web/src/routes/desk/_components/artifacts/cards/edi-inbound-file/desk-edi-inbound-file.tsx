import { recordPath } from "@/config/record-links";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { describeApiError } from "@/lib/api-error-message";
import { edi } from "@/lib/queries/edi";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatFileSize } from "@trenova/shared/lib/utils";
import type {
  EDIInboundFile,
  EDIInspectionDiagnostic,
  EDITransfer,
} from "@trenova/shared/types/edi";
import { useMemo } from "react";
import { Link } from "react-router";
import {
  CardBar,
  CardBarHint,
  CardFacts,
  CardHead,
  CardIcon,
  CardSection,
  CardSectionError,
  CardStatus,
  CardStepper,
  CardUnreadable,
  RecordCard,
} from "../../record-card/record-card";
import { cardMoment, neighborsIn } from "../../record-card/record-card-values";
import { useListedRecords, type RecordArtifactProps } from "../registry";
import {
  fileIntake,
  fileManifest,
  fileStanding,
  looseFindings,
  processingNotes,
  rawGlimpse,
  routingLabel,
  segmentRuler,
} from "./edi-inbound-file-card";
import { transferStanding } from "../../record-card/edi-transfer-standing";
import {
  EnvelopeChecks,
  FileGlyph,
  FileManifest,
  FindingList,
  IntakeChevrons,
  type ManifestTender,
  ProcessingLedger,
  RawGlimpse,
  RoutingLabelView,
  SegmentRulerView,
} from "./edi-inbound-file-parts";
import {
  DeskEdiInboundFileSkeleton,
  EnvelopeSkeleton,
  ManifestSkeleton,
} from "./edi-inbound-file-skeleton";

/**
 * What the card is given: the file, and when it was opened from a table or
 * from a card that lists it, the rows in the order shown, how to step to
 * another, and the way back.
 */
export type EdiInboundFileCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the file was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];
const NO_FINDINGS: readonly EDIInspectionDiagnostic[] = [];

/** The lines of a file that is not X12 the card shows before it says how many more there are. */
const GLIMPSE_LINES = 6;

/**
 * One file a trading partner sent, as an EDI coordinator reads it, opened
 * from a table's row or looked up by the agent: how it came in and how far
 * processing took it, what processing said of it, the routing label its ISA
 * envelope carries, what it carried group by group with each transaction set
 * opening as the message it was kept as, and its interchange's envelope
 * checked end to end. It reads; reprocessing stays on the EDI inbound files
 * page, one click away.
 *
 * The card steps through the rows it was opened from in the order shown.
 * A step draws the next file's skeleton until it is read, never the one it left.
 */
export function DeskEdiInboundFile({
  recordId,
  order: rowOrder = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: EdiInboundFileCardProps) {
  const t = useT();
  const query = useQuery({ ...edi.inboundFile(recordId), retry: false });
  const file = query.data;

  if (!file) {
    return query.isError ? (
      <CardUnreadable
        kind={t("EDI inbound file")}
        title={t("This EDI inbound file could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskEdiInboundFileSkeleton stepper={onSelect !== undefined && rowOrder.length > 1} />
    );
  }

  return (
    <EdiInboundFileCard
      key={file.id}
      file={file}
      rowOrder={rowOrder}
      onSelect={onSelect}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/**
 * The file as an artifact of its own, when the agent looked it up: the same
 * card, with nothing to step through and nowhere to go back to.
 */
export function DeskEdiInboundFileArtifact({ recordId }: RecordArtifactProps) {
  return <DeskEdiInboundFile recordId={recordId} />;
}

/** Whether the file's raw content is gone, by the retention policy or never kept. */
function rawGone(file: EDIInboundFile): boolean {
  return (file.rawPurgedAt !== null && file.rawPurgedAt !== undefined) || !file.rawContent?.trim();
}

/** Why the file carries no transaction set the card can list. */
function emptyManifestNote(file: EDIInboundFile, gone: boolean, t: ReturnType<typeof useT>) {
  if (file.status === "Received")
    return t("Its transaction sets are listed once its envelope is read.");
  if (gone) return t("No messages were kept from it.");
  return t("Nothing in it could be read as a transaction set.");
}

/** A tender's bill of lading, or what it is when it carries none. */
function tenderSlipTitle(tender: EDITransfer, t: ReturnType<typeof useT>): string {
  return tender.tenderPayload.bol?.trim() || t("without a BOL");
}

/** What the bar says of where the file's actions are. */
function barHint(file: EDIInboundFile, t: ReturnType<typeof useT>): string {
  switch (file.status) {
    case "Quarantined":
    case "PartiallyProcessed":
      return t("Reprocess it on the EDI inbound files page");
    case "Duplicate":
      return t("The first copy of this interchange was processed already");
    case "Received":
    case "Parsed":
      return t("Processing is under way");
    case "Processed":
      return t("Open a transaction set to read the message it was kept as");
  }
}

function EdiInboundFileCard({
  file,
  rowOrder,
  onSelect,
  onBack,
  backLabel,
}: Omit<EdiInboundFileCardProps, "order" | "recordId"> & {
  file: EDIInboundFile;
  rowOrder: readonly string[];
}) {
  const t = useT();
  const id = file.id;
  const gone = rawGone(file);
  const inspection = useQuery({
    ...edi.inboundFileInspection(id),
    retry: false,
    enabled: !gone,
  });
  const inspected = inspection.data;

  const standing = fileStanding(file, t);
  const neighbors = neighborsIn(rowOrder, id);
  const partner = file.partner ?? null;
  const notes = useMemo(() => processingNotes(file), [file]);
  const manifest = useMemo(
    () => fileManifest(file, inspected, notes, t),
    [file, inspected, notes, t],
  );
  const ledger = useMemo(() => {
    const tied = new Set(
      manifest.flatMap((group) => group.entries.flatMap((entry) => entry.notes)),
    );
    return notes.filter((note) => !tied.has(note));
  }, [manifest, notes]);
  const ruler = useMemo(() => (inspected ? segmentRuler(inspected) : null), [inspected]);
  const loose = inspected ? looseFindings(inspected) : NO_FINDINGS;
  const interchange = inspected?.interchange ?? false;

  const carriedTender = (file.messages ?? []).some((message) => message.transactionSet === "204");
  const tenders = useQuery({
    ...edi.inboundFileTransfers(id),
    retry: false,
    enabled: carriedTender,
  }).data;
  const tenderByMessage = useMemo(() => {
    const byMessage = new Map<string, ManifestTender>();
    for (const tender of tenders ?? []) {
      if (!tender.inboundMessageId) continue;
      byMessage.set(tender.inboundMessageId, {
        id: tender.id,
        label: t(
          "Tender {0} · {1}",
          tenderSlipTitle(tender, t),
          transferStanding(tender, "inbound", t).label,
        ),
      });
    }
    return byMessage;
  }, [tenders, t]);
  const tenderIds = useMemo(() => {
    const ids = [...tenderByMessage.values()].map((tender) => tender.id);
    return ids.length > 0 ? ids : NO_ROWS;
  }, [tenderByMessage]);

  const messageIds = useMemo(() => {
    const ids = manifest.flatMap((group) =>
      group.entries.flatMap((entry) => (entry.message ? [entry.message.id] : [])),
    );
    return ids.length > 0 ? ids : NO_ROWS;
  }, [manifest]);
  const listedMessages = useListedRecords({
    entity: "edi_message",
    order: messageIds,
    backLabel: t("Back to {0}", file.fileName),
  });
  const listedTenders = useListedRecords({
    entity: "edi_transfer",
    order: tenderIds,
    backLabel: t("Back to {0}", file.fileName),
  });

  // A message or a tender opened from the card takes the card's place, and its
  // way back returns here.
  const opened = listedMessages.opened ?? listedTenders.opened;
  if (opened) {
    return opened;
  }

  const waitingOnInspection = !gone && !inspected && !inspection.isError;

  return (
    <RecordCard className="dk-eif">
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={[
          t("Inbound file"),
          partner?.code ?? "",
          file.method === "Internal" ? t("Internal") : file.method,
        ]}
        stepper={
          onSelect &&
          rowOrder.length > 1 && (
            <CardStepper
              neighbors={neighbors}
              onStep={onSelect}
              previousLabel={t("Previous file")}
              nextLabel={t("Next file")}
            />
          )
        }
        lead={<FileGlyph fileName={file.fileName} method={file.method} />}
        title={<span className="dk-eif-name">{file.fileName}</span>}
        subtitle={partner ? t("From {0}", partner.name) : t("No partner named")}
        meta={
          <>
            <CardStatus key={standing.label} tone={standing.tone} mark={standing.mark}>
              {standing.label}
            </CardStatus>
            <CardFacts>
              {[
                cardMoment(file.receivedAt),
                formatFileSize(file.sizeBytes),
                file.transactionCount > 0
                  ? t(
                      "{0, plural, one {# transaction set} other {# transaction sets}}",
                      file.transactionCount,
                    )
                  : "",
                gone ? t("Raw file purged") : "",
              ]}
            </CardFacts>
          </>
        }
      />

      <CardSection title={t("Intake")}>
        <IntakeChevrons stages={fileIntake(file, t)} />
        {ledger.length > 0 && <ProcessingLedger notes={ledger} label={t("Processing notes")} />}
      </CardSection>

      <CardSection title={t("Routing label")}>
        <RoutingLabelView label={routingLabel(file, t)} />
      </CardSection>

      <CardSection
        title={t("What it carried")}
        aside={
          inspected && interchange ? (
            <span className="dk-eif-aside">
              {t(
                "{0, plural, one {# group} other {# groups}} · {1, plural, one {# set} other {# sets}}",
                inspected.summary.groupCount,
                inspected.summary.transactionCount,
              )}
            </span>
          ) : undefined
        }
      >
        {waitingOnInspection ? (
          <ManifestSkeleton />
        ) : manifest.length > 0 ? (
          <>
            {inspected && interchange && (
              <SegmentRulerView runs={ruler ?? []} total={inspected.summary.segmentCount} />
            )}
            <FileManifest
              groups={manifest}
              messagePath={(messageId) => recordPath("edi_message", messageId)}
              onOpen={listedMessages.open}
              tenders={tenderByMessage}
              onOpenTender={listedTenders.open}
            />
          </>
        ) : (
          <p className="dk-rec-note">{emptyManifestNote(file, gone, t)}</p>
        )}
      </CardSection>

      <CardSection
        title={t("The interchange")}
        aside={gone ? undefined : <CopyRaw raw={file.rawContent ?? ""} />}
      >
        {gone ? (
          <p className="dk-rec-note">
            {file.rawPurgedAt
              ? t(
                  "The raw file was purged {0} under the retention policy; the messages kept from it are listed above.",
                  cardMoment(file.rawPurgedAt),
                )
              : t("The raw file was not kept.")}
          </p>
        ) : inspection.isError ? (
          <CardSectionError>
            {describeApiError(inspection.error, t("The interchange could not be inspected."))}
          </CardSectionError>
        ) : !inspected ? (
          <EnvelopeSkeleton />
        ) : interchange ? (
          <>
            <EnvelopeChecks inspection={inspected} />
            {loose.length > 0 && <FindingList findings={loose} />}
          </>
        ) : (
          <>
            <p className="dk-rec-note">{t("This file is not an X12 interchange. It begins:")}</p>
            <RawGlimpse {...rawGlimpse(file.rawContent ?? "", GLIMPSE_LINES)} />
            <FindingList findings={loose} />
          </>
        )}
      </CardSection>

      <CardBar>
        <CardBarHint>{barHint(file, t)}</CardBarHint>
        <Link className="dk-ax-btn dk-ink" to={recordPath("edi_inbound_file", id)}>
          <CardIcon name="ext" size={13} />
          {t("Open file")}
        </Link>
      </CardBar>
    </RecordCard>
  );
}

/** The way to copy the file whole, as it was received. */
function CopyRaw({ raw }: { raw: string }) {
  const t = useT();
  const { copy, isCopied } = useCopyToClipboard();
  return (
    <Button variant="bare" size="bare" className="dk-rec-lnk" onClick={() => void copy(raw)}>
      <CardIcon name={isCopied ? "check" : "copy"} size={11} />
      {isCopied ? t("Copied") : t("Copy file")}
    </Button>
  );
}
