import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { cn } from "@trenova/shared/lib/utils";
import type {
  EDIInboundFile,
  EDIInspectionDiagnostic,
  EDIX12Inspection,
} from "@trenova/shared/types/edi";
import { useState, type CSSProperties } from "react";
import { EdiFinding } from "../../record-card/edi-finding";
import { recordPath } from "@/config/record-links";
import { CardIcon, CardOpenLink, CardRecordRow } from "../../record-card/record-card";
import { cardMoment } from "../../record-card/record-card-values";
import type {
  IntakeStage,
  ManifestEntry,
  ManifestGroup,
  ProcessingNote,
  RoutingLabel,
  RoutingParty,
  RulerRun,
} from "./edi-inbound-file-card";

/** The findings a list shows before it folds the rest away. */
const FINDINGS_HEAD = 3;

/** The extension a file name ends in, or none. */
function extensionOf(fileName: string): string {
  const dot = fileName.lastIndexOf(".");
  return dot > 0 && dot < fileName.length - 1 ? fileName.slice(dot + 1, dot + 5) : "";
}

const METHOD_TAG: Record<EDIInboundFile["method"], string> = {
  AS2: "AS2",
  SFTP: "SFTP",
  VAN: "VAN",
  Internal: "TMS",
};

/**
 * The file as a sheet with its corner folded down: its extension on the sheet
 * and, tabbed to its foot, the way it came in.
 */
export function FileGlyph({
  fileName,
  method,
}: {
  fileName: string;
  method: EDIInboundFile["method"];
}) {
  const extension = extensionOf(fileName);
  return (
    <span className="dk-eif-glyph" aria-hidden>
      <span className="dk-eif-sheet">
        <b>{extension ? `.${extension.toLowerCase()}` : "—"}</b>
      </span>
      <span className="dk-eif-tab">{METHOD_TAG[method]}</span>
    </span>
  );
}

const STAGE_ICON: Record<IntakeStage["state"], "check" | "x" | "pause" | null> = {
  done: "check",
  now: null,
  partial: "pause",
  failed: "x",
  ahead: null,
  skipped: null,
};

/**
 * The file's intake as chevrons pointing on, one per stage: a stage it came
 * through filled, the one it is in drawn in ink, one it came through only in
 * part in amber, one it failed in red, one ahead or never reached left open.
 */
export function IntakeChevrons({ stages }: { stages: readonly IntakeStage[] }) {
  const t = useT();
  return (
    <ol className="dk-eif-intake" aria-label={t("Intake")}>
      {stages.map((stage) => {
        const icon = STAGE_ICON[stage.state];
        return (
          <li
            key={stage.key}
            className={cn("dk-eif-stage", `dk-${stage.state}`)}
            aria-current={stage.state === "now" ? "step" : undefined}
          >
            <span className="dk-eif-chev">
              {icon ? <CardIcon name={icon} size={11} stroke={2.8} /> : <i aria-hidden />}
              <b>{stage.label}</b>
            </span>
            {stage.detail && <span className="dk-eif-stage-d">{stage.detail}</span>}
            {stage.at ? <time>{cardMoment(stage.at)}</time> : null}
          </li>
        );
      })}
    </ol>
  );
}

const NOTE_ICON: Record<ProcessingNote["severity"], "x" | "warn" | "info"> = {
  bad: "x",
  warn: "warn",
  info: "info",
};

/** One thing processing said, with the transaction set it names tabbed beside it. */
function NoteLine({ note, showRef }: { note: ProcessingNote; showRef: boolean }) {
  return (
    <li className={cn("dk-eif-note", `dk-${note.severity}`)}>
      <span className="dk-eif-note-i" aria-hidden>
        <CardIcon name={NOTE_ICON[note.severity]} size={10} stroke={2.8} />
      </span>
      <span className="dk-eif-note-t">{note.text}</span>
      {showRef && note.set && note.control && (
        <code className="dk-eif-ref">
          {note.set} · {note.control}
        </code>
      )}
    </li>
  );
}

/** What processing said of the file, kept as the service wrote it, a note to a line. */
export function ProcessingLedger({
  notes,
  label,
}: {
  notes: readonly ProcessingNote[];
  label: string;
}) {
  return (
    <ul className="dk-eif-notes" aria-label={label}>
      {notes.map((note) => (
        <NoteLine key={note.key} note={note} showRef />
      ))}
    </ul>
  );
}

