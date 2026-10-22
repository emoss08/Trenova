import { checkTariffSource } from "@/components/formula-editor/tariff-service";
import { BillView } from "@/components/tariff-bill/bill-view";
import { refusalText } from "@/components/tariff-bill/refusal-text";
import { TariffCode } from "@/components/tariff-code/tariff-code";
import { recordPath } from "@/config/record-links";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { usePermission } from "@/hooks/use-permission";
import { useTariffCheckContext } from "@/hooks/use-tariff-check-context";
import { describeApiError } from "@/lib/api-error-message";
import { formulaTemplate } from "@/lib/queries/formula-template";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { formatList } from "@trenova/shared/i18n/format";
import { useRules } from "@trenova/shared/hooks/use-rules";
import { translateMessage } from "@trenova/shared/i18n/message";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatTariffValue } from "@trenova/shared/lib/tariff-value";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type {
  FormulaPreviewResponse,
  FormulaTemplate,
  ReadinessCheck,
} from "@trenova/shared/types/formula-template";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useMemo } from "react";
import { Link } from "react-router";
import {
  CardBar,
  CardBarHint,
  CardCheck,
  CardChecks,
  CardFacts,
  CardHead,
  CardIcon,
  CardLink,
  CardSection,
  CardStatus,
  CardStepper,
  CardTimeline,
  RecordCard,
  type CardCheckState,
  type CardTone,
} from "../../record-card/record-card";
import { cardDay, cardMoment, neighborsIn } from "../../record-card/record-card-values";
import type { RecordArtifactProps } from "../registry";
import {
  formulaStage,
  formulaStatusLabel,
  formulaStatusNote,
  totalClauseLabel,
  usageLabel,
  type FormulaStage,
} from "./formula-template-card";
import { DeskFormulaTemplateSkeleton } from "./formula-template-skeleton";

/** The versions the card lists; the studio holds the rest. */
const VERSION_ROWS = 4;

const STAGE_TONE: Record<FormulaStage, CardTone> = {
  active: "success",
  review: "progress",
  draft: "draft",
  inactive: "muted",
};

const READINESS_STATE: Record<ReadinessCheck["status"], CardCheckState> = {
  pass: "ok",
  warn: "warn",
  fail: "bad",
};

/**
 * What the card is given: the template, and when it was opened from a table, the table's rows
 * in the order shown, how to step to another, and the way back.
 */
export type FormulaTemplateCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the template was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];

/**
 * One formula template as a biller reads it, opened from a table's row: whether it prices
 * shipments now, its tariff source coloured from the rules module's tokens, what it bills (each
 * charge and total clause the checker outlines), the inputs a contract sets with their
 * defaults, the bill it gives the sample load, what stands between a draft and approval, and how
 * it got here. Editing, testing and approving stay in the Formula Studio, one click away.
 *
 * The card steps through the table's rows in the order the table shows them. A step draws the
 * next template's skeleton until it is read, never the one it left.
 */
