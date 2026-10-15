import { buildKnownIdentifiers } from "@/components/formula-editor/known-identifiers";
import { DEFAULT_TEST_VALUES } from "@/components/formula-editor/test-data-editor";
import { recordPath } from "@/config/record-links";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { usePermission } from "@/hooks/use-permission";
import { describeApiError } from "@/lib/api-error-message";
import { formulaTestRequest } from "@/lib/formula-test-request";
import { formulaTemplate } from "@/lib/queries/formula-template";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type {
  FormulaTemplate,
  ReadinessCheck,
  TestExpressionRequest,
  TestExpressionResponse,
  VariableDefinition,
} from "@trenova/shared/types/formula-template";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useMemo, useState } from "react";
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
import { cardDay, cardMoment, money, neighborsIn } from "../../record-card/record-card-values";
import type { RecordArtifactProps } from "../registry";
import {
  formulaStage,
  formulaStatusLabel,
  formulaStatusNote,
  formulaTypeLabel,
  guardrailNote,
  isEditableVariable,
  rootOf,
  roundingLabel,
  roundingNote,
  sampleLoadSummary,
  tokenizeFormula,
  usageLabel,
  variableDraft,
  variableDrafts,
  variableOverrides,
  variableTypeLabel,
  variableValue,
  type FormulaStage,
  type FormulaToken,
} from "./formula-template-card";
import { DeskFormulaTemplateSkeleton } from "./formula-template-skeleton";

/** How long a person's typing settles before the engine prices it again. */
const PRICE_SETTLE_MS = 450;
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

const NO_REQUEST: TestExpressionRequest = { expression: "", schemaId: "shipment", variables: {} };

/**
 * One formula template as a biller reads it, opened from a table's row: what
 * it charges for and whether it prices shipments now, the formula with the
 * template's own variables set apart from the shipment's fields, the limits
 * and rounding that hold its price, what it comes to on a sample load with
 * the variables as the person sets them, what stands between a draft and
 * approval, and how it got here. It reads; editing, testing against a saved
 * shipment and approving stay in the Formula Studio, one click away.
 *
 * The card steps through the table's rows in the order the table shows them.
 * A step draws the next template's skeleton until it is read, never the one it left.
 */
