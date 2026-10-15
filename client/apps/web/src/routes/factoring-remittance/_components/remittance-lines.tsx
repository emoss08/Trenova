import { FactoringRemittanceMatchBadge } from "@/components/factoring/factoring-remittance-badges";
import { SectionPanel } from "@/components/section-panel";
import { recordPath } from "@/config/record-links";
import {
  canEditRemittanceLines,
  FACTORING_REMITTANCE_LINE_KIND_LABELS,
  FACTORING_REMITTANCE_MATCH_REASON_LABELS,
  isRemittanceException,
  matchScoreLabel,
} from "@/lib/factoring-remittance";
import type {
  FactoringRemittance,
  FactoringRemittanceLine,
} from "@/lib/graphql/factoring-remittance";
import type { FactoringRemittanceLineKind } from "@trenova/graphql/generated/graphql";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { formatMinor } from "@trenova/shared/lib/money";
import { useState } from "react";
import { Link } from "react-router";
import { MatchItemDialog } from "./match-item-dialog";
import { ReasonDialog } from "./reason-dialog";
import { useLineUpdate } from "./use-line-update";

const LINE_KINDS: FactoringRemittanceLineKind[] = [
  "Funding",
  "Rejection",
  "ReserveRelease",
  "Chargeback",
  "Other",
];

function lineAmount(line: FactoringRemittanceLine): number | null {
  return line.cashMinor ?? line.advanceMinor ?? line.reserveMinor ?? line.faceMinor ?? null;
}

function LineReference({ line }: { line: FactoringRemittanceLine }) {
  const t = useT();
  return (
    <div className="flex flex-col">
      <span className="font-mono font-medium">
        {line.invoiceNumber || line.factorReference || t("No reference")}
      </span>
      {line.debtorName ? (
        <span className="text-muted-foreground text-xs">{line.debtorName}</span>
      ) : null}
    </div>
  );
}

/**
 * Lines the matcher could not settle on its own, each with its ranked
 * suggestions. A person takes a suggestion, picks another open invoice, says
 * what the line records, or sets it aside with a reason.
 */
