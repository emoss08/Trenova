import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormSaveDock } from "@/components/form-save-dock";
import { Badge, type BadgeVariant } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Form, FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { Separator } from "@/components/ui/separator";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import {
  sequenceTypes,
  sequenceConfigDocumentSchema,
  type SequenceConfig,
  type SequenceConfigDocument,
  type SequenceType,
} from "@/types/sequence-config";
import { zodResolver } from "@hookform/resolvers/zod";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import { FormProvider, useForm, useFormContext, useWatch } from "react-hook-form";

const sequenceTitles: Record<SequenceType, string> = {
  pro_number: "Pro Number",
  consolidation: "Consolidation Number",
  invoice: "Invoice Number",
  work_order: "Work Order Number",
  journal_batch: "Journal Batch Number",
  journal_entry: "Journal Entry Number",
  manual_journal_request: "Manual Journal Request Number",
};

const sequenceDescriptions: Record<SequenceType, string> = {
  pro_number: "Controls generated PRO numbers for shipment creation and tracking.",
  consolidation: "Controls consolidation number generation for grouped shipment operations.",
  invoice: "Controls invoice identifier generation for billing documents.",
  work_order: "Controls work order identifier generation for operational workflows.",
  journal_batch: "Controls journal batch numbering for accounting posting groups.",
  journal_entry: "Controls journal entry numbering for posted ledger entries.",
  manual_journal_request:
    "Controls manual journal request numbering before approval and posting.",
};

const sequenceOrder: SequenceType[] = [...sequenceTypes];

type SequenceColorConfig = {
  badge: BadgeVariant;
  accentBorder: string;
  previewBg: string;
  previewText: string;
  previewBorder: string;
};

const sequenceColors: Record<SequenceType, SequenceColorConfig> = {
  pro_number: {
    badge: "info",
    accentBorder: "border-l-blue-500",
    previewBg: "bg-blue-600/5",
    previewText: "text-blue-600 dark:text-blue-400",
    previewBorder: "border-blue-600/20",
  },
  consolidation: {
    badge: "purple",
    accentBorder: "border-l-purple-500",
    previewBg: "bg-purple-600/5",
    previewText: "text-purple-600 dark:text-purple-400",
    previewBorder: "border-purple-600/20",
  },
  invoice: {
    badge: "teal",
    accentBorder: "border-l-teal-500",
    previewBg: "bg-teal-600/5",
    previewText: "text-teal-600 dark:text-teal-400",
    previewBorder: "border-teal-600/20",
  },
  work_order: {
    badge: "orange",
    accentBorder: "border-l-orange-500",
    previewBg: "bg-orange-600/5",
    previewText: "text-orange-600 dark:text-orange-400",
    previewBorder: "border-orange-600/20",
  },
  journal_batch: {
    badge: "secondary",
    accentBorder: "border-l-slate-500",
    previewBg: "bg-slate-600/5",
    previewText: "text-slate-600 dark:text-slate-300",
    previewBorder: "border-slate-600/20",
  },
  journal_entry: {
    badge: "indigo",
    accentBorder: "border-l-indigo-500",
    previewBg: "bg-indigo-600/5",
    previewText: "text-indigo-600 dark:text-indigo-400",
    previewBorder: "border-indigo-600/20",
  },
  manual_journal_request: {
    badge: "warning",
    accentBorder: "border-l-yellow-500",
    previewBg: "bg-yellow-600/5",
    previewText: "text-yellow-700 dark:text-yellow-400",
    previewBorder: "border-yellow-600/20",
  },
};

const separatorOptions = [
  { label: "Hyphen (-)", value: "-" },
  { label: "Underscore (_)", value: "_" },
  { label: "Slash (/)", value: "/" },
  { label: "Period (.)", value: "." },
];

const tokenBadges = [
  { token: "{P}", label: "Prefix" },
  { token: "{Y}", label: "Year" },
  { token: "{M}", label: "Month" },
  { token: "{W}", label: "Week" },
  { token: "{D}", label: "Day" },
  { token: "{L}", label: "Location" },
  { token: "{B}", label: "Business Unit" },
  { token: "{S}", label: "Sequence" },
  { token: "{R}", label: "Random" },
  { token: "{C}", label: "Check Digit" },
];