/**
 * What the card is given: the template, and when it was opened from a table,
 * the table's rows in the order shown, how to step to another, and the way
 * back.
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
 * The template as an artifact of its own, when the agent looked it up: the
 * same card, with nothing to step through and nowhere to go back to.
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
  const userId = useAuthStore((state) => state.user?.id);
  const { allowed: canCreate } = usePermission(Resource.FormulaTemplate, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.FormulaTemplate, Operation.Update);
  // The engine prices for those who may write a formula, as the studio does.
  const canPrice = canCreate || canUpdate;
  const stage = formulaStage(template.status);
  const definitions = template.variableDefinitions;
  const studioPath = recordPath("formula_template", recordId);

  const [drafts, setDrafts] = useState<Record<string, string>>(() => variableDrafts(definitions));
  const [focus, setFocus] = useState<string | null>(null);
  const changed = definitions.some(
    (definition) => drafts[definition.name] !== variableDraft(definition),
  );
  const overrides = useMemo(() => variableOverrides(definitions, drafts), [definitions, drafts]);
  const settled = useDebounce(overrides, PRICE_SETTLE_MS);
  const request = useMemo(
    () => formulaTestRequest(template, { sampleValues: DEFAULT_TEST_VALUES, overrides: settled }),
    [template, settled],
  );

  const price = useQuery({
    ...formulaTemplate.price(request ?? NO_REQUEST),
    enabled: canPrice && request !== null,
    placeholderData: keepPreviousData,
    retry: false,
    staleTime: 5 * 60_000,
  });
  const { data: schema } = useQuery({
    ...formulaTemplate.schema(template.schemaId || "shipment"),
    staleTime: 10 * 60_000,
  });
  const { data: usage } = useQuery(formulaTemplate.usage(recordId));
  const { data: versions } = useQuery(formulaTemplate.versions(recordId, VERSION_ROWS));
  const { data: readiness } = useQuery({
    ...formulaTemplate.readiness(recordId),
    enabled: stage === "draft" || stage === "review",
  });

  const known = useMemo(() => buildKnownIdentifiers(schema, definitions), [schema, definitions]);
  const names = useMemo(
    () => new Set(definitions.map((definition) => definition.name)),
    [definitions],
  );
  const tokens = useMemo(
    () => tokenizeFormula(template.expression, known, names),
    [template.expression, known, names],
  );
  const neighbors = neighborsIn(order, recordId);
  const result = price.data;
  const { copy, isCopied } = useCopyToClipboard();

  const usageText = usage
    ? usage.inUse
      ? t(
          "Used by {0}",
          new Intl.ListFormat(undefined, { style: "long", type: "conjunction" }).format(
            usage.usages.map((entry) => usageLabel(entry, t)),
          ),
        )
      : t("Not used yet")
    : "";

  return (
    <RecordCard className={cn("dk-ft", `dk-ft-${stage}`)}>
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={[
          formulaTypeLabel(template.type, t),
          ...(template.currentVersionNumber ? [t("v{0}", template.currentVersionNumber)] : []),
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
        figure={canPrice ? <SamplePriceFigure result={result} pending={price.isPending} /> : null}
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
        title={t("Formula")}
        aside={
          <Button
            variant="quiet"
            size="xs"
            className="text-dsk-subtle"
            onClick={() => void copy(template.expression)}
          >
            <CardIcon name={isCopied ? "check" : "copy"} size={12} />
            {isCopied ? t("Copied") : t("Copy")}
          </Button>
        }
      >
        <FormulaCode tokens={tokens} focus={focus} onFocus={setFocus} label={t("Formula")} />
        <div className="dk-ft-legend" aria-hidden>
          <span className="dk-ft-var">{t("Template variable")}</span>
          <span className="dk-ft-field">{t("Shipment field")}</span>
          <span className="dk-ft-fn">{t("Function")}</span>
        </div>
        <dl className="dk-ft-rules">
          <div>
            <dt>{t("Minimum")}</dt>
            <dd>{template.minCharge != null ? money(template.minCharge) : t("None")}</dd>
          </div>
          <div>
            <dt>{t("Maximum")}</dt>
            <dd>{template.maxCharge != null ? money(template.maxCharge) : t("None")}</dd>
          </div>
          <div>
            <dt>{t("Rounding")}</dt>
            <dd>{roundingLabel(template.roundingMode, template.roundingPrecision, t)}</dd>
          </div>
        </dl>
      </CardSection>

      {template.breakdownDefinitions.length > 0 && (
        <CardSection title={t("Itemized as")}>
          <div className="dk-ft-items">
            {template.breakdownDefinitions.map((item) => {
              const line = result?.breakdown?.find((entry) => entry.name === item.name);
              return (
                <div key={item.name} className="dk-ft-item">
                  <span>
                    <b>{item.label || item.name}</b>
                    <FormulaCode
                      tokens={tokenizeFormula(item.expression, known, names)}
                      focus={focus}
                      onFocus={setFocus}
                      label={item.label || item.name}
                      inline
                    />
                  </span>
                  {line && (
                    <span className={cn("dk-ft-n", line.error && "dk-bad")}>
                      {line.error ? t("Error") : money(line.amount)}
                    </span>
                  )}
                </div>
              );
            })}
          </div>
        </CardSection>
      )}

      {definitions.length > 0 && (
        <CardSection
          title={t("Variables")}
          aside={
            changed ? (
              <Button
                variant="quiet"
                size="xs"
                className="text-dsk-subtle"
                onClick={() => setDrafts(variableDrafts(definitions))}
              >
                <CardIcon name="reset" size={11} stroke={2.2} />
                {t("Reset to defaults")}
              </Button>
            ) : (
              <span>
                {t("{0, plural, one {# variable} other {# variables}}", definitions.length)}
              </span>
            )
          }
        >
          <div className="dk-ft-vars">
            {definitions.map((definition) => (
              <VariableRow
                key={definition.name}
                definition={definition}
                draft={drafts[definition.name] ?? ""}
                editable={canPrice && isEditableVariable(definition.type)}
                focused={focus === definition.name}
                onFocus={setFocus}
                onChange={(value) =>
                  setDrafts((current) => ({ ...current, [definition.name]: value }))
                }
              />
            ))}
          </div>
          {!canPrice && (
            <p className="dk-ft-note">
              {t("Pricing a sample load needs permission to write formula templates.")}
            </p>
          )}
        </CardSection>
      )}

      {canPrice && (
        <CardSection
          title={t("Sample price")}
          aside={
            <span className="dk-ft-load">
              {price.isFetching && <i className="dk-ft-pulse" aria-hidden />}
              {sampleLoadSummary(DEFAULT_TEST_VALUES, t).join(" · ")}
            </span>
          }
        >
          <SamplePrice
            result={result}
            error={price.error}
            pending={price.isPending && request !== null}
            empty={request === null}
          />
        </CardSection>
      )}

      {readiness && (stage === "draft" || stage === "review") && (
        <CardSection
          title={t("Before it can be approved")}
          aside={
            readiness.scenarioTotal > 0 ? (
              <span
                className={
                  readiness.scenarioPassed === readiness.scenarioTotal ? "dk-ok" : "dk-warn"
                }
              >
                {t("{0} of {1} scenarios pass", readiness.scenarioPassed, readiness.scenarioTotal)}
              </span>
            ) : undefined
          }
        >
          <CardChecks>
            {readiness.checks.map((check) => (
              <CardCheck
                key={check.key}
                state={READINESS_STATE[check.status]}
                title={check.label}
                detail={check.detail ?? ""}
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

/**
 * A formula set as code, the template's variables, the shipment's fields and
 * the functions each in their own colour. Pointing at a variable marks it
 * here and in the list of variables, so a reader finds where each is used.
 */
