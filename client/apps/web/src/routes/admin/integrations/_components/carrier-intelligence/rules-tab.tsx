import { Checkbox } from "@/components/animate-ui/components/base/checkbox";
import { MultiCheckboxField } from "@/components/fields/multi-checkbox-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { useCarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import type { CarrierIntelRuleDefinition } from "@/lib/graphql/carrier-intelligence";
import type { CarrierIntelRuleParamDefinition } from "@/lib/graphql/carrier-intel-settings";
import type {
  CarrierIntelRuleAction,
  CarrierIntelSection,
} from "@trenova/graphql/generated/graphql";
import { Badge } from "@trenova/shared/components/ui/badge";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Label } from "@trenova/shared/components/ui/label";
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
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { CSA_BASIC_LABELS } from "@trenova/shared/lib/csa";
import { cn, parseCommaSeparatedList, toTitleCase } from "@trenova/shared/lib/utils";
import { useMemo } from "react";
import { Controller, useFormContext, useWatch, type Path } from "react-hook-form";
import {
  CARRIER_INTEL_SECTION_ORDER,
  CARRIER_INTEL_SYNC_FIELDS,
  isRuleSupported,
  outagePolicyChoices,
  ruleActionChoices,
  type CarrierIntelSettingsFormValues,
} from "./carrier-intel-settings-schema";
import { SettingsSection } from "./settings-section";

type FormPath = Path<CarrierIntelSettingsFormValues>;

const actionBadgeVariant: Record<
  CarrierIntelRuleAction,
  "danger" | "warning" | "info" | "neutral"
> = {
  Block: "danger",
  Warn: "warning",
  Notify: "info",
  Off: "neutral",
};

type RuleGroup = {
  section: CarrierIntelSection;
  rules: { definition: CarrierIntelRuleDefinition; index: number }[];
};

export function groupRulesBySection(catalog: readonly CarrierIntelRuleDefinition[]): RuleGroup[] {
  const groups = new Map<CarrierIntelSection, RuleGroup>();
  catalog.forEach((definition, index) => {
    const group = groups.get(definition.category);
    if (group) {
      group.rules.push({ definition, index });
      return;
    }
    groups.set(definition.category, {
      section: definition.category,
      rules: [{ definition, index }],
    });
  });
  return [...groups.values()].sort(
    (left, right) =>
      CARRIER_INTEL_SECTION_ORDER.indexOf(left.section) -
      CARRIER_INTEL_SECTION_ORDER.indexOf(right.section),
  );
}

export function paramOptionLabel(option: string): string {
  return CSA_BASIC_LABELS[option] ?? toTitleCase(option);
}