export default function SequenceConfigForm() {
  const { data } = useSuspenseQuery({
    ...queries.sequenceConfig.get(),
  });

  const form = useForm({
    resolver: zodResolver(sequenceConfigDocumentSchema),
    defaultValues: data,
  });

  const { handleSubmit, reset, setError, watch } = form;

  const { mutateAsync } = useOptimisticMutation({
    queryKey: queries.sequenceConfig.get._def,
    mutationFn: async (values: SequenceConfigDocument) =>
      apiService.sequenceConfigService.update(values),
    resourceName: "Sequence Configuration",
    resetForm: reset,
    setFormError: setError,
    invalidateQueries: [queries.sequenceConfig.get._def],
  });

  const onSubmit = useCallback(
    async (values: SequenceConfigDocument) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  const configs = watch("configs") || [];

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="flex flex-col gap-5 pb-14">
          <Tabs defaultValue={sequenceOrder[0]} className="gap-4">
            <TabsList className="grid w-full grid-cols-2 md:grid-cols-4 xl:grid-cols-7">
              {sequenceOrder.map((type) => {
                const colors = sequenceColors[type];
                return (
                  <TabsTrigger key={type} value={type} className="gap-1.5">
                    <span>{sequenceTitles[type]}</span>
                    <Badge variant={colors.badge} className="px-1.5 py-0 text-[10px]">
                      {type.replace("_", " ")}
                    </Badge>
                  </TabsTrigger>
                );
              })}
            </TabsList>

            {sequenceOrder.map((type) => {
              const index = configs.findIndex((cfg) => cfg?.sequenceType === type);
              if (index < 0) {
                return null;
              }

              return (
                <TabsContent key={type} value={type}>
                  <SequenceTypePanel index={index} sequenceType={type} />
                </TabsContent>
              );
            })}
          </Tabs>

          <FormSaveDock saveButtonContent="Save Changes" />
        </div>
      </Form>
    </FormProvider>
  );
}

function SequenceTypePanel({ index, sequenceType }: { index: number; sequenceType: SequenceType }) {
  const { control } = useFormContext<SequenceConfigDocument>();
  const colors = sequenceColors[sequenceType];

  const config = useWatch({
    control,
    name: `configs.${index}`,
  }) as SequenceConfig | undefined;

  const includeYear = useWatch({
    control,
    name: `configs.${index}.includeYear`,
  });
  const includeRandomDigits = useWatch({
    control,
    name: `configs.${index}.includeRandomDigits`,
  });
  const useSeparators = useWatch({
    control,
    name: `configs.${index}.useSeparators`,
  });
  const allowCustomFormat = useWatch({
    control,
    name: `configs.${index}.allowCustomFormat`,
  });

  const preview = useMemo(() => buildSequencePreview(config), [config]);

  return (
    <Card className="overflow-hidden">
      <CardHeader>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <CardTitle>{sequenceTitles[sequenceType]}</CardTitle>
            <CardDescription>{sequenceDescriptions[sequenceType]}</CardDescription>
          </div>
          <Badge variant={colors.badge} className="uppercase">
            {sequenceType.replace("_", " ")}
          </Badge>
        </div>
      </CardHeader>

      <CardContent className="space-y-5">
        <div
          className={cn(
            "rounded-lg border border-l-4 p-4",
            colors.previewBg,
            colors.previewBorder,
            colors.accentBorder,
          )}
        >
          <div className="mb-2 flex items-center gap-2">
            <span className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
              Live Preview
            </span>
          </div>
          <code
            key={preview}
            className={cn("block font-mono text-xl font-bold", colors.previewText)}
          >
            {preview || "\u2014"}
          </code>
          <p className="mt-2 text-xs text-muted-foreground">
            Representative sample \u2014 actual values increment sequentially.
          </p>
        </div>

        <Separator />

        <FormSection
          title="Core Structure"
          description="Primary sequence components and delimiter behavior."
        >
          <FormGroup cols={2}>
            <FormControl>
              <InputField
                control={control}
                name={`configs.${index}.prefix`}
                label="Prefix"
                placeholder="Enter prefix"
                maxLength={20}
              />
            </FormControl>
            <FormControl>
              <NumberField
                control={control}
                name={`configs.${index}.sequenceDigits`}
                label="Sequence Digits"
                min={1}
                max={10}
              />
            </FormControl>
            <FormControl>
              <SwitchField
                control={control}
                name={`configs.${index}.useSeparators`}
                label="Use Separators"
                description="Insert a delimiter character between each token segment"
                position="left"
                outlined
              />
            </FormControl>
            {useSeparators && (
              <FormControl>
                <SelectField
                  control={control}
                  name={`configs.${index}.separatorChar`}
                  label="Separator Character"
                  options={separatorOptions}
                  placeholder="Select separator"
                />
              </FormControl>
            )}
          </FormGroup>
        </FormSection>

        <Separator />

        <FormSection
          title="Date Components"
          description="Include date tokens to encode period context."
        >
          <FormGroup cols={2}>
            <FormControl>
              <SwitchField
                control={control}
                name={`configs.${index}.includeYear`}
                label="Include Year"
                description="Embed 2 or 4-digit year in the generated value"
                position="left"
                outlined
              />
            </FormControl>
            {includeYear && (
              <FormControl>
                <NumberField
                  control={control}
                  name={`configs.${index}.yearDigits`}
                  label="Year Digits"
                  min={2}
                  max={4}
                />
              </FormControl>
            )}
            <FormControl>
              <SwitchField
                control={control}
                name={`configs.${index}.includeMonth`}
                label="Include Month"
                description="Append the current month as a 2-digit number"
                position="left"
                outlined
              />
            </FormControl>
            <FormControl>
              <SwitchField
                control={control}
                name={`configs.${index}.includeWeekNumber`}
                label="Include ISO Week Number"
                description="Append the ISO week number for weekly grouping"
                position="left"
                outlined
              />
            </FormControl>
            <FormControl>
              <SwitchField
                control={control}
                name={`configs.${index}.includeDay`}
                label="Include Day"
                description="Append the day of the month as a 2-digit number"
                position="left"
                outlined
              />
            </FormControl>
          </FormGroup>
        </FormSection>

        <Separator />

        <FormSection
          title="Context Components"
          description="Embed operational identity fields in the generated value."
        >
          <FormGroup cols={2}>
            <FormControl>
              <SwitchField
                control={control}
                name={`configs.${index}.includeLocationCode`}
                label="Include Location Code"
                description="Embed the origin location's code — resolved automatically from the shipment"
                position="left"
                outlined
              />
            </FormControl>
            <FormControl>
              <SwitchField
                control={control}
                name={`configs.${index}.includeBusinessUnitCode`}
                label="Include Business Unit Code"
                description="Embed the business unit's code — resolved automatically from the organization"
                position="left"
                outlined
              />
            </FormControl>
          </FormGroup>
        </FormSection>

        <Separator />

        <FormSection
          title="Quality And Overrides"
          description="Validation helpers and advanced custom formatting."
        >
          <FormGroup cols={2}>
            <FormControl>
              <SwitchField
                control={control}
                name={`configs.${index}.includeRandomDigits`}
                label="Include Random Digits"
                description="Append random digits for collision avoidance"
                position="left"
                outlined
              />
            </FormControl>
            {includeRandomDigits && (
              <FormControl>
                <NumberField
                  control={control}
                  name={`configs.${index}.randomDigitsCount`}
                  label="Random Digits Count"
                  min={0}
                  max={10}
                />
              </FormControl>
            )}
            <FormControl>
              <SwitchField
                control={control}
                name={`configs.${index}.includeCheckDigit`}
                label="Include Check Digit"
                description="Append a computed check digit for validation"
                position="left"
                outlined
              />
            </FormControl>
            <FormControl>
              <SwitchField
                control={control}
                name={`configs.${index}.allowCustomFormat`}
                label="Allow Custom Format"
                description="Override auto-composition with a custom token template"
                position="left"
                outlined
              />
            </FormControl>
            {allowCustomFormat && (
              <FormControl cols="full">
                <InputField
                  control={control}
                  name={`configs.${index}.customFormat`}
                  label="Custom Format"
                  placeholder="{P}{Y}{M}{S}"
                  description="Supported placeholders: {P}, {Y}, {M}, {W}, {D}, {L}, {B}, {S}, {R}, {C}"
                />
              </FormControl>
            )}
          </FormGroup>
        </FormSection>
      </CardContent>

      <CardFooter className="border-t bg-muted/20 px-6 py-3">
        <div className="flex flex-wrap gap-1.5">
          {tokenBadges.map(({ token, label }) => (
            <Badge key={token} variant="outline" className="text-[10px]">
              {token} {label}
            </Badge>
          ))}
        </div>
      </CardFooter>
    </Card>
  );
}