export function DeskFormulaTemplate({
  recordId,
  order = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: FormulaTemplateCardProps) {
  const t = useT();
  const query = useQuery({ ...formulaTemplate.get(recordId), retry: false });
  const template = query.data;

  if (!template) {
    return query.isError ? (
      <RecordCard>
        <CardHead
          onBack={onBack}
          backLabel={backLabel ?? t("Back to the table")}
          kicker={[t("Formula template")]}
          title={t("This formula template could not be read")}
          subtitle={describeApiError(
            query.error,
            t("It may have been removed, or you may not be allowed to see it."),
          )}
        />
      </RecordCard>
    ) : (
      <DeskFormulaTemplateSkeleton stepper={onSelect !== undefined && order.length > 1} />
    );
  }

  return (
    <FormulaCard
      key={template.id ?? recordId}
      template={template}
      recordId={template.id ?? recordId}
      order={order}
      onSelect={onSelect}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/**
 * The template as an artifact of its own, when the agent looked it up: the same card, with
 * nothing to step through and nowhere to go back to.
 */
export function DeskFormulaTemplateArtifact({ recordId }: RecordArtifactProps) {
  return <DeskFormulaTemplate recordId={recordId} />;
}

function FormulaCard({
  template,
  recordId,
  order,
  onSelect,
  onBack,
  backLabel,
}: Required<Pick<FormulaTemplateCardProps, "order">> &
  Omit<FormulaTemplateCardProps, "order"> & { template: FormulaTemplate }) {
  const t = useT();
  // The card reads the tariff's outline with the rules module and colours its source from the
  // module's tokens; it suspends behind the card's skeleton until the module has loaded.
  const rules = useRules();
  const { context } = useTariffCheckContext();
  const userId = useAuthStore((state) => state.user?.id);
  const { allowed: canCreate } = usePermission(Resource.FormulaTemplate, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.FormulaTemplate, Operation.Update);
  // The server rates a sample load for those who may write a template, as the studio does.
  const canPrice = canCreate || canUpdate;
  const stage = formulaStage(template.status);
  const studioPath = recordPath("formula_template", recordId);

  const check = useMemo(
    () => checkTariffSource(rules, template.source, "tariff", context),
    [rules, template.source, context],
  );
  const outline = check.outline?.items ?? [];
  const charges = outline.filter((item) => item.kind === "charge");
  const totals = outline.filter((item) => item.kind === "total");

  const price = useQuery({
    ...formulaTemplate.preview({ source: template.source }),
    enabled: canPrice && template.source.trim() !== "",
    retry: false,
    staleTime: 5 * 60_000,
  });
  const { data: usage } = useQuery(formulaTemplate.usage(recordId));
  const { data: versions } = useQuery(formulaTemplate.versions(recordId, VERSION_ROWS));
  const { data: readiness } = useQuery({
    ...formulaTemplate.readiness(recordId),
    enabled: stage === "draft" || stage === "review",
  });
  const neighbors = neighborsIn(order, recordId);
  const { copy, isCopied } = useCopyToClipboard();

  const usageText = usage
    ? usage.inUse
      ? t("Used by {0}", formatList(usage.usages.map((entry) => usageLabel(entry, t))))
      : t("Not used yet")
    : "";

  return (
    <RecordCard className={cn("dk-ft", `dk-ft-${stage}`)}>
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={[
          t("Tariff"),
          ...(template.currentVersionNumber ? [t("v{0}", template.currentVersionNumber)] : []),
          ...(template.imports.length > 0
            ? [
                t(
                  "{0, plural, one {# library} other {# libraries}}",
                  template.imports.length,
                ),
              ]
            : []),
        ]}
        stepper={
          onSelect &&
          order.length > 1 && (
            <CardStepper
              neighbors={neighbors}
              onStep={onSelect}
              previousLabel={t("Previous template")}
              nextLabel={t("Next template")}
            />
          )
        }
        title={template.name}
        subtitle={template.description || undefined}
        figure={canPrice ? <SampleTotalFigure result={price.data} pending={price.isPending} /> : null}
        meta={
          <>
            <CardStatus
              key={template.status}
              tone={STAGE_TONE[stage]}
              mark={stage === "active" ? "check" : "dot"}
            >
              {formulaStatusLabel(template.status, t)}
            </CardStatus>
            <CardFacts>
              {[
                formulaStatusNote(template.status, t),
                template.updatedAt ? t("Updated {0}", cardDay(template.updatedAt)) : "",
                usageText,
              ]}
            </CardFacts>
          </>
        }
      />

      <CardSection
        title={t("Source")}
        aside={
          <Button
            variant="quiet"
            size="xs"
            className="text-dsk-subtle"
            onClick={() => void copy(template.source)}
          >
            <CardIcon name={isCopied ? "check" : "copy"} size={12} />
            {isCopied ? t("Copied") : t("Copy")}
          </Button>
        }
      >
        <TariffCode code={template.source} className="dk-ft-code" />
      </CardSection>

      {(charges.length > 0 || totals.length > 0) && (
        <CardSection
          title={t("Bills")}
          aside={
            <span>{t("{0, plural, one {# charge} other {# charges}}", charges.length)}</span>
          }
        >
          <div className="dk-ft-items">
            {charges.map((item) => (
              <div key={item.range.from} className="dk-ft-item">
                <span style={{ paddingLeft: `${item.depth * 12}px` }}>
                  <b>{item.name}</b>
                  {item.depth > 0 && <em className="dk-ft-when">{t("when it applies")}</em>}
                </span>
                {item.code && <code className="dk-ft-n">{item.code}</code>}
              </div>
            ))}
            {totals.map((item) => (
              <div key={item.range.from} className="dk-ft-item">
                <span>{totalClauseLabel(item, t)}</span>
                <code className="dk-ft-n">
                  {template.source.slice(item.range.from, item.range.to).trim()}
                </code>
              </div>
            ))}
          </div>
        </CardSection>
      )}

      {template.inputs.length > 0 && (
        <CardSection
          title={t("Inputs")}
          aside={
            <span>{t("{0, plural, one {# input} other {# inputs}}", template.inputs.length)}</span>
          }
        >
          <div className="dk-ft-vars">
            {template.inputs.map((input) => (
              <div key={input.name} className="dk-ft-v">
                <div className="dk-ft-vn">
                  <span>
                    <code>{input.name}</code>
                    {input.required && <em>{t("Required")}</em>}
                  </span>
                  <span>{[input.type, input.doc].filter(Boolean).join(" · ")}</span>
                </div>
                <span className="dk-ft-vd">
                  {input.required || input.default === undefined
                    ? t("Set by each contract")
                    : formatTariffValue(input.default, t)}
                </span>
              </div>
            ))}
          </div>
        </CardSection>
      )}

      {canPrice ? (
        <CardSection title={t("Sample bill")} aside={<span>{t("Sample load, default inputs")}</span>}>
          <SampleBill result={price.data} error={price.error} pending={price.isPending} />
        </CardSection>
      ) : (
        <CardSection title={t("Sample bill")}>
          <p className="dk-ft-note">
            {t("Rating a sample load needs permission to write formula templates.")}
          </p>
        </CardSection>
      )}

      {readiness && (stage === "draft" || stage === "review") && (
        <CardSection
          title={t("Before it can be approved")}
          aside={
            readiness.testTotal > 0 ? (
              <span className={readiness.testPassed === readiness.testTotal ? "dk-ok" : "dk-warn"}>
                {t("{0} of {1} tests pass", readiness.testPassed, readiness.testTotal)}
              </span>
            ) : undefined
          }
        >
          <CardChecks>
            {readiness.checks.map((item) => (
              <CardCheck
                key={item.key}
                state={READINESS_STATE[item.status]}
                title={item.label}
                detail={item.detail ?? ""}
              />
            ))}
          </CardChecks>
        </CardSection>
      )}

      {versions && versions.results.length > 0 && (
        <CardSection
          title={t("Versions")}
          aside={
            (versions.totalCount ?? versions.count) > versions.results.length ? (
              <CardLink to={studioPath}>{t("All versions")}</CardLink>
            ) : undefined
          }
        >
          <CardTimeline
            entries={[...versions.results]
              .sort((a, b) => a.versionNumber - b.versionNumber)
              .map((version) => ({
                id: version.id,
                text: t("v{0} · {1}", version.versionNumber, version.changeMessage || t("No note")),
                meta: [version.createdBy?.name, cardMoment(version.createdAt)]
                  .filter(Boolean)
                  .join(" · "),
                mine: version.createdById === userId,
              }))}
          />
        </CardSection>
      )}

      <CardBar>
        <CardBarHint>{t("Edit, test and approve it in the Formula Studio")}</CardBarHint>
        <Link className="dk-ax-btn dk-ink" to={studioPath}>
          <CardIcon name="ext" size={13} />
          {t("Open in Formula Studio")}
        </Link>
      </CardBar>
    </RecordCard>
  );
}

/** Why the sample load got no bill: the tariff's refusal, the error, or the check's finding. */
function failureText(result: FormulaPreviewResponse, t: ReturnType<typeof useT>): string {
  if (result.refusal) return refusalText(result.refusal, t);
  if (result.error) return translateMessage(result.error.message, t);
  const problem = result.diagnostics.find((finding) => finding.severity === "error");
  return problem ? translateMessage(problem.message, t) : t("Does not rate");
}

/** The sample total at the head of the card. */
function SampleTotalFigure({
  result,
  pending,
}: {
  result: FormulaPreviewResponse | undefined;
  pending: boolean;
}) {
  const t = useT();
  if (pending || !result) {
    return (
      <>
        <b className="dk-ft-wait">—</b>
        <span>{t("on a sample load")}</span>
      </>
    );
  }
  return result.bill ? (
    <>
      <b key={result.bill.total.$q}>{formatTariffValue(result.bill.total, t)}</b>
      <span>{t("on a sample load")}</span>
    </>
  ) : (
    <>
      <b className="dk-ft-wait">—</b>
      <span className="dk-ft-err">{t("Does not rate")}</span>
    </>
  );
}

/** The bill the sample load gets, line by line; or why it got none, in the tariff's words. */
function SampleBill({
  result,
  error,
  pending,
}: {
  result: FormulaPreviewResponse | undefined;
  error: unknown;
  pending: boolean;
}) {
  const t = useT();
  if (error && !result) {
    return (
      <div className="dk-ax-warn">
        <CardIcon name="warn" size={13} />
        <span>{describeApiError(error, t("The sample load could not be rated."))}</span>
      </div>
    );
  }
  if (pending || !result) {
    return (
      <div className="dk-ft-led dk-ft-led-wait" aria-busy>
        <i className="dk-sk" />
        <i className="dk-sk" />
      </div>
    );
  }
  if (!result.bill) {
    return (
      <div className="dk-ax-warn">
        <CardIcon name="warn" size={13} />
        <span>{failureText(result, t)}</span>
      </div>
    );
  }
  return <BillView bill={result.bill} />;
}