export function CarrierIntelRulesTab({
  catalog,
  sections,
  providerName,
  providerConfigured,
  readOnly,
}: {
  catalog: readonly CarrierIntelRuleDefinition[];
  sections: readonly CarrierIntelSection[];
  providerName: string;
  providerConfigured: boolean;
  readOnly: boolean;
}) {
  const t = useT();
  const labels = useCarrierIntelLabels();
  const { control } = useFormContext<CarrierIntelSettingsFormValues>();
  const syncFieldOptions = useMemo(
    () =>
      CARRIER_INTEL_SYNC_FIELDS.map((field) => ({ value: field, label: labels.syncField[field] })),
    [labels],
  );
  const preTenderRefreshEnabled = useWatch({ control, name: "preTenderRefreshEnabled" });

  return (
    <fieldset disabled={readOnly} className="m-0 min-w-0 space-y-6 border-0 p-0">
      <SettingsSection
        title={t("Vetting rules")}
        description={t(
          "Choose what happens when a carrier or broker trips a rule. Block stops tenders and assignments, Warn asks for acknowledgement, Notify only records an event.",
        )}
      >
        <CarrierIntelRulesTable
          catalog={catalog}
          sections={sections}
          providerName={providerName}
          providerConfigured={providerConfigured}
        />
      </SettingsSection>
      <SettingsSection
        title={t("Enforcement")}
        description={t("How stale or missing intelligence is treated when a carrier is used.")}
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <SelectField
              control={control}
              name="outagePolicy"
              label={t("Provider outage policy")}
              description={t(
                "What happens to tenders and assignments when the provider cannot be reached.",
              )}
              options={outagePolicyChoices.map((choice) => ({
                value: choice.value,
                label: t(choice.label),
              }))}
              isReadOnly={readOnly}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="hardMaxAgeHours"
              label={t("Maximum intelligence age")}
              description={t("Intelligence older than this is refreshed before a carrier is used.")}
              sideText={t("hours")}
              min={1}
              max={8760}
              readOnly={readOnly}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              name="preTenderMaxAgeHours"
              label={t("Pre-tender freshness")}
              description={t("Refresh before tendering when the snapshot is older than this.")}
              sideText={t("hours")}
              min={1}
              max={720}
              readOnly={readOnly || !preTenderRefreshEnabled}
            />
          </FormControl>
          <FormControl cols="full" className="min-h-0">
            <SwitchField
              control={control}
              name="preTenderRefreshEnabled"
              label={t("Refresh before tender")}
              description={t("Fetch fresh intelligence before a load is tendered to a carrier.")}
              position="left"
              readOnly={readOnly}
            />
          </FormControl>
          <FormControl cols="full" className="min-h-0">
            <SwitchField
              control={control}
              name="confirmBlockingChanges"
              label={t("Confirm blocking changes")}
              description={t(
                "Re-check with the provider before a new blocking finding takes effect.",
              )}
              position="left"
              readOnly={readOnly}
            />
          </FormControl>
          <FormControl cols="full" className="min-h-0">
            <SwitchField
              control={control}
              name="autoDisqualifyOnBlock"
              label={t("Disqualify on block")}
              description={t("Mark the carrier as disqualified when a blocking rule fires.")}
              position="left"
              readOnly={readOnly}
            />
          </FormControl>
          <FormControl cols="full" className="min-h-0">
            <SwitchField
              control={control}
              name="autoApplySafetyRating"
              label={t("Apply safety rating")}
              description={t("Copy the FMCSA safety rating onto the carrier record automatically.")}
              position="left"
              readOnly={readOnly}
            />
          </FormControl>
          <FormControl cols="full">
            <MultiCheckboxField
              control={control}
              name="autoSyncFields"
              label={t("Fields synced automatically")}
              description={t(
                "Changes to these fields are applied to the carrier without review. Everything else is offered as a suggestion.",
              )}
              options={syncFieldOptions}
            />
          </FormControl>
        </FormGroup>
      </SettingsSection>
    </fieldset>
  );
}

export function CarrierIntelRulesTable({
  catalog,
  sections,
  providerName,
  providerConfigured,
}: {
  catalog: readonly CarrierIntelRuleDefinition[];
  sections: readonly CarrierIntelSection[];
  providerName: string;
  providerConfigured: boolean;
}) {
  const t = useT();
  const groups = useMemo(() => groupRulesBySection(catalog), [catalog]);

  if (catalog.length === 0) {
    return (
      <div className="border-border bg-muted/20 text-muted-foreground rounded-md border p-4 text-sm">
        {t("No vetting rules are available.")}
      </div>
    );
  }

  return (
    <Table containerClassName="rounded-md border">
      <TableHeader>
        <TableRow>
          <TableHead>{t("Rule")}</TableHead>
          <TableHead className="w-36">{t("Action")}</TableHead>
          <TableHead className="w-80">{t("Parameters")}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {groups.map((group) => (
          <RuleGroupRows
            key={group.section}
            group={group}
            sections={sections}
            providerName={providerName}
            providerConfigured={providerConfigured}
          />
        ))}
      </TableBody>
    </Table>
  );
}

