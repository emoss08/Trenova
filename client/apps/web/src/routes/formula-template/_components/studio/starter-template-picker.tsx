import { ControlledFormulaTemplateAutocompleteField } from "@/components/autocomplete-fields";
import { queries } from "@/lib/queries";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import type {
  FormulaTemplateFormValues,
  StandardTemplate,
} from "@trenova/shared/types/formula-template";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { CopyIcon, LayoutTemplateIcon } from "lucide-react";
import { useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { applyTemplateValues, copyValuesFrom, starterValuesFrom } from "./starter-values";

const SKELETON_KEYS = ["a", "b", "c", "d"] as const;

export function StarterTemplatePicker() {
  const { control, setValue, getValues } = useFormContext<FormulaTemplateFormValues>();
  const queryClient = useQueryClient();
  const [copyFromId, setCopyFromId] = useState("");
  const [copying, setCopying] = useState(false);

  const templateType = useWatch({ control, name: "type" });
  const standards = useQuery({ ...queries.formulaTemplate.standards, staleTime: Infinity });
  const matchingStandards = (standards.data ?? []).filter(
    (standard) => !templateType || standard.type === templateType,
  );

  const applyStandard = (standard: StandardTemplate) => {
    applyTemplateValues(setValue, starterValuesFrom(standard));
    if (!getValues("description")?.trim()) {
      setValue("description", standard.description, { shouldDirty: true });
    }
  };

  const copyFrom = async (templateId: string) => {
    setCopyFromId(templateId);
    if (!templateId) return;
    setCopying(true);
    try {
      const source = await queryClient.fetchQuery(queries.formulaTemplate.get(templateId));
      applyTemplateValues(setValue, copyValuesFrom(source));
      if (!getValues("description")?.trim()) {
        setValue("description", source.description ?? "", { shouldDirty: true });
      }
      toast.success(`Copied from ${source.name}`, {
        description: "The formula, variables, lines, and charge policy are in the editor.",
      });
    } catch {
      toast.error("Could not load that template");
      setCopyFromId("");
    } finally {
      setCopying(false);
    }
  };

  return (
    <div className="bg-muted/30 space-y-3 rounded-lg border p-3">
      <div className="space-y-2">
        <div className="text-muted-foreground flex items-center gap-2 text-xs font-medium tracking-wide uppercase">
          <LayoutTemplateIcon className="size-3" />
          Start from a standard
        </div>
        {standards.isPending ? (
          <div className="grid grid-cols-2 gap-2">
            {SKELETON_KEYS.map((key) => (
              <Skeleton key={key} className="h-14 w-full" />
            ))}
          </div>
        ) : standards.isError ? (
          <p className="text-muted-foreground text-xs">
            The standard library could not be loaded. Write the formula by hand or copy an existing
            template below.
          </p>
        ) : matchingStandards.length === 0 ? (
          <p className="text-muted-foreground text-xs">
            No standards exist for this template type yet.
          </p>
        ) : (
          <div className="grid grid-cols-2 gap-2">
            {matchingStandards.map((standard) => (
              <Button
                key={standard.name}
                type="button"
                variant="outline"
                onClick={() => applyStandard(standard)}
                className="h-auto flex-col items-start gap-0.5 px-3 py-2 text-left whitespace-normal"
              >
                <span className="text-xs font-semibold">{standard.name}</span>
                <span className="text-muted-foreground text-2xs font-normal">
                  {standard.description}
                </span>
              </Button>
            ))}
          </div>
        )}
      </div>

      <div className="space-y-2 border-t pt-3">
        <div className="text-muted-foreground flex items-center gap-2 text-xs font-medium tracking-wide uppercase">
          <CopyIcon className="size-3" />
          Or copy an existing template
        </div>
        <ControlledFormulaTemplateAutocompleteField
          label=""
          placeholder="Search your templates..."
          value={copyFromId}
          onValueChange={(value) => void copyFrom(value)}
          disabled={copying}
        />
        <p className="text-2xs text-muted-foreground">
          Copies the formula and charge policy into this new template. The original is untouched and
          keeps its own history.
        </p>
      </div>
    </div>
  );
}