function FormulaCode({
  tokens,
  focus,
  onFocus,
  label,
  inline = false,
}: {
  tokens: readonly FormulaToken[];
  focus: string | null;
  onFocus: (name: string | null) => void;
  label: string;
  inline?: boolean;
}) {
  const Tag = inline ? "code" : "pre";
  return (
    <Tag className={cn("dk-ft-code", inline && "dk-ft-code-inline")} aria-label={label}>
      {tokens.map((token, index) => {
        const name = token.kind === "var" ? rootOf(token.text) : null;
        return (
          <span
            // oxlint-disable-next-line react/no-array-index-key -- a formula's pieces never reorder
            key={index}
            className={cn(`dk-ft-${token.kind}`, name !== null && name === focus && "dk-on")}
            onMouseEnter={name !== null ? () => onFocus(name) : undefined}
            onMouseLeave={name !== null ? () => onFocus(null) : undefined}
          >
            {token.text}
          </span>
        );
      })}
    </Tag>
  );
}

function VariableRow({
  definition,
  draft,
  editable,
  focused,
  onFocus,
  onChange,
}: {
  definition: VariableDefinition;
  draft: string;
  editable: boolean;
  focused: boolean;
  onFocus: (name: string | null) => void;
  onChange: (value: string) => void;
}) {
  const t = useT();
  const invalid =
    editable && draft.trim() !== "" && variableValue(definition.type, draft) === undefined;
  return (
    <div
      className={cn("dk-ft-v", focused && "dk-on")}
      onMouseEnter={() => onFocus(definition.name)}
      onMouseLeave={() => onFocus(null)}
    >
      <div className="dk-ft-vn">
        <span>
          <code>{definition.name}</code>
          {definition.required && <em>{t("Required")}</em>}
        </span>
        <span>
          {[variableTypeLabel(definition.type, t), definition.description]
            .filter(Boolean)
            .join(" · ")}
        </span>
      </div>
      {editable ? (
        definition.type === "Boolean" ? (
          <span className="dk-ft-yn" role="group" aria-label={definition.name}>
            {(["true", "false"] as const).map((value) => (
              <Button
                key={value}
                variant="bare"
                size="bare"
                aria-pressed={draft === value}
                className={cn(draft === value && "dk-on")}
                onClick={() => onChange(value)}
              >
                {value === "true" ? t("Yes") : t("No")}
              </Button>
            ))}
          </span>
        ) : (
          <input
            className={cn("dk-ft-in", invalid && "dk-bad")}
            value={draft}
            inputMode={definition.type === "Number" ? "decimal" : undefined}
            placeholder={t("Default")}
            aria-label={definition.name}
            aria-invalid={invalid || undefined}
            title={invalid ? t("Not a number; the default is used") : undefined}
            onFocus={() => onFocus(definition.name)}
            onBlur={() => onFocus(null)}
            onChange={(event) => onChange(event.target.value)}
          />
        )
      ) : (
        <span className="dk-ft-vd">{draft || "—"}</span>
      )}
    </div>
  );
}

