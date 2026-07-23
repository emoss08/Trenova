import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { FormSection, FormGroup, FormControl } from "@/components/ui/form";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { cn } from "@/lib/utils";
import {
  Controller,
  useFormContext,
  useWatch,
  type Control,
  type FieldValues,
  type Path,
} from "react-hook-form";
import { TagInput } from "../shared/tag-input";
import { KeyValueEditor } from "../shared/key-value-editor";
import type {
  FixtureFieldAssertion,
  FixtureFieldAssertionOperator,
  FixtureFormValues,
} from "@/types/document-parsing-rule";
import { PlusIcon, XIcon } from "lucide-react";
import { useCallback, useMemo } from "react";

const REVIEW_STATUS_OPTIONS = [
  { value: "", label: "Not Set" },
  { value: "Ready", label: "Ready" },
  { value: "NeedsReview", label: "Needs Review" },
  { value: "Unavailable", label: "Unavailable" },
];

const FIELD_ASSERTION_OPERATOR_OPTIONS: Array<{
  value: FixtureFieldAssertionOperator;
  label: string;
  description: string;
}> = [
  {
    value: "exists",
    label: "Exists",
    description: "Field key is present in the extraction result.",
  },
  {
    value: "not_empty",
    label: "Not Empty",
    description: "Field value must be present and non-blank.",
  },
  {
    value: "equals",
    label: "Equals",
    description: "Field value must exactly match the provided string.",
  },
  {
    value: "matches_regex",
    label: "Matches Regex",
    description: "Field value must match the provided regex pattern.",
  },
  {
    value: "one_of",
    label: "One Of",
    description: "Field value must equal one of the provided candidates.",
  },
];

const EMPTY_FIELD_ASSERTION: FixtureFieldAssertion = {
  operator: "not_empty",
  value: "",
  values: [],
  pattern: "",
};

export function AssertionsEditor() {
  const { control } = useFormContext<FixtureFormValues>();

  const expectedFields = useWatch({ control, name: "assertions.expectedFields" });
  const fieldAssertions = useWatch({
    control,
    name: "assertions.fieldAssertions",
  });
  const requiredStopRoles = useWatch({ control, name: "assertions.requiredStopRoles" });
  const minimumStopCount = useWatch({ control, name: "assertions.minimumStopCount" });

  const summary = useMemo(() => {
    const parts: string[] = [];
    const ruleCount = Object.values(fieldAssertions ?? {}).reduce(
      (total, assertions) => total + (assertions?.length ?? 0),
      0,
    );
    if (ruleCount > 0) {
      parts.push(`${ruleCount} field assertion${ruleCount !== 1 ? "s" : ""}`);
    }
    const legacyFieldCount = Object.keys(expectedFields ?? {}).length;
    if (legacyFieldCount > 0) {
      parts.push(
        `${legacyFieldCount} legacy exact match${legacyFieldCount !== 1 ? "es" : ""}`,
      );
    }
    const roleCount = (requiredStopRoles ?? []).length;
    if (roleCount > 0) {
      parts.push(`${roleCount} required role${roleCount !== 1 ? "s" : ""}`);
    }
    if (minimumStopCount && minimumStopCount > 0) {
      parts.push(`min ${minimumStopCount} stop${minimumStopCount !== 1 ? "s" : ""}`);
    }
    return parts;
  }, [expectedFields, fieldAssertions, requiredStopRoles, minimumStopCount]);

  return (
    <FormSection
      title="Assertions"
      description="Define the expected extraction results for this fixture. Use field assertions for presence, shape, and format checks; keep exact-value matches only for legacy or intentionally strict fixtures."
      action={
        summary.length > 0 ? (
          <div className="flex items-center gap-1.5">
            {summary.map((s) => (
              <Badge key={s} variant="outline" className="font-normal">
                {s}
              </Badge>
            ))}
          </div>
        ) : undefined
      }
    >
      <div className="space-y-4">
        <div>
          <FieldAssertionsEditor
            control={control}
            name="assertions.fieldAssertions"
            label="Field Assertions"
            description="Recommended for production fixtures. Assert that fields exist, are non-empty, match regexes, or match one of several acceptable values."
          />
        </div>

        <Separator />

        <div>
          <KeyValueEditor
            control={control}
            name="assertions.expectedFields"
            label="Legacy Exact Matches"
            description="Optional strict assertions for fixtures that intentionally require an exact extracted value. Prefer field assertions above for general template validation."
            keyPlaceholder="Field key (e.g. referenceNumber)"
            valuePlaceholder="Expected value"
          />
        </div>

        <Separator />

        <FormGroup cols={2}>
          <FormControl>
            <TagInput
              control={control}
              name="assertions.requiredStopRoles"
              label="Required Stop Roles"
              description="Stop roles that must appear in the extraction result (e.g. pickup, delivery)."
              placeholder="Add role..."
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="assertions.minimumStopCount"
              label="Minimum Stop Count"
              description="The minimum number of stops the parser must extract for this fixture to pass."
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="assertions.reviewStatus"
              label="Review Status"
              description="Track whether this fixture's assertions have been verified as correct."
              options={REVIEW_STATUS_OPTIONS}
            />
          </FormControl>
        </FormGroup>
      </div>
    </FormSection>
  );
}