/** One side of the address: the ISA's ID set large, its qualifier tabbed beside it, the name under. */
function LabelParty({ caption, party }: { caption: string; party: RoutingParty }) {
  const t = useT();
  return (
    <div className="dk-eif-party">
      <span className="dk-eif-cap">{caption}</span>
      <span className="dk-eif-id">
        {party.id ? <code>{party.id}</code> : <em>{t("Not read")}</em>}
        {party.qualifier && (
          <abbr className="dk-eif-qual" title={party.qualifierName}>
            {party.qualifier}
          </abbr>
        )}
      </span>
      {party.name && <span className="dk-eif-pname">{party.name}</span>}
    </div>
  );
}

/** The checksum drawn as bars, two to a hex digit, so two files can be told apart at a look. */
function FingerprintBars({ fingerprint }: { fingerprint: string }) {
  const digits = fingerprint.replaceAll(" ", "");
  return (
    <span className="dk-eif-bars" aria-hidden>
      {[...digits].map((digit, index) => {
        const value = Number.parseInt(digit, 16) || 0;
        return (
          // oxlint-disable-next-line react/no-array-index-key -- a fingerprint's digits never reorder
          <span key={index}>
            <i style={{ "--dk-eif-w": (value >> 2) + 1 } as CSSProperties} />
            <i style={{ "--dk-eif-w": (value & 3) + 1 } as CSSProperties} />
          </span>
        );
      })}
    </span>
  );
}

/**
 * The file's routing label: from whom to whom as its ISA says, the
 * interchange it carries, then the channel, the AS2 message or path it came
 * as, and its fingerprint and size along the foot.
 */
export function RoutingLabelView({ label }: { label: RoutingLabel }) {
  const t = useT();
  const fields = [
    { key: "channel", label: t("Channel"), value: label.channel },
    { key: "way", label: label.way.label, value: label.way.value },
    { key: "size", label: t("Size"), value: label.size },
  ].filter((field) => field.value);
  return (
    <div className="dk-eif-label">
      <div className="dk-eif-addr">
        <LabelParty caption={t("Sender")} party={label.from} />
        <span className="dk-eif-arrow" aria-hidden>
          <CardIcon name="forward" size={14} stroke={2.2} />
        </span>
        <LabelParty caption={t("Receiver")} party={label.to} />
        <div className="dk-eif-icn">
          <span className="dk-eif-cap">{t("Interchange number")}</span>
          {label.interchange ? <code>{label.interchange}</code> : <em>{t("Not read")}</em>}
        </div>
      </div>
      <dl className="dk-eif-foot">
        {fields.map((field) => (
          <div key={field.key} className={`dk-eif-f-${field.key}`}>
            <dt>{field.label}</dt>
            <dd title={field.value}>{field.value}</dd>
          </div>
        ))}
        {label.fingerprint && (
          <div className="dk-eif-f-print">
            <dt>{t("Fingerprint")}</dt>
            <dd>
              <FingerprintBars fingerprint={label.fingerprint} />
              <code>{label.fingerprint}</code>
            </dd>
          </div>
        )}
      </dl>
    </div>
  );
}

/**
 * The interchange as one ruler, a run per transaction set as long as its
 * segments, shaded by the group it sits in, a flagged set in the colour of its
 * worst finding, and the envelope as the gaps between.
 */
export function SegmentRulerView({ runs, total }: { runs: readonly RulerRun[]; total: number }) {
  const t = useT();
  return (
    <div className="dk-eif-ruler">
      <div
        className="dk-eif-runs"
        role="img"
        aria-label={t("{0, plural, one {# segment} other {# segments}}", total)}
      >
        {runs.map((run) => (
          <span
            key={run.key}
            className={cn(
              "dk-eif-run",
              `dk-${run.kind}`,
              run.group !== null && (run.group % 2 === 0 ? "dk-even" : "dk-odd"),
              run.state !== "ok" && `dk-${run.state}`,
            )}
            style={{ flexGrow: run.span } as CSSProperties}
          />
        ))}
      </div>
      <div className="dk-eif-scale" aria-hidden>
        <span>{formatNumber(1)}</span>
        <span>{t("{0, plural, one {# segment} other {# segments}}", total)}</span>
      </div>
    </div>
  );
}

