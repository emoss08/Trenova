import { SectionPanel } from "@/components/section-panel";
import {
  FACTORING_REMITTANCE_FIELD_LABELS,
  FACTORING_REMITTANCE_FIELDS,
  mappingFromColumns,
  mappingProblems,
  mappingToColumns,
  type MappingProblem,
  type RemittanceMapping,
} from "@/lib/factoring-remittance";
import {
  stageFactoringRemittance,
  type FactoringRemittance,
} from "@/lib/graphql/factoring-remittance";
import { invalidateFactoringQueries } from "@/lib/queries/factoring";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { FactoringRemittanceField } from "@trenova/graphql/generated/graphql";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { useMemo, useState } from "react";
import { toast } from "sonner";

const NOT_IN_FILE = "none";

function problemText(t: TranslateFn, problem: MappingProblem): string {
  switch (problem.kind) {
    case "reference":
      return t("Choose the column that holds the invoice number or the factor's reference.");
    case "amount":
      return t(
        "Choose the column that holds the amount paid, the advance, the reserve or the invoice amount.",
      );
    case "duplicate":
      return t(
        "{0} and {1} use the same column.",
        FACTORING_REMITTANCE_FIELD_LABELS[problem.fields[0]],
        FACTORING_REMITTANCE_FIELD_LABELS[problem.fields[1]],
      );
  }
}

/**
 * The file's headers did not say which columns hold what. A person maps them
 * once; the mapping is saved on the factor, so its next file reads on its own.
 */
export function RemittanceMappingStep({ remittance }: { remittance: FactoringRemittance }) {
  const t = useT();
  const queryClient = useQueryClient();
  const [mapping, setMapping] = useState<RemittanceMapping>(() =>
    mappingFromColumns(remittance.mapping),
  );
  const problems = useMemo(() => mappingProblems(mapping), [mapping]);

  const stage = useMutation({
    mutationFn: () => {
      if (!remittance.documentId) throw new Error("The remittance has no file");
      return stageFactoringRemittance({
        id: remittance.id,
        version: remittance.version,
        documentId: remittance.documentId,
        mapping: mappingToColumns(mapping),
      });
    },
    onSuccess: () => {
      invalidateFactoringQueries(queryClient);
      toast.success(t("Columns mapped"), {
        description: t("The layout is saved for this factor's next file."),
      });
    },
    onError: (error) => {
      const described = describeError(error);
      toast.error(t("The remittance could not be read with this mapping"), {
        description: described.detail ?? described.message,
      });
    },
  });

  const choose = (field: FactoringRemittanceField, value: string) => {
    setMapping((current) => ({
      ...current,
      [field]: value === NOT_IN_FILE ? null : Number(value),
    }));
  };

  return (
    <SectionPanel
      title={t("Map the columns")}
      help={t(
        "Say which column of {0} holds each field. Leave a field out when the file has no such column.",
        remittance.fileName ?? t("the file"),
      )}
    >
      <div className="flex flex-col gap-3 p-3">
        <div className="grid gap-3 sm:grid-cols-2">
          {FACTORING_REMITTANCE_FIELDS.map((field) => {
            const column = mapping[field];
            return (
              <label key={field} className="flex flex-col gap-1 text-sm">
                <span className="font-medium">{FACTORING_REMITTANCE_FIELD_LABELS[field]}</span>
                <Select
                  value={column === null || column === undefined ? NOT_IN_FILE : String(column)}
                  onValueChange={(value) => choose(field, String(value))}
                >
                  <SelectTrigger aria-label={FACTORING_REMITTANCE_FIELD_LABELS[field]}>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={NOT_IN_FILE}>{t("Not in the file")}</SelectItem>
                    {remittance.headers.map((header, index) => (
                      <SelectItem key={`${index}-${header}`} value={String(index)}>
                        {header || t("Column {0}", index + 1)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </label>
            );
          })}
        </div>
        {problems.length > 0 ? (
          <Alert size="sm" variant="warning">
            <AlertDescription>
              <ul className="list-disc pl-4">
                {problems.map((problem) => (
                  <li key={JSON.stringify(problem)}>{problemText(t, problem)}</li>
                ))}
              </ul>
            </AlertDescription>
          </Alert>
        ) : null}
        <div className="flex justify-end">
          <Button
            type="button"
            disabled={problems.length > 0 || !remittance.documentId}
            isLoading={stage.isPending}
            onClick={() => stage.mutate()}
          >
            {t("Read with this mapping")}
          </Button>
        </div>
      </div>
    </SectionPanel>
  );
}
