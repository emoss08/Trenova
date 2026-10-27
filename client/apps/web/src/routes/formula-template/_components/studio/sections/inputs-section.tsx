import { TariffInputsFields } from "@/components/tariff-inputs/tariff-inputs-fields";
import { NovaButton } from "@/components/tariff-studio/nova-button";
import { inputDocEdits } from "@/components/tariff-studio/source-edits";
import { describeApiError } from "@/lib/api-error-message";
import { apiService } from "@/services/api";
import { useMutation } from "@tanstack/react-query";
import { toast } from "@trenova/shared/components/ui/toast";
import { VariableIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { FormProvider, useWatch } from "react-hook-form";
import { useStudio } from "../studio-context";
import { SectionHead } from "./section-head";

/**
 * The tariff's contract interface, read from its `input` declarations: each input with its
 * label (the doc comment), its name and type (a link to its declaration) and its default, as the
 * rate agreement rule form will ask for it. The values set here are the preview's sample inputs; they belong to the preview, not
 * to the template, and an input left alone takes its default.
 */
export function InputsSection() {
  const t = useT();
  const { check, errors, previewInputs, reveal, draftContent, applyFix } = useStudio();
  const declarations = check.inputs;
  const undescribed = declarations.filter((declaration) => !declaration.doc).length;
  // Nova writes a doc comment for each input declared without one; they land in the editor as
  // one change, so a single undo takes them all back.
  const describe = useMutation({
    mutationFn: () => apiService.formulaTemplateService.describeInputs(draftContent()),
    onSuccess: (answer) => {
      // Placed against the file as it is now, which the check was made from.
      const changes = inputDocEdits(draftContent().source, check.inputs, answer.descriptions);
      if (changes.length === 0) {
        toast(t("Nova had nothing to add: every input is described."));
        return;
      }
      applyFix({ changes });
    },
    onError: (error) =>
      toast.error(describeApiError(error, t("Nova could not describe the inputs."))),
  });
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
        {undescribed > 0 && (
          <NovaButton
            busy={describe.isPending}
            disabled={errors > 0 || describe.isPending}
            onClick={() => describe.mutate()}
          >
            {t("Describe {0, plural, one {# input} other {# inputs}}", undescribed)}
          </NovaButton>
        )}
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

      <div className="bg-card rounded-lg border p-3">
        {declarations.length === 0 ? (
          <p className="text-foreground-muted text-sm">
            {t(
              "This tariff declares no inputs. Write input rate: Money/mi = $2.35/mi to let each contract set its own rate.",
            )}
          </p>
        ) : (
          <FormProvider {...previewInputs}>
            <TariffInputsFields
              declarations={declarations}
              name="inputs"
              onReveal={(declaration) => declaration.range && reveal(declaration.range)}
            />
          </FormProvider>
        )}
      </div>
    </section>
  );
}
