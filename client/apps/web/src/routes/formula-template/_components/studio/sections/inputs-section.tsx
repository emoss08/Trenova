import { TariffInputsFields } from "@/components/tariff-inputs/tariff-inputs-fields";
import { VariableIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { FormProvider, useWatch } from "react-hook-form";
import { useStudio } from "../studio-context";
import { SectionHead } from "./section-head";

/**
 * The tariff's contract interface, read from its `input` declarations: each input with its
 * label (the doc comment), its type and its default, as the rate agreement rule form will ask
 * for it. The values set here are the preview's sample inputs; they belong to the preview, not
 * to the template, and an input left alone takes its default.
 */
export function InputsSection() {
  const t = useT();
  const { check, previewInputs, reveal } = useStudio();
  const declarations = check.inputs;
  const sample = useWatch({ control: previewInputs.control, name: "inputs" });
  const changed = Object.values(sample ?? {}).filter((value) => value !== undefined).length;

  return (
    <section className="flex flex-col gap-3">
      <SectionHead
        section="inputs"
        icon={VariableIcon}
        title={t("Inputs")}
        description={t(
          "What a rate agreement sets for this tariff. The values here only price the preview.",
        )}
      >
        {changed > 0 && (
          <Button
            type="button"
            variant="ghost"
            size="xs"
            onClick={() => previewInputs.reset({ inputs: {} })}
          >
            {t("Reset {0, plural, one {# sample value} other {# sample values}}", changed)}
          </Button>
        )}
      </SectionHead>

      <div className="bg-card flex flex-col gap-3 rounded-lg border p-3">
        {declarations.length === 0 ? (
          <p className="text-foreground-muted text-sm">
            {t(
              "This tariff declares no inputs. Write input rate: Money/mi = $2.35/mi to let each contract set its own rate.",
            )}
          </p>
        ) : (
          <>
            <ul className="flex flex-wrap gap-1.5" aria-label={t("Declared inputs")}>
              {declarations.map((declaration) => (
                <li key={declaration.name}>
                  <button
                    type="button"
                    disabled={!declaration.range}
                    onClick={() => declaration.range && reveal(declaration.range)}
                    className="ui-focus-ring hover:bg-surface-hover rounded-badge flex items-center gap-1.5 border px-2 py-0.5 font-mono text-xs"
                  >
                    {declaration.name}
                    <span className="text-syntax-type">{declaration.type}</span>
                    {declaration.required && (
                      <span className="text-danger-foreground font-sans">{t("required")}</span>
                    )}
                  </button>
                </li>
              ))}
            </ul>
            <FormProvider {...previewInputs}>
              <TariffInputsFields declarations={declarations} name="inputs" />
            </FormProvider>
          </>
        )}
      </div>
    </section>
  );
}