const VERDICT_ICON: Record<ManifestEntry["state"], "check" | "warn" | "x" | "pause" | null> = {
  routed: "check",
  noted: "warn",
  failed: "x",
  pending: "pause",
  skipped: null,
};

/** One transaction set's row: its number, its name and control number, what it says, what became of it. */
function EntryRow({ entry }: { entry: ManifestEntry }) {
  const t = useT();
  const icon = VERDICT_ICON[entry.state];
  return (
    <>
      <span className="dk-eif-set">{entry.set || "—"}</span>
      <span className="dk-rec-main">
        <b>
          {entry.name}
          <code>{entry.control}</code>
        </b>
        <span className="dk-rec-sub">
          {[entry.gist, t("{0, plural, one {# segment} other {# segments}}", entry.segments)]
            .filter(Boolean)
            .join(" · ")}
        </span>
      </span>
      <span className={cn("dk-eif-verdict", `dk-${entry.state}`)}>
        {icon && <CardIcon name={icon} size={10} stroke={2.8} />}
        {entry.verdict}
      </span>
    </>
  );
}

/** The tender a 204 opened, as a manifest row names it. */
export type ManifestTender = { id: string; label: string };

/** One transaction set the file carried, opened as its message when one was kept. */
function ManifestRow({
  entry,
  messagePath,
  onOpen,
  tender,
  onOpenTender,
}: {
  entry: ManifestEntry;
  messagePath: (id: string) => string;
  onOpen?: (id: string) => void;
  tender?: ManifestTender;
  onOpenTender?: (id: string) => void;
}) {
  const t = useT();
  const message = entry.message;
  const count = entry.countCheck;
  return (
    <li className={cn("dk-eif-entry", `dk-${entry.state}`)}>
      {message ? (
        <CardRecordRow
          to={messagePath(message.id)}
          onOpen={onOpen && (() => onOpen(message.id))}
          className="dk-eif-row"
        >
          <EntryRow entry={entry} />
          <CardIcon name="forward" size={12} />
        </CardRecordRow>
      ) : (
        <div className="dk-eif-row dk-unkept">
          <EntryRow entry={entry} />
        </div>
      )}
      {count && !count.ok && (
        <p className="dk-eif-miscount">
          <CardIcon name="x" size={10} stroke={2.8} />
          {t(
            "SE01 says {0}; {1} counted",
            formatNumber(count.expected),
            formatNumber(count.actual),
          )}
        </p>
      )}
      {tender && (
        <div className="dk-eif-tender">
          <CardOpenLink
            to={recordPath("edi_transfer", tender.id)}
            onOpen={onOpenTender && (() => onOpenTender(tender.id))}
          >
            {tender.label}
          </CardOpenLink>
        </div>
      )}
      {entry.notes.length > 0 && (
        <ul className="dk-eif-notes dk-under" aria-label={t("Notes on {0}", entry.control)}>
          {entry.notes.map((note) => (
            <NoteLine key={note.key} note={note} showRef={false} />
          ))}
        </ul>
      )}
      {entry.findings.length > 0 && (
        <div className="dk-eif-findings dk-under">
          {entry.findings.map((finding, index) => (
            // oxlint-disable-next-line react/no-array-index-key -- a set's findings never reorder
            <EdiFinding key={`${finding.code}-${index}`} diagnostic={finding} />
          ))}
        </div>
      )}
    </li>
  );
}

/**
 * What the file carried, a folder per functional group: its identifier, what
 * it holds and its control number, whether its trailer counts it right, then
 * each transaction set in it.
 */