/** The sample price at the head of the card, moving as the variables do. */
function SamplePriceFigure({
  result,
  pending,
}: {
  result: TestExpressionResponse | undefined;
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
  return result.valid ? (
    <>
      <b key={String(result.result)}>{money(result.result)}</b>
      <span>{t("on a sample load")}</span>
    </>
  ) : (
    <>
      <b className="dk-ft-wait">—</b>
      <span className="dk-ft-err">{t("Does not price")}</span>
    </>
  );
}

/**
 * What the formula comes to on the sample load: each itemized line, the
 * total, and what the limits and rounding did to it; or why it could not be
 * priced, in the engine's words.
 */
function SamplePrice({
  result,
  error,
  pending,
  empty,
}: {
  result: TestExpressionResponse | undefined;
  error: unknown;
  pending: boolean;
  empty: boolean;
}) {
  const t = useT();
  if (empty) {
    return <p className="dk-ft-note">{t("This template has no formula to price.")}</p>;
  }
  if (error && !result) {
    return (
      <div className="dk-ax-warn">
        <CardIcon name="warn" size={13} />
        <span>{describeApiError(error, t("The sample load could not be priced."))}</span>
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
  if (!result.valid) {
    return (
      <div className="dk-ax-warn">
        <CardIcon name="warn" size={13} />
        <span>{result.error || result.message}</span>
      </div>
    );
  }

  const notes = [
    guardrailNote(result.guardrail, t),
    result.rounding?.applied ? roundingNote(result.rounding.unroundedAmount, t) : "",
  ].filter((note) => note !== "");

  return (
    <>
      <div className="dk-ft-led">
        {(result.breakdown ?? []).map((line, index) => (
          <div
            key={line.name}
            className={cn("dk-ft-lr", line.error && "dk-bad")}
            style={{ animationDelay: `${index * 40}ms` }}
          >
            <span>{line.label || line.name}</span>
            <span className="dk-ft-n">{line.error ? line.error : money(line.amount)}</span>
          </div>
        ))}
        <div className="dk-ft-lt">
          <span>{t("Total")}</span>
          <span className="dk-ft-n" key={String(result.result)}>
            {money(result.result)}
          </span>
        </div>
      </div>
      {notes.map((note) => (
        <p key={note} className="dk-ft-note">
          {note}
        </p>
      ))}
      {(result.warnings ?? []).map((warning) => (
        <div key={`${warning.scope}:${warning.field}`} className="dk-ax-warn">
          <CardIcon name="warn" size={13} />
          <span>
            {warning.message}
            {warning.suggestion ? ` ${warning.suggestion}` : ""}
          </span>
        </div>
      ))}
    </>
  );
}
