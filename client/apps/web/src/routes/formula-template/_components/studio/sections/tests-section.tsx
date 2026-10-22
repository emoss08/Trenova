import { newTestBlock } from "@/components/tariff-studio/source-edits";
import { lineOf } from "@/components/tariff-studio/tests-panel";
import { Beaker02Icon, PlusIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { FormulaTemplateFormValues } from "@trenova/shared/types/formula-template";
import { useFormContext, useWatch } from "react-hook-form";
import { NovaButton } from "@/components/tariff-studio/nova-button";
import { useStudio } from "../studio-context";
import { SectionHead } from "./section-head";

const DOT = { passed: "bg-success", failed: "bg-danger", error: "bg-warning" } as const;

/**
 * The tariff's `test` blocks, as the outline lists them, with each one's verdict from the run
 * the browser makes on every change. The full results, with the failing expectations and their
 * values, are in the Tests panel; a row here takes the person to the test in the source.
 */
export function TestsSection() {
  const t = useT();
  const { check, tests, testStatus, reveal, appendToSource, setTab, askNova } = useStudio();
  const { control } = useFormContext<FormulaTemplateFormValues>();
  const source = useWatch({ control, name: "source" }) ?? "";
  const items = (check.outline?.items ?? []).filter((item) => item.kind === "test");

  const addTest = () =>
    appendToSource(
      newTestBlock(
        "tariff",
        items.map((item) => item.name),
        t("a new test"),
      ),
    );

  return (
    <section className="flex flex-col gap-3">
      <SectionHead
        section="tests"
        icon={Beaker02Icon}
        title={t("Tests")}
        description={t(
          "Tests live in the source and run on every change. A version with a failing test cannot be submitted.",
        )}
      >
        {tests.status === "ran" && (
          <Badge variant={tests.failed > 0 ? "danger" : "success"}>
            {t("{0} of {1} passing", tests.passed, tests.passed + tests.failed)}
          </Badge>
        )}
        <NovaButton onClick={() => askNova(t("Suggest tests for the cases this tariff could get wrong"))}>
          {t("Suggest tests")}
        </NovaButton>
        <Button type="button" variant="outline" size="xs" className="gap-1.5" onClick={addTest}>
          <PlusIcon className="size-3" />
          {t("Add a test")}
        </Button>
      </SectionHead>

      <div className="bg-card rounded-lg border">
        {items.length === 0 ? (
          <p className="text-foreground-muted px-3 py-3 text-sm">
            {t(
              "No tests yet. Add one, or rate a real shipment in the preview and save it as a test.",
            )}
          </p>
        ) : (
          <ul className="divide-border divide-y">
            {items.map((item) => {
              const status = testStatus.get(item.name);
              return (
                <li key={item.range.from}>
                  <button
                    type="button"
                    onClick={() => reveal(item.selection)}
                    className="ui-focus-ring hover:bg-surface-hover flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm"
                  >
                    <span
                      aria-hidden
                      className={cn(
                        "size-1.5 shrink-0 rounded-full",
                        status ? DOT[status] : "bg-foreground-subtle",
                      )}
                    />
                    <span className="min-w-0 flex-1 truncate">{item.name}</span>
                    <span className="text-foreground-subtle font-mono text-2xs tabular-nums">
                      {t("Ln {0}", lineOf(source, item.range.from))}
                    </span>
                  </button>
                </li>
              );
            })}
          </ul>
        )}
        {tests.status === "ran" && tests.failed > 0 && (
          <div className="border-t px-3 py-1.5">
            <Button type="button" variant="link" size="xs" onClick={() => setTab("tests")}>
              {t("See what failed")}
            </Button>
          </div>
        )}
      </div>
    </section>
  );
}