function buildSequencePreview(cfg?: SequenceConfig): string {
  if (!cfg) {
    return "";
  }

  if (cfg.allowCustomFormat && cfg.customFormat?.trim()) {
    return applyTemplate(cfg.customFormat, cfg);
  }

  const separator = cfg.useSeparators ? cfg.separatorChar || "-" : "";
  const parts: string[] = [];

  if (cfg.prefix) parts.push(cfg.prefix);
  if (cfg.includeYear) parts.push(cfg.yearDigits === 4 ? "2026" : "26");
  if (cfg.includeMonth) parts.push("02");
  if (cfg.includeWeekNumber) parts.push("09");
  if (cfg.includeDay) parts.push("28");
  if (cfg.includeLocationCode) parts.push("LOC");
  if (cfg.includeBusinessUnitCode) parts.push("BU");
  parts.push("9".repeat(Math.max(1, cfg.sequenceDigits || 1)));
  if (cfg.includeRandomDigits) {
    parts.push("7".repeat(Math.max(1, cfg.randomDigitsCount || 1)));
  }
  if (cfg.includeCheckDigit) parts.push("3");

  return parts.join(separator);
}

function applyTemplate(template: string, cfg: SequenceConfig): string {
  const tokenMap: Record<string, string> = {
    P: cfg.prefix || "",
    Y: cfg.yearDigits === 4 ? "2026" : "26",
    M: "02",
    W: "09",
    D: "28",
    L: "LOC",
    B: "BU",
    S: "9".repeat(Math.max(1, cfg.sequenceDigits || 1)),
    R: "7".repeat(Math.max(1, cfg.randomDigitsCount || 1)),
    C: "3",
  };

  return template.replace(/\{([PYMWDLBSRC])\}/g, (_, token: string) => {
    return tokenMap[token] ?? "";
  });
}