export function FileManifest({
  groups,
  messagePath,
  onOpen,
  tenders,
  onOpenTender,
}: {
  groups: readonly ManifestGroup[];
  messagePath: (id: string) => string;
  onOpen?: (id: string) => void;
  /** The tender each kept 204 opened, by the message's id. */
  tenders?: ReadonlyMap<string, ManifestTender>;
  onOpenTender?: (id: string) => void;
}) {
  const t = useT();
  return (
    <div className="dk-eif-groups">
      {groups.map((group) => (
        <section key={group.key} className="dk-eif-group" aria-label={group.name}>
          <header className="dk-eif-gh">
            {group.code && <span className="dk-eif-gcode">{group.code}</span>}
            <b>{group.name}</b>
            {group.control && <code>{t("GS {0}", group.control)}</code>}
            {group.countCheck && (
              <span className={cn("dk-eif-gcount", group.countCheck.ok ? "dk-ok" : "dk-bad")}>
                <CardIcon name={group.countCheck.ok ? "check" : "x"} size={10} stroke={2.8} />
                {group.countCheck.ok
                  ? t(
                      "{0, plural, one {# set, as declared} other {# sets, as declared}}",
                      group.countCheck.actual,
                    )
                  : t(
                      "GE01 says {0}; {1} found",
                      formatNumber(group.countCheck.expected),
                      formatNumber(group.countCheck.actual),
                    )}
              </span>
            )}
          </header>
          <ol className="dk-eif-entries">
            {group.entries.map((entry) => (
              <ManifestRow
                key={entry.key}
                entry={entry}
                messagePath={messagePath}
                onOpen={onOpen}
                tender={entry.message ? tenders?.get(entry.message.id) : undefined}
                onOpenTender={onOpenTender}
              />
            ))}
          </ol>
        </section>
      ))}
    </div>
  );
}

/** Findings, the first few shown and the rest folded until asked for. */
export function FindingList({ findings }: { findings: readonly EDIInspectionDiagnostic[] }) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const shown = open ? findings : findings.slice(0, FINDINGS_HEAD);
  const hidden = findings.length - FINDINGS_HEAD;
  return (
    <div className="dk-eif-findings">
      {shown.map((finding, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- the file's findings never reorder
        <EdiFinding key={`${finding.code}-${index}`} diagnostic={finding} />
      ))}
      {hidden > 0 && (
        <Button
          variant="bare"
          size="bare"
          className="dk-eif-more"
          onClick={() => setOpen((current) => !current)}
          aria-expanded={open}
        >
          <CardIcon name={open ? "up" : "down"} size={11} />
          {open
            ? t("Show the first {0}", formatNumber(FINDINGS_HEAD))
            : t("{0, plural, one {Show # more finding} other {Show # more findings}}", hidden)}
        </Button>
      )}
    </div>
  );
}

/** The start of a file that is not X12, as it was sent, and how much more of it there is. */
export function RawGlimpse({ lines, more }: { lines: readonly string[]; more: number }) {
  const t = useT();
  return (
    <figure className="dk-eif-glimpse">
      <pre>
        {lines.map((line, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- a file's lines never reorder
          <span key={index}>
            <i aria-hidden>{formatNumber(index + 1)}</i>
            {line || " "}
          </span>
        ))}
      </pre>
      {more > 0 && (
        <figcaption>{t("{0, plural, one {# more line} other {# more lines}}", more)}</figcaption>
      )}
    </figure>
  );
}

/**
 * The interchange's envelope as checks: its ISA control number closed by the
 * IEA's, the groups it declares against those it holds, and the separators it
 * was read with.
 */
export function EnvelopeChecks({ inspection }: { inspection: EDIX12Inspection }) {
  const t = useT();
  const { envelope, separators } = inspection;
  const isa = envelope.isaControlNumber?.trim() ?? "";
  const iea = envelope.ieaControlNumber?.trim() ?? "";
  const closed = isa !== "" && isa === iea;
  const counted = envelope.expectedGroups === envelope.actualGroups;
  return (
    <ul className="dk-eif-checks">
      <li className={closed ? "dk-ok" : "dk-bad"}>
        <CardIcon name={closed ? "check" : "x"} size={10} stroke={2.8} />
        <span>
          {closed
            ? t("ISA {0} closed by its IEA", isa)
            : t("ISA {0} closed by IEA {1}", isa || "—", iea || "—")}
        </span>
      </li>
      <li className={counted ? "dk-ok" : "dk-bad"}>
        <CardIcon name={counted ? "check" : "x"} size={10} stroke={2.8} />
        <span>
          {counted
            ? t(
                "{0, plural, one {# group, as declared} other {# groups, as declared}}",
                envelope.actualGroups,
              )
            : t(
                "IEA01 says {0}; {1} found",
                formatNumber(envelope.expectedGroups),
                formatNumber(envelope.actualGroups),
              )}
        </span>
      </li>
      <li className="dk-sep">
        <span>{t("Separators")}</span>
        <code title={t("Element")}>{separators.element}</code>
        <code title={t("Segment")}>{separators.segment}</code>
        <code title={t("Component")}>{separators.component}</code>
      </li>
    </ul>
  );
}
