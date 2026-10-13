import { formulaTemplate } from "@/lib/queries/formula-template";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { NovaMark } from "@trenova/shared/components/ui/nova-mark";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import type {
  FormulaTemplate,
  FormulaTemplateFormValues,
  StandardTemplate,
} from "@trenova/shared/types/formula-template";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { useFormContext } from "react-hook-form";
import { toast } from "sonner";
import { applyTemplateValues, copyValuesFrom, starterValuesFrom } from "../starter-values";
import { useStudio } from "../studio-context";

const SKELETON_KEYS = ["a", "b", "c", "d"] as const;
const RECENT_TEMPLATES = 6;

type Starter = {
  key: string;
  name: string;
  expression: string;
  meta: string;
  apply: () => void;
};

/**
 * Where a new template starts: describe the method and Nova drafts it, or
 * start from the standard library or one of the organization's templates.
 * Starting from one fills the editor and can be undone.
 */
export function StartCard() {
  const t = useT();
  const { askNova, novaBusy } = useStudio();
  const form = useFormContext<FormulaTemplateFormValues>();
  const [question, setQuestion] = useState("");
  const [library, setLibrary] = useState<"standard" | "org">("standard");

  const standards = useQuery({ ...formulaTemplate.standards, staleTime: Infinity });
  const recent = useQuery({
    ...formulaTemplate.recent(RECENT_TEMPLATES),
    enabled: library === "org",
    staleTime: 60_000,
  });

  const draft = () => {
    const text = question.trim();
    if (!text || novaBusy) return;
    askNova(t("Write a formula template: {0}", text));
    setQuestion("");
  };

  const startFrom = (name: string, fill: () => void) => {
    const before = form.getValues();
    fill();
    toast(t("Started from “{0}”", name), {
      action: {
        label: t("Undo"),
        onClick: () => {
          for (const key of [
            "name",
            "description",
            "type",
            "schemaId",
            "expression",
            "variableDefinitions",
            "breakdownDefinitions",
            "minCharge",
            "maxCharge",
            "roundingMode",
            "roundingPrecision",
          ] as const) {
            form.setValue(key, before[key] as never, { shouldDirty: true });
          }
        },
      },
    });
  };

  const fillIfEmpty = (key: "name" | "description", value: string) => {
    if (!form.getValues(key)?.trim()) {
      form.setValue(key, value, { shouldDirty: true });
    }
  };

  const fromStandard = (standard: StandardTemplate) =>
    startFrom(standard.name, () => {
      applyTemplateValues(form.setValue, starterValuesFrom(standard));
      form.setValue("type", standard.type, { shouldDirty: true });
      fillIfEmpty("name", standard.name);
      fillIfEmpty("description", standard.description);
    });

  const fromTemplate = (template: FormulaTemplate) =>
    startFrom(template.name, () => {
      applyTemplateValues(form.setValue, copyValuesFrom(template));
      form.setValue("type", template.type, { shouldDirty: true });
      fillIfEmpty("name", t("Copy of {0}", template.name));
      fillIfEmpty("description", template.description ?? "");
    });

  const starters: Starter[] =
    library === "standard"
      ? (standards.data ?? []).map((standard) => ({
          key: standard.name,
          name: standard.name,
          expression: standard.expression,
          meta: [
            standard.type === "AccessorialCharge" ? t("Accessorial") : t("Freight"),
            standard.variableDefinitions.length > 0
              ? t(
                  "{0, plural, one {# input} other {# inputs}}",
                  standard.variableDefinitions.length,
                )
              : null,
          ]
            .filter(Boolean)
            .join(" · "),
          apply: () => fromStandard(standard),
        }))
      : (recent.data?.results ?? []).map((template) => ({
          key: template.id ?? template.name,
          name: template.name,
          expression: template.expression,
          meta: [
            template.status === "InReview" ? t("In review") : t(template.status),
            template.currentVersionNumber ? t("v{0}", template.currentVersionNumber) : null,
            template.usageCount ? t("{0} in use", template.usageCount) : null,
          ]
            .filter(Boolean)
            .join(" · "),
          apply: () => fromTemplate(template),
        }));

  const loading = library === "standard" ? standards.isPending : recent.isPending;
  const failed = library === "standard" ? standards.isError : recent.isError;

  return (
    <div className="bg-card flex flex-col gap-3 rounded-lg border p-3">
      <Input
        value={question}
        onChange={(event) => setQuestion(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Enter") {
            event.preventDefault();
            draft();
          }
        }}
        aria-label={t("Describe the method for Nova to draft")}
        placeholder={t(
          "Describe the method: “per mile, 18% fuel, $250 minimum, $50 a stop after two”",
        )}
        disabled={novaBusy}
        className="h-9 pr-22 pl-8"
        leftElement={<NovaMark spin={novaBusy ? "always" : "none"} className="size-4" />}
        rightElement={
          <Button type="button" size="sm" disabled={!question.trim() || novaBusy} onClick={draft}>
            {t("Draft it")}
          </Button>
        }
      />

      <div className="flex items-center gap-2">
        <span className="text-foreground-subtle text-xs">{t("or start from")}</span>
        <span className="bg-border h-px flex-1" aria-hidden />
        <SegmentedControl<"standard" | "org">
          aria-label={t("Start from")}
          value={library}
          onValueChange={setLibrary}
          items={[
            { value: "standard", label: t("Standard library") },
            { value: "org", label: t("Your templates") },
          ]}
        />
      </div>

      {loading ? (
        <div className="grid grid-cols-2 gap-2">
          {SKELETON_KEYS.map((key) => (
            <Skeleton key={key} className="h-18 w-full" />
          ))}
        </div>
      ) : failed ? (
        <p className="text-foreground-muted text-xs">
          {t("These could not be loaded. Write the formula by hand or ask Nova to draft it.")}
        </p>
      ) : starters.length === 0 ? (
        <p className="text-foreground-muted text-xs">
          {library === "standard"
            ? t("The standard library is empty.")
            : t("Your organization has no templates yet.")}
        </p>
      ) : (
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          {starters.map((starter) => (
            <Button
              key={starter.key}
              type="button"
              variant="outline"
              size="bare"
              onClick={starter.apply}
              className="flex h-auto min-w-0 flex-col items-stretch gap-1 rounded-md px-3 py-2 text-left whitespace-normal"
            >
              <span className="truncate text-sm font-medium">{starter.name}</span>
              <code className="text-foreground-muted line-clamp-2 font-mono text-xs wrap-break-word">
                {starter.expression}
              </code>
              {starter.meta && (
                <span className="text-foreground-subtle text-2xs font-normal">{starter.meta}</span>
              )}
            </Button>
          ))}
        </div>
      )}
    </div>
  );
}
