import {
  FormulaTemplateAutocompleteField,
  RateMatrixAutocompleteField,
} from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { rateDirectionChoices } from "@/lib/choices";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import {
  findCoverageIssues,
  laneDisplayLabel,
  laneKeyPreview,
  laneSpecificity,
} from "@trenova/shared/lib/rate";
import type { RateAgreement, RateAgreementRule } from "@trenova/shared/types/rate";
import { PlusIcon, TriangleAlertIcon } from "lucide-react";
import { useMemo } from "react";
import type { Control } from "react-hook-form";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";
import { LaneScopeFields } from "./lane-scope-fields";

const NEW_LANE: RateAgreementRule = {
  label: "",
  status: "Active",
  originScopeType: "Any",
  originScopeValue: "",
  originCity: "",
  destinationScopeType: "Any",
  destinationScopeValue: "",
  destinationCity: "",
  direction: "Directional",
  freightClassSource: "Commodity",
  allowDeficitRating: true,
  daysOfWeek: 0,
  hazmatOnly: false,
  tempControlOnly: false,
  priority: 0,
  effectiveFrom: 0,
  breaks: [],
} as RateAgreementRule;

/**
 * The lane grid, which is where the whole contract actually lives.
 *
 * Lanes are shown with the score the engine will rank them by and the key it
 * will match them on, because "why did the other rate apply" is nearly always
 * answered by one of those two numbers. Lanes that can never fire are called
 * out as they are typed rather than discovered from an invoice weeks later.
 */
export function LaneEditor() {
  const { control } = useFormContext<RateAgreement>();
  const { fields, append, remove } = useFieldArray({ control, name: "rules" });
  const rules = (useWatch({ control, name: "rules" }) ?? []) as RateAgreementRule[];

  // `rules` is a watched array and so is a fresh reference on every render.
  // Keying the memo on what coverage actually depends on means it recomputes
  // when a lane changes rather than on every keystroke elsewhere in the form.
  const coverageKey = rules
    .map(
      (rule) =>
        `${rule.status}|${laneKeyPreview(rule) ?? "radius"}|${laneSpecificity(rule)}|${rule.priority ?? 0}`,
    )
    .join(";");

  // eslint-disable-next-line react-hooks/exhaustive-deps
  const issues = useMemo(() => findCoverageIssues(rules), [coverageKey]);
  const issueByIndex = useMemo(
    () => new Map(issues.map((issue) => [issue.index, issue])),
    [issues],
  );

  const appendLane = () => {
    append({
      ...NEW_LANE,
      effectiveFrom: Math.floor(Date.now() / 1000),
    });
  };

  return (
    <div className="flex flex-col gap-4">
      {fields.length === 0 && (
        <p className="text-muted-foreground text-sm">
          No lanes yet. An active agreement needs at least one, otherwise nothing can price against
          it.
        </p>
      )}

      {issues.length > 0 && (
        <Alert variant="destructive">
          <TriangleAlertIcon className="size-4" />
          <AlertTitle>
            {issues.length === 1 ? "One lane" : `${issues.length} lanes`} can never apply
          </AlertTitle>
          <AlertDescription>
            A lane written exactly as narrowly as another one leaves the winner to a tie-break.
            Narrow it, widen the other, or give one a higher priority.
          </AlertDescription>
        </Alert>
      )}

      {fields.map((field, index) => (
        <LaneRow
          key={field.id}
          control={control}
          index={index}
          rule={rules[index]}
          issue={issueByIndex.get(index)?.message}
          onRemove={() => remove(index)}
        />
      ))}

      <Button type="button" variant="outline" size="sm" onClick={appendLane}>
        <PlusIcon className="mr-1 size-3.5" />
        Add lane
      </Button>
    </div>
  );
}

type LaneRowProps = {
  control: Control<RateAgreement>;
  index: number;
  rule?: RateAgreementRule;
  issue?: string;
  onRemove: () => void;
};