function RuleGroupRows({
  group,
  sections,
  providerName,
  providerConfigured,
}: {
  group: RuleGroup;
  sections: readonly CarrierIntelSection[];
  providerName: string;
  providerConfigured: boolean;
}) {
  const labels = useCarrierIntelLabels();

  return (
    <>
      <TableRow className="bg-muted/40 hover:bg-muted/40">
        <TableCell
          colSpan={3}
          className="text-muted-foreground py-1.5 text-xs font-semibold uppercase"
        >
          {labels.section[group.section] ?? group.section}
        </TableCell>
      </TableRow>
      {group.rules.map(({ definition, index }) => (
        <RuleRow
          key={definition.code}
          definition={definition}
          index={index}
          supported={providerConfigured && isRuleSupported(definition, sections)}
          providerName={providerName}
          providerConfigured={providerConfigured}
        />
      ))}
    </>
  );
}

function RuleRow({
  definition,
  index,
  supported,
  providerName,
  providerConfigured,
}: {
  definition: CarrierIntelRuleDefinition;
  index: number;
  supported: boolean;
  providerName: string;
  providerConfigured: boolean;
}) {
  const t = useT();
  const { control } = useFormContext<CarrierIntelSettingsFormValues>();
  const action = useWatch({ control, name: `rules.${index}.action` });

  return (
    <TableRow data-testid={`carrier-intel-rule-${definition.code}`} data-supported={supported}>
      <TableCell className="align-top whitespace-normal">
        <div className={cn("space-y-1", !supported && "opacity-70")}>
          <div className="flex flex-wrap items-center gap-1.5">
            <span className="text-sm font-medium">{definition.label}</span>
            {definition.gateRelevant ? <Badge variant="neutral" appearance="outline">{t("Gates tenders")}</Badge> : null}
            {!supported ? (
              <Tooltip>
                <TooltipTrigger
                  render={
                    <Badge variant="warning">
                      {providerConfigured
                        ? t("Not provided by {0}", providerName)
                        : t("No provider connected")}
                    </Badge>
                  }
                />
                <TooltipContent className="max-w-xs">
                  {t(
                    "This rule needs data the provider does not return, so it evaluates as Unverifiable.",
                  )}
                </TooltipContent>
              </Tooltip>
            ) : null}
          </div>
          <p className="text-muted-foreground text-xs">{definition.description}</p>
          {action && action !== definition.recommendedAction ? (
            <p className="text-muted-foreground text-2xs">
              {t("Recommended: {0}", t(definition.recommendedAction))}
            </p>
          ) : null}
        </div>
      </TableCell>
      <TableCell className="align-top">
        <RuleActionSelect index={index} label={definition.label} />
      </TableCell>
      <TableCell className="align-top whitespace-normal">
        {definition.params.length === 0 ? (
          <span className="text-muted-foreground text-xs">{t("None")}</span>
        ) : (
          <div className="space-y-2">
            {definition.params.map((param) => (
              <RuleParamInput
                key={param.key}
                index={index}
                param={param}
                disabled={action === "Off"}
              />
            ))}
          </div>
        )}
      </TableCell>
    </TableRow>
  );
}