type FieldAssertionsEditorProps<T extends FieldValues> = {
  control: Control<T>;
  name: Path<T>;
  label?: string;
  description?: string;
  disabled?: boolean;
};

function FieldAssertionsEditor<T extends FieldValues>({
  control,
  name,
  label,
  description,
  disabled,
}: FieldAssertionsEditorProps<T>) {
  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => (
        <FieldAssertionsEditorInner
          value={
            (field.value as Record<string, FixtureFieldAssertion[]> | undefined) ??
            {}
          }
          onChange={field.onChange}
          label={label}
          description={description}
          disabled={disabled}
          error={fieldState.error?.message}
        />
      )}
    />
  );
}

function FieldAssertionsEditorInner({
  value,
  onChange,
  label,
  description,
  disabled,
  error,
}: {
  value: Record<string, FixtureFieldAssertion[]>;
  onChange: (value: Record<string, FixtureFieldAssertion[]>) => void;
  label?: string;
  description?: string;
  disabled?: boolean;
  error?: string;
}) {
  const entries = Object.entries(value);

  const addField = useCallback(() => {
    let nextIndex = entries.length + 1;
    let nextKey = `field_${nextIndex}`;
    while (Object.hasOwn(value, nextKey)) {
      nextIndex++;
      nextKey = `field_${nextIndex}`;
    }
    onChange({ ...value, [nextKey]: [{ ...EMPTY_FIELD_ASSERTION }] });
  }, [entries.length, onChange, value]);

  const removeField = useCallback(
    (fieldKey: string) => {
      const next = Object.fromEntries(
        Object.entries(value).filter(([currentKey]) => currentKey !== fieldKey),
      );
      onChange(next);
    },
    [onChange, value],
  );

  const renameField = useCallback(
    (oldKey: string, newKey: string) => {
      if (oldKey === newKey) return;

      const next: Record<string, FixtureFieldAssertion[]> = {};
      const renamedAssertions = value[oldKey] ?? [];
      for (const [currentKey, assertions] of Object.entries(value)) {
        if (currentKey === oldKey) {
          continue;
        }
        if (currentKey === newKey) {
          next[currentKey] = [...assertions, ...renamedAssertions];
          continue;
        }
        next[currentKey] = assertions;
      }
      if (!Object.hasOwn(next, newKey)) {
        next[newKey] = renamedAssertions;
      }
      onChange(next);
    },
    [onChange, value],
  );

  const addAssertion = useCallback(
    (fieldKey: string) => {
      onChange({
        ...value,
        [fieldKey]: [...(value[fieldKey] ?? []), { ...EMPTY_FIELD_ASSERTION }],
      });
    },
    [onChange, value],
  );

  const removeAssertion = useCallback(
    (fieldKey: string, index: number) => {
      const nextAssertions = (value[fieldKey] ?? []).filter(
        (_, currentIndex) => currentIndex !== index,
      );

      if (nextAssertions.length === 0) {
        removeField(fieldKey);
        return;
      }

      onChange({
        ...value,
        [fieldKey]: nextAssertions,
      });
    },
    [onChange, removeField, value],
  );

  const updateAssertion = useCallback(
    (
      fieldKey: string,
      index: number,
      nextAssertion: FixtureFieldAssertion,
    ) => {
      const nextAssertions = [...(value[fieldKey] ?? [])];
      nextAssertions[index] = nextAssertion;
      onChange({
        ...value,
        [fieldKey]: nextAssertions,
      });
    },
    [onChange, value],
  );

  return (
    <div className="space-y-1.5">
      {label && <label className="text-sm leading-none font-medium">{label}</label>}
      <div className="space-y-3">
        {entries.length === 0 && (
          <div className="rounded-md border border-dashed px-3 py-4 text-xs text-muted-foreground">
            No field assertions yet. Add rules for things like “referenceNumber
            is not empty” or “rate matches a currency pattern.”
          </div>
        )}

        {entries.map(([fieldKey, assertions]) => (
          <div key={fieldKey} className="space-y-3 rounded-md border p-3">
            <div className="flex items-center gap-2">
              <Input
                value={fieldKey}
                onChange={(event) => renameField(fieldKey, event.target.value)}
                placeholder="Field key (e.g. referenceNumber)"
                disabled={disabled}
                className="flex-1"
              />
              {!disabled && (
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  onClick={() => removeField(fieldKey)}
                  aria-label={`Remove ${fieldKey} assertions`}
                >
                  <XIcon className="size-4" />
                </Button>
              )}
            </div>

            <div className="space-y-2">
              {(assertions ?? []).map((assertion, index) => (
                <FieldAssertionRow
                  key={`${fieldKey}-${index}`}
                  assertion={assertion}
                  onChange={(nextAssertion) =>
                    updateAssertion(fieldKey, index, nextAssertion)
                  }
                  onRemove={() => removeAssertion(fieldKey, index)}
                  disabled={disabled}
                />
              ))}
            </div>

            {!disabled && (
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="gap-1"
                onClick={() => addAssertion(fieldKey)}
              >
                <PlusIcon className="size-3.5" />
                Add Assertion
              </Button>
            )}
          </div>
        ))}

        {!disabled && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="gap-1"
            onClick={addField}
          >
            <PlusIcon className="size-3.5" />
            Add Field Assertion
          </Button>
        )}
      </div>
      {(error || description) && (
        <p
          className={cn(
            "text-2xs",
            error ? "text-destructive" : "text-muted-foreground",
          )}
        >
          {error || description}
        </p>
      )}
    </div>
  );
}

