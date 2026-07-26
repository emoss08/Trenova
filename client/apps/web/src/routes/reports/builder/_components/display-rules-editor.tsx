import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { cn } from "@trenova/shared/lib/utils";
import {
  MAX_DISPLAY_RULES,
  REPORT_RULE_OP_CHOICES,
  REPORT_TONE_CHOICES,
  ruleNeedsUpper,
  type ReportDisplayRuleOp,
  type ReportDisplayRuleSpec,
  type ReportDisplayTone,
} from "@/types/report";
import { PlusIcon, XIcon } from "lucide-react";

type DisplayRulesEditorProps = {
  rules: ReportDisplayRuleSpec[];
  onChange: (rules: ReportDisplayRuleSpec[] | undefined) => void;
};

/** Same emphasis vocabulary the grid and the exports use. */
const TONE_SWATCH: Record<ReportDisplayTone, string> = {
  positive: "bg-emerald-500",
  warning: "bg-amber-500",
  negative: "bg-destructive",
  neutral: "bg-foreground",
};

export function DisplayRulesEditor({ rules, onChange }: DisplayRulesEditorProps) {
  const update = (index: number, patch: Partial<ReportDisplayRuleSpec>) =>
    onChange(rules.map((rule, i) => (i === index ? { ...rule, ...patch } : rule)));

  const remove = (index: number) => {
    const next = rules.filter((_, i) => i !== index);
    onChange(next.length > 0 ? next : undefined);
  };

  return (
    <div className="flex flex-col gap-2 border-t border-border/60 pt-2">
      <div className="flex items-center gap-2">
        <span className="text-2xs font-medium tracking-wide text-muted-foreground uppercase">
          Highlight when
        </span>
        <div className="flex-1" />
        <Button
          variant="ghost"
          size="sm"
          className="h-6"
          disabled={rules.length >= MAX_DISPLAY_RULES}
          onClick={() => onChange([...rules, { op: "gt", value: 0, tone: "negative" }])}
        >
          <PlusIcon className="size-3.5" />
          Rule
        </Button>
      </div>

      {rules.length === 0 ? (
        <p className="text-2xs text-muted-foreground">
          Colour a value when it crosses a threshold, so an exception is visible without reading
          every number. The first matching rule wins.
        </p>
      ) : (
        rules.map((rule, index) => (
          <div key={index} className="flex flex-wrap items-center gap-1.5">
            <Select
              value={rule.op}
              onValueChange={(op) => {
                if (op) update(index, { op: op as ReportDisplayRuleOp });
              }}
              items={REPORT_RULE_OP_CHOICES}
            >
              <SelectTrigger className="h-7 w-32">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {REPORT_RULE_OP_CHOICES.map((choice) => (
                  <SelectItem key={choice.value} value={choice.value}>
                    {choice.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Input
              className="h-7 w-20"
              type="number"
              step="any"
              value={String(rule.value)}
              onChange={(event) => update(index, { value: Number(event.target.value) })}
            />

            {ruleNeedsUpper(rule.op) && (
              <>
                <span className="text-2xs text-muted-foreground">and</span>
                <Input
                  className="h-7 w-20"
                  type="number"
                  step="any"
                  value={String(rule.upper ?? 0)}
                  onChange={(event) => update(index, { upper: Number(event.target.value) })}
                />
              </>
            )}

            <Select
              value={rule.tone}
              onValueChange={(tone) => {
                if (tone) update(index, { tone: tone as ReportDisplayTone });
              }}
              items={REPORT_TONE_CHOICES}
            >
              <SelectTrigger className="h-7 w-28">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {REPORT_TONE_CHOICES.map((choice) => (
                  <SelectItem key={choice.value} value={choice.value}>
                    <span className="flex items-center gap-2">
                      <span className={cn("size-2 rounded-full", TONE_SWATCH[choice.value])} />
                      {choice.label}
                    </span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Button
              variant="ghost"
              size="icon"
              className="size-6"
              onClick={() => remove(index)}
              aria-label="Remove rule"
            >
              <XIcon className="size-3.5" />
            </Button>
          </div>
        ))
      )}

      <p className="text-2xs text-muted-foreground">
        Percent columns compare against the displayed value, so 12 means 12%.
      </p>
    </div>
  );
}