function RuleActionSelect({ index, label }: { index: number; label: string }) {
  const t = useT();
  const { control } = useFormContext<CarrierIntelSettingsFormValues>();
  const items = useMemo(
    () => ruleActionChoices.map((choice) => ({ value: choice.value, label: t(choice.label) })),
    [t],
  );

  return (
    <Controller
      control={control}
      name={`rules.${index}.action`}
      render={({ field, fieldState }) => (
        <div className="space-y-1">
          <Select
            items={items}
            value={field.value}
            onValueChange={(value) => {
              if (value) {
                field.onChange(value);
              }
            }}
          >
            <SelectTrigger
              className="w-28"
              aria-label={t("{0} action", label)}
              aria-invalid={fieldState.invalid || undefined}
            >
              <SelectValue>
                {(value: CarrierIntelRuleAction | null) =>
                  value ? <Badge variant={actionBadgeVariant[value]}>{t(value)}</Badge> : null
                }
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              {items.map((item) => (
                <SelectItem key={item.value} value={item.value}>
                  {item.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {fieldState.error?.message ? (
            <p className="text-2xs text-destructive">{fieldState.error.message}</p>
          ) : null}
        </div>
      )}
    />
  );
}

const decimalScaleByType: Record<string, number> = {
  Integer: 0,
  Decimal: 2,
  Number: 4,
};

function RuleParamInput({
  index,
  param,
  disabled,
}: {
  index: number;
  param: CarrierIntelRuleParamDefinition;
  disabled: boolean;
}) {
  const t = useT();
  const { control } = useFormContext<CarrierIntelSettingsFormValues>();
  const name = `rules.${index}.params.${param.key}` as const;

  if (param.type === "Select") {
    return <RuleParamSelect name={name} param={param} disabled={disabled} />;
  }

  if (param.type === "MultiSelect") {
    return <RuleParamMultiSelect name={name} param={param} disabled={disabled} />;
  }

  return (
    <NumberField
      control={control}
      name={name as FormPath}
      valueType="string"
      label={param.label}
      description={param.helpText ?? undefined}
      placeholder={param.default === "" ? t("Not set") : param.default}
      decimalScale={decimalScaleByType[param.type] ?? 4}
      min={param.min ?? undefined}
      max={param.max ?? undefined}
      readOnly={disabled}
    />
  );
}

function RuleParamSelect({
  name,
  param,
  disabled,
}: {
  name: `rules.${number}.params.${string}`;
  param: CarrierIntelRuleParamDefinition;
  disabled: boolean;
}) {
  const { control } = useFormContext<CarrierIntelSettingsFormValues>();
  const items = useMemo(
    () =>
      (param.options ?? []).map((option) => ({ value: option, label: paramOptionLabel(option) })),
    [param.options],
  );

  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => (
        <div className="space-y-1">
          <Label className="text-xs">{param.label}</Label>
          <Select
            items={items}
            value={field.value ?? ""}
            disabled={disabled}
            onValueChange={(value) => field.onChange(value ?? "")}
          >
            <SelectTrigger className="w-full" aria-label={param.label}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {items.map((item) => (
                <SelectItem key={item.value} value={item.value}>
                  {item.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {param.helpText ? (
            <p className="text-2xs text-muted-foreground">{param.helpText}</p>
          ) : null}
          {fieldState.error?.message ? (
            <p className="text-2xs text-destructive">{fieldState.error.message}</p>
          ) : null}
        </div>
      )}
    />
  );
}

function RuleParamMultiSelect({
  name,
  param,
  disabled,
}: {
  name: `rules.${number}.params.${string}`;
  param: CarrierIntelRuleParamDefinition;
  disabled: boolean;
}) {
  const { control } = useFormContext<CarrierIntelSettingsFormValues>();

  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => {
        const selected = new Set(parseCommaSeparatedList(field.value ?? ""));
        const toggle = (option: string, checked: boolean) => {
          if (checked) {
            selected.add(option);
          } else {
            selected.delete(option);
          }
          field.onChange((param.options ?? []).filter((item) => selected.has(item)).join(","));
        };

        return (
          <div className="space-y-1" role="group" aria-label={param.label}>
            <Label className="text-xs">{param.label}</Label>
            <div className="grid grid-cols-2 gap-1">
              {(param.options ?? []).map((option) => {
                const id = `${name}-${option}`;
                return (
                  <div key={option} className="flex items-center gap-1.5">
                    <Checkbox
                      id={id}
                      size="sm"
                      checked={selected.has(option)}
                      disabled={disabled}
                      onCheckedChange={(state) => toggle(option, state === true)}
                    />
                    <Label htmlFor={id} className="text-2xs cursor-pointer font-normal">
                      {paramOptionLabel(option)}
                    </Label>
                  </div>
                );
              })}
            </div>
            {param.helpText ? (
              <p className="text-2xs text-muted-foreground">{param.helpText}</p>
            ) : null}
            {fieldState.error?.message ? (
              <p className="text-2xs text-destructive">{fieldState.error.message}</p>
            ) : null}
          </div>
        );
      }}
    />
  );
}