function FieldAssertionRow({
  assertion,
  onChange,
  onRemove,
  disabled,
}: {
  assertion: FixtureFieldAssertion;
  onChange: (assertion: FixtureFieldAssertion) => void;
  onRemove: () => void;
  disabled?: boolean;
}) {
  const operatorMeta = FIELD_ASSERTION_OPERATOR_OPTIONS.find(
    (option) => option.value === assertion.operator,
  );

  const handleOperatorChange = (nextOperator: FixtureFieldAssertionOperator) => {
    onChange({
      operator: nextOperator,
      value: nextOperator === "equals" ? assertion.value ?? "" : "",
      values: nextOperator === "one_of" ? assertion.values ?? [] : [],
      pattern:
        nextOperator === "matches_regex" ? assertion.pattern ?? "" : "",
    });
  };

  return (
    <div className="rounded-md border bg-muted/30 p-3">
      <div className="flex items-start gap-2">
        <div className="grid flex-1 gap-2 md:grid-cols-[180px_minmax(0,1fr)]">
          <div className="space-y-1">
            <label className="text-2xs font-medium text-muted-foreground uppercase">
              Operator
            </label>
            <select
              value={assertion.operator}
              onChange={(event) =>
                handleOperatorChange(
                  event.target.value as FixtureFieldAssertionOperator,
                )
              }
              disabled={disabled}
              className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm outline-none"
            >
              {FIELD_ASSERTION_OPERATOR_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </div>

          <div className="space-y-1">
            <label className="text-2xs font-medium text-muted-foreground uppercase">
              {assertion.operator === "matches_regex"
                ? "Pattern"
                : assertion.operator === "one_of"
                  ? "Accepted Values"
                  : assertion.operator === "equals"
                    ? "Expected Value"
                    : "Details"}
            </label>
            {assertion.operator === "equals" && (
              <Input
                value={assertion.value}
                onChange={(event) =>
                  onChange({ ...assertion, value: event.target.value })
                }
                placeholder="Expected exact value"
                disabled={disabled}
              />
            )}
            {assertion.operator === "matches_regex" && (
              <Input
                value={assertion.pattern}
                onChange={(event) =>
                  onChange({ ...assertion, pattern: event.target.value })
                }
                placeholder="e.g. ^\\$[0-9]+(?:\\.[0-9]{2})$"
                disabled={disabled}
                className="font-mono text-xs"
              />
            )}
            {assertion.operator === "one_of" && (
              <Input
                value={(assertion.values ?? []).join(", ")}
                onChange={(event) =>
                  onChange({
                    ...assertion,
                    values: event.target.value
                      .split(",")
                      .map((value) => value.trim())
                      .filter(Boolean),
                  })
                }
                placeholder="Comma-separated acceptable values"
                disabled={disabled}
              />
            )}
            {(assertion.operator === "exists" ||
              assertion.operator === "not_empty") && (
              <div className="flex h-9 items-center rounded-md border border-dashed px-3 text-xs text-muted-foreground">
                {operatorMeta?.description}
              </div>
            )}
          </div>
        </div>

        {!disabled && (
          <Button
            type="button"
            variant="ghost"
            size="icon"
            onClick={onRemove}
            aria-label="Remove assertion"
          >
            <XIcon className="size-4" />
          </Button>
        )}
      </div>

      {operatorMeta && (
        <p className="mt-2 text-2xs text-muted-foreground">
          {operatorMeta.description}
        </p>
      )}
    </div>
  );
}