function LaneRow({ control, index, rule, issue, onRemove }: LaneRowProps) {
  // A lane prices through exactly one of the two, so whichever is chosen hides
  // the other — clearing the selection brings the alternative back.
  const usesMatrix = Boolean(rule?.rateMatrixId);
  const usesFormula = Boolean(rule?.formulaTemplateId);
  const laneKey = rule ? laneKeyPreview(rule) : null;

  return (
    <div className="rounded-md border bg-card p-4">
      <div className="mb-3 flex items-start justify-between gap-2">
        <div className="flex flex-col gap-1">
          <p className="text-sm font-medium">
            {rule ? laneDisplayLabel(rule, index) : `Lane ${index + 1}`}
          </p>
          <div className="flex flex-wrap items-center gap-1.5">
            <Badge variant="outline" className="font-mono text-[10px]">
              {laneKey ?? "matched by radius"}
            </Badge>
            {rule && (
              <Badge variant="secondary" className="text-[10px]">
                specificity {laneSpecificity(rule)}
              </Badge>
            )}
          </div>
        </div>
        <Button type="button" variant="ghost" size="sm" className="h-6 text-xs" onClick={onRemove}>
          Remove
        </Button>
      </div>

      {issue && (
        <p className="mb-3 rounded-sm bg-destructive/10 p-2 text-xs text-destructive">{issue}</p>
      )}

      <FormGroup cols={2}>
        <FormControl>
          <InputField
            control={control}
            name={`rules.${index}.label` as never}
            label="Label"
            placeholder="Dallas to Chicago"
            description="What this lane is called on a rate confirmation and in a trace"
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name={`rules.${index}.direction` as never}
            label="Direction"
            placeholder="Direction"
            description="Whether the lane also prices the return trip"
            options={rateDirectionChoices}
          />
        </FormControl>
      </FormGroup>

      <div className="mt-3 grid gap-3 md:grid-cols-2">
        <div className="rounded-md border bg-muted/30 p-3">
          <p className="text-muted-foreground mb-2 text-xs font-medium tracking-wide uppercase">
            Origin
          </p>
          <LaneScopeFields control={control} side="origin" namePrefix={`rules.${index}.`} />
        </div>
        <div className="rounded-md border bg-muted/30 p-3">
          <p className="text-muted-foreground mb-2 text-xs font-medium tracking-wide uppercase">
            Destination
          </p>
          <LaneScopeFields control={control} side="destination" namePrefix={`rules.${index}.`} />
        </div>
      </div>

      <FormGroup cols={3} className="mt-3">
        {!usesMatrix && (
          <FormControl>
            <FormulaTemplateAutocompleteField
              control={control}
              rules={{ required: !usesMatrix }}
              name={`rules.${index}.formulaTemplateId` as never}
              label="Rating Method"
              placeholder="Select rating method"
              description="The formula template this lane prices through — the lane's rate binds in as the template's base rate"
            />
          </FormControl>
        )}
        {!usesMatrix && (
          <FormControl>
            <NumberField
              control={control}
              name={`rules.${index}.rate` as never}
              label="Rate"
              placeholder="2.55"
              sideText="$"
              decimalScale={4}
              thousandSeparator
              description="Feeds the rating method as its base rate — leave empty when the lane is banded by weight"
            />
          </FormControl>
        )}
        {!usesFormula && (
          <FormControl cols={usesMatrix ? 3 : undefined}>
            <RateMatrixAutocompleteField
              control={control}
              rules={{ required: !usesFormula }}
              name={`rules.${index}.rateMatrixId` as never}
              label="Rate Matrix"
              placeholder="Matrix"
              description="Reads the price from a grid instead of a rating method — the matrix's own rating method says what its cells mean"
            />
          </FormControl>
        )}
      </FormGroup>

      <FormGroup cols={4} className="mt-3">
        <FormControl>
          <NumberField
            control={control}
            name={`rules.${index}.minCharge` as never}
            label="Minimum Charge"
            placeholder="850.00"
            sideText="$"
            decimalScale={2}
            thousandSeparator
            description="The floor this lane never prices below"
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name={`rules.${index}.maxCharge` as never}
            label="Maximum Charge"
            placeholder="0.00"
            sideText="$"
            decimalScale={2}
            thousandSeparator
            description="A ceiling, when the contract sets one"
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name={`rules.${index}.minBillableDistance` as never}
            label="Minimum Billable Miles"
            placeholder="250"
            sideText="mi"
            description="Short hauls bill at this distance however far they actually ran"
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name={`rules.${index}.priority` as never}
            label="Priority"
            placeholder="0"
            description="Breaks a tie between lanes written equally narrowly"
          />
        </FormControl>
      </FormGroup>
    </div>
  );
}