export function RemittanceExceptions({
  remittance,
  lines,
}: {
  remittance: FactoringRemittance;
  lines: FactoringRemittanceLine[];
}) {
  const t = useT();
  const editable = canEditRemittanceLines(remittance.status);
  const [picking, setPicking] = useState<FactoringRemittanceLine | null>(null);
  const [ignoring, setIgnoring] = useState<FactoringRemittanceLine | null>(null);
  const update = useLineUpdate(remittance, () => {
    setPicking(null);
    setIgnoring(null);
  });
  const money = (minor: number | null) =>
    minor === null ? "" : formatMinor(minor, remittance.currencyCode);
  const exceptions = lines.filter((line) => isRemittanceException(line.matchStatus));
  const ignored = lines.filter((line) => line.matchStatus === "Ignored");

  return (
    <SectionPanel
      title={t("To resolve")}
      count={exceptions.length}
      help={t(
        "Lines that did not match one open invoice on their own. Nothing here is booked until it is matched and the remittance applied.",
      )}
    >
      {exceptions.length === 0 && ignored.length === 0 ? (
        <p className="text-muted-foreground p-3 text-sm">{t("Every line is matched.")}</p>
      ) : (
        <ul className="divide-border divide-y">
          {exceptions.map((line) => (
            <li key={line.id} className="flex flex-col gap-2 p-3">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="flex items-start gap-3">
                  <span className="text-muted-foreground w-12 text-xs tabular-nums">
                    {t("Row {0}", line.rowNumber)}
                  </span>
                  <LineReference line={line} />
                  <span className="tabular-nums">{money(lineAmount(line))}</span>
                  <FactoringRemittanceMatchBadge status={line.matchStatus} />
                </div>
                {editable ? (
                  <div className="flex flex-wrap items-center gap-2">
                    <Select
                      value={line.kind}
                      onValueChange={(kind) =>
                        update.mutate({
                          lineId: line.id,
                          action: "SetKind",
                          kind: kind as FactoringRemittanceLineKind,
                        })
                      }
                    >
                      <SelectTrigger size="sm" aria-label={t("What the line records")}>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {LINE_KINDS.map((kind) => (
                          <SelectItem key={kind} value={kind}>
                            {FACTORING_REMITTANCE_LINE_KIND_LABELS[kind]}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    {line.matchStatus !== "Error" && line.kind !== "Other" ? (
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onClick={() => setPicking(line)}
                      >
                        {t("Pick invoice")}
                      </Button>
                    ) : null}
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => setIgnoring(line)}
                    >
                      {t("Ignore")}
                    </Button>
                  </div>
                ) : null}
              </div>
              {line.error ? <p className="text-danger text-sm">{line.error}</p> : null}
              {line.reason && line.suggestions.length === 0 ? (
                <p className="text-muted-foreground text-sm">
                  {FACTORING_REMITTANCE_MATCH_REASON_LABELS[line.reason]}
                </p>
              ) : null}
              {line.suggestions.length > 0 ? (
                <ul className="flex flex-col gap-1">
                  {line.suggestions.map((suggestion) => (
                    <li
                      key={suggestion.itemId}
                      className="bg-muted/30 flex items-center justify-between gap-3 rounded-md border px-2 py-1.5 text-sm"
                    >
                      <span className="flex items-center gap-2">
                        <AssistMark className="text-brand size-3.5" aria-label={t("Suggested")} />
                        <Link
                          to={recordPath("factoring_submission", suggestion.submissionId)}
                          className="font-mono hover:underline"
                        >
                          {suggestion.invoiceNumber}
                        </Link>
                        <span className="text-muted-foreground">
                          {FACTORING_REMITTANCE_MATCH_REASON_LABELS[suggestion.reason]}
                        </span>
                        <span className="text-muted-foreground tabular-nums">
                          {matchScoreLabel(suggestion.score)}
                        </span>
                      </span>
                      {editable ? (
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          isLoading={
                            update.isPending && update.variables?.itemId === suggestion.itemId
                          }
                          onClick={() =>
                            update.mutate({
                              lineId: line.id,
                              action: "Match",
                              itemId: suggestion.itemId,
                            })
                          }
                        >
                          {t("Use")}
                        </Button>
                      ) : null}
                    </li>
                  ))}
                </ul>
              ) : null}
            </li>
          ))}
          {ignored.map((line) => (
            <li key={line.id} className="flex flex-wrap items-center justify-between gap-3 p-3">
              <div className="flex items-start gap-3">
                <span className="text-muted-foreground w-12 text-xs tabular-nums">
                  {t("Row {0}", line.rowNumber)}
                </span>
                <LineReference line={line} />
                <FactoringRemittanceMatchBadge status={line.matchStatus} />
                <span className="text-muted-foreground text-sm">{line.ignoredReason}</span>
              </div>
              {editable ? (
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onClick={() => update.mutate({ lineId: line.id, action: "Restore" })}
                >
                  {t("Restore")}
                </Button>
              ) : null}
            </li>
          ))}
        </ul>
      )}
      <MatchItemDialog
        factoringCompanyId={remittance.factoringCompanyId}
        currencyCode={remittance.currencyCode}
        line={picking}
        pending={update.isPending}
        onMatch={(itemId) => {
          if (picking) update.mutate({ lineId: picking.id, action: "Match", itemId });
        }}
        onOpenChange={(open) => {
          if (!open) setPicking(null);
        }}
      />
      <ReasonDialog
        open={ignoring !== null}
        title={t("Ignore line")}
        description={t(
          "An ignored line is not booked from the remittance. Book a fee or adjustment by hand.",
        )}
        placeholder={t("Such as: wire fee, booked by hand")}
        confirmLabel={t("Ignore")}
        pending={update.isPending}
        onConfirm={(reason) => {
          if (ignoring) update.mutate({ lineId: ignoring.id, action: "Ignore", reason });
        }}
        onOpenChange={(open) => {
          if (!open) setIgnoring(null);
        }}
      />
    </SectionPanel>
  );
}

/** Lines matched to an open invoice, waiting to be applied, and the ones booked. */
export function RemittanceMatched({
  remittance,
  lines,
}: {
  remittance: FactoringRemittance;
  lines: FactoringRemittanceLine[];
}) {
  const t = useT();
  const editable = canEditRemittanceLines(remittance.status);
  const update = useLineUpdate(remittance);
  const money = (minor: number | null) =>
    minor === null ? "" : formatMinor(minor, remittance.currencyCode);
  const matched = lines.filter(
    (line) => line.matchStatus === "Matched" || line.matchStatus === "Applied",
  );

  return (
    <SectionPanel title={t("Matched")} count={matched.length}>
      {matched.length === 0 ? (
        <p className="text-muted-foreground p-3 text-sm">{t("No line is matched yet.")}</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("Row")}</TableHead>
              <TableHead>{t("Line")}</TableHead>
              <TableHead>{t("Records")}</TableHead>
              <TableHead>{t("Invoice")}</TableHead>
              <TableHead className="text-right">{t("Amount")}</TableHead>
              <TableHead>{t("Date")}</TableHead>
              <TableHead>{t("Status")}</TableHead>
              {editable ? (
                <TableHead>
                  <span className="sr-only">{t("Actions")}</span>
                </TableHead>
              ) : null}
            </TableRow>
          </TableHeader>
          <TableBody>
            {matched.map((line) => (
              <TableRow key={line.id}>
                <TableCell className="tabular-nums">{line.rowNumber}</TableCell>
                <TableCell>
                  <LineReference line={line} />
                </TableCell>
                <TableCell>{FACTORING_REMITTANCE_LINE_KIND_LABELS[line.kind]}</TableCell>
                <TableCell>
                  {line.matchedSubmissionId ? (
                    <Link
                      to={recordPath("factoring_submission", line.matchedSubmissionId)}
                      className="font-mono hover:underline"
                    >
                      {line.matchedInvoiceNumber}
                    </Link>
                  ) : (
                    line.matchedInvoiceNumber
                  )}
                  {line.matchedBy === "Auto" ? (
                    <AssistMark
                      className="text-brand ml-1 inline size-3"
                      aria-label={t("Matched automatically")}
                    />
                  ) : null}
                </TableCell>
                <TableCell className="text-right tabular-nums">{money(lineAmount(line))}</TableCell>
                <TableCell>{line.occurredOn ? formatUnixDate(line.occurredOn) : ""}</TableCell>
                <TableCell>
                  <FactoringRemittanceMatchBadge status={line.matchStatus} />
                </TableCell>
                {editable ? (
                  <TableCell>
                    {line.matchStatus === "Matched" ? (
                      <Button
                        type="button"
                        size="sm"
                        variant="ghost"
                        isLoading={update.isPending && update.variables?.lineId === line.id}
                        onClick={() => update.mutate({ lineId: line.id, action: "Unmatch" })}
                      >
                        {t("Unmatch")}
                      </Button>
                    ) : null}
                  </TableCell>
                ) : null}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </SectionPanel>
  );
}
