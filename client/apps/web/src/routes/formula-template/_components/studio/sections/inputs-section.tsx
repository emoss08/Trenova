import { coerceVariableDefaultValue } from "@/components/formula-editor/variable-definition-editor";
import { describeApiError } from "@/lib/api-error-message";
import { apiService } from "@/services/api";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Switch } from "@trenova/shared/components/ui/switch";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type {
  FormulaTemplateFormValues,
  VariableValueType,
} from "@trenova/shared/types/formula-template";
import { PlusIcon, Trash01Icon, VariableIcon } from "@trenova/shared/components/icons";
import { useMutation } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { Controller, useFieldArray, useFormContext, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { NovaButton } from "../ai/nova-button";
import { sanitizeIdentifier } from "../model/expression-analysis";
import { useStudio } from "../studio-context";
import { SectionHead } from "./section-head";

const INPUT_TYPES: readonly VariableValueType[] = ["Number", "String", "Boolean"];

function Tag({ tone, children }: { tone: "neutral" | "danger"; children: string }) {
  return (
    <span
      className={cn(
        "shrink-0 rounded-full px-1.5 text-2xs",
        tone === "danger"
          ? "bg-danger-subtle text-danger-subtle-foreground"
          : "bg-surface-hover text-foreground-muted",
      )}
    >
      {children}
    </span>
  );
}

function defaultText(value: unknown): string {
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return "";
}

/**
 * The values this template adds to the schema, as a table: name, type,
 * default, whether a lane must set it, and a line for what it means. Lanes
 * and rate agreements override each default.
 */
export function InputsSection() {
  const t = useT();
  const { known, analysis, draftContent } = useStudio();
  const form = useFormContext<FormulaTemplateFormValues>();
  const { control } = form;
  const { fields, append, remove } = useFieldArray({ control, name: "variableDefinitions" });
  const variables = useWatch({ control, name: "variableDefinitions" }) ?? [];
  const [described, setDescribed] = useState<ReadonlySet<string>>(() => new Set());

  const schemaNames = useMemo(
    () => new Set(known.variables.filter((variable) => !variable.custom).map((v) => v.name)),
    [known.variables],
  );
  const missing = variables.filter(
    (variable) => variable.name && !(variable.description ?? "").trim(),
  ).length;

  const describe = useMutation({
    mutationFn: () => apiService.formulaTemplateService.describeInputs(draftContent()),
    onSuccess: ({ descriptions }) => {
      const current = form.getValues("variableDefinitions") ?? [];
      const filled: string[] = [];
      current.forEach((variable, index) => {
        const text = descriptions[variable.name]?.trim();
        if (text && !(variable.description ?? "").trim()) {
          form.setValue(`variableDefinitions.${index}.description`, text, { shouldDirty: true });
          filled.push(variable.name);
        }
      });
      setDescribed((previous) => new Set([...previous, ...filled]));
    },
    onError: (error) => {
      toast.error(t("Nova could not describe the inputs"), {
        description: describeApiError(error),
      });
    },
  });

  const add = () =>
    append({ name: "", type: "Number", description: "", required: false, defaultValue: undefined });

  return (
    <section className="flex flex-col gap-3">
      <SectionHead
        section="inputs"
        icon={VariableIcon}
        title={t("Inputs")}
        description={t(
          "Values this template adds to the schema. Lanes and rate agreements can override each default.",
        )}
      >
        {missing > 0 && (
          <NovaButton
            busy={describe.isPending}
            disabled={describe.isPending}
            onClick={() => describe.mutate()}
          >
            {describe.isPending ? t("Writing…") : t("Describe {0}", missing)}
          </NovaButton>
        )}
        <Button type="button" variant="outline" size="xs" className="gap-1" onClick={add}>
          <PlusIcon className="size-3" />
          {t("Add input")}
        </Button>
      </SectionHead>

      {fields.length === 0 ? (
        <div className="text-foreground-muted flex items-center justify-between gap-3 rounded-lg border border-dashed px-3 py-2.5 text-sm">
          <span>
            {t(
              "No inputs. Shipment variables cover most methods; add one for a fee or percentage lanes should set.",
            )}
          </span>
          <Button type="button" variant="ghost" size="xs" className="gap-1" onClick={add}>
            <PlusIcon className="size-3" />
            {t("Add input")}
          </Button>
        </div>
      ) : (
        <div className="bg-card overflow-hidden rounded-lg border">
          <div className="bg-sunken text-foreground-subtle grid grid-cols-[minmax(0,1.6fr)_7rem_minmax(0,1fr)_4.5rem_1.75rem] items-center gap-2 border-b px-3 py-1.5 text-xs font-medium">
            <span>{t("Name")}</span>
            <span>{t("Type")}</span>
            <span>{t("Default")}</span>
            <span>{t("Required")}</span>
            <span className="sr-only">{t("Remove")}</span>
          </div>
          {fields.map((field, index) => {
            const variable = variables[index] ?? field;
            const name = variable.name ?? "";
            const unused = name !== "" && !analysis.inputReads.has(name);
            const taken =
              name !== "" &&
              (schemaNames.has(name) ||
                variables.filter((other) => other.name === name).length > 1);
            return (
              <div
                key={field.id}
                className="flex flex-col gap-1 border-b px-3 py-2 last:border-b-0"
              >
                <div className="grid grid-cols-[minmax(0,1.6fr)_7rem_minmax(0,1fr)_4.5rem_1.75rem] items-center gap-2">
                  <div className="flex min-w-0 items-center gap-1.5">
                    <Controller
                      control={control}
                      name={`variableDefinitions.${index}.name`}
                      render={({ field: nameField, fieldState }) => (
                        <Input
                          ref={nameField.ref}
                          value={nameField.value ?? ""}
                          onBlur={nameField.onBlur}
                          onChange={(event) =>
                            nameField.onChange(sanitizeIdentifier(event.target.value))
                          }
                          aria-label={t("Input name")}
                          aria-invalid={taken || fieldState.invalid || undefined}
                          placeholder="fuelSurchargePercent"
                          inputContainerClassName="min-w-0 flex-1"
                          className="font-mono"
                        />
                      )}
                    />
                    {unused && <Tag tone="neutral">{t("unused")}</Tag>}
                    {taken && <Tag tone="danger">{t("name taken")}</Tag>}
                  </div>
                  <Controller
                    control={control}
                    name={`variableDefinitions.${index}.type`}
                    render={({ field: typeField }) => (
                      <Select
                        value={
                          typeField.value && INPUT_TYPES.includes(typeField.value)
                            ? typeField.value
                            : "Number"
                        }
                        onValueChange={(next) => {
                          const type = next as VariableValueType;
                          typeField.onChange(type);
                          form.setValue(
                            `variableDefinitions.${index}.defaultValue`,
                            type === "Boolean" ? false : type === "String" ? "" : undefined,
                            { shouldDirty: true },
                          );
                        }}
                      >
                        <SelectTrigger className="w-full" aria-label={t("Input type")}>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {INPUT_TYPES.map((type) => (
                            <SelectItem key={type} value={type}>
                              {t(type)}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    )}
                  />
                  <Controller
                    control={control}
                    name={`variableDefinitions.${index}.defaultValue`}
                    render={({ field: defaultField }) =>
                      variable.type === "Boolean" ? (
                        <span className="flex items-center gap-2 text-xs">
                          <Switch
                            size="sm"
                            checked={defaultField.value === true}
                            onCheckedChange={(checked) => defaultField.onChange(checked)}
                            aria-label={t("Default")}
                          />
                          <span className="text-foreground-muted font-mono">
                            {defaultField.value === true ? "true" : "false"}
                          </span>
                        </span>
                      ) : (
                        <Input
                          value={defaultText(defaultField.value)}
                          onBlur={defaultField.onBlur}
                          onChange={(event) =>
                            defaultField.onChange(
                              variable.type === "Number"
                                ? coerceVariableDefaultValue("Number", event.target.value)
                                : event.target.value,
                            )
                          }
                          inputMode={variable.type === "Number" ? "decimal" : undefined}
                          aria-label={t("Default value")}
                          placeholder="—"
                          className="font-mono"
                        />
                      )
                    }
                  />
                  <Controller
                    control={control}
                    name={`variableDefinitions.${index}.required`}
                    render={({ field: requiredField }) => (
                      <Switch
                        size="sm"
                        checked={requiredField.value === true}
                        onCheckedChange={(checked) => requiredField.onChange(checked)}
                        aria-label={t("Required")}
                      />
                    )}
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-xs"
                    aria-label={t("Remove input {0}", name || t("untitled"))}
                    onClick={() => remove(index)}
                  >
                    <Trash01Icon className="size-3.5" />
                  </Button>
                </div>
                <Controller
                  control={control}
                  name={`variableDefinitions.${index}.description`}
                  render={({ field: descriptionField }) => (
                    <Textarea
                      bare
                      ref={descriptionField.ref}
                      value={descriptionField.value ?? ""}
                      onBlur={descriptionField.onBlur}
                      onChange={(event) => {
                        descriptionField.onChange(event.target.value);
                        if (described.has(name)) {
                          setDescribed((previous) => {
                            const next = new Set(previous);
                            next.delete(name);
                            return next;
                          });
                        }
                      }}
                      minRows={1}
                      aria-label={t("Input description")}
                      placeholder={t("Description lanes see when they override it")}
                      className={cn(
                        "placeholder:text-foreground-subtle px-0 text-xs md:text-xs",
                        described.has(name)
                          ? "text-accent-violet-on-subtle"
                          : "text-foreground-muted",
                      )}
                    />
                  )}
                />
              </div>
            );
          })}
        </div>
      )}
    </section>
  );
}
