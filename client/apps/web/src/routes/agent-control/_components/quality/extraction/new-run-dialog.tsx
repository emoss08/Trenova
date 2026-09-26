import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import {
  EXTRACTION_ACCURACY_KEY,
  EXTRACTION_EVAL_RUN_LIST_KEY,
  startExtractionEvalRun,
} from "@/lib/graphql/extraction-eval";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { useT } from "@trenova/shared/i18n/use-t";
import { useId, useMemo, useState } from "react";
import { toast } from "sonner";
import { DEFAULT_CASE_LIMIT, MAX_CASE_LIMIT } from "./extraction-model";

type NewRunDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  activeCases: number;
  onStarted: (runId: string) => void;
};

/**
 * Starts an evaluation of one provider's model over the active cases. Only a
 * provider enabled for document extraction can be chosen, and the run is pinned
 * to it: an answer from any other model would be credited to the wrong one.
 */
export function NewRunDialog({ open, onOpenChange, activeCases, onStarted }: NewRunDialogProps) {
  const t = useT();
  const providerFieldId = useId();
  const limitFieldId = useId();
  const queryClient = useQueryClient();
  const providers = useQuery({ ...queries.aiProvider.list(), enabled: open });
  const [providerId, setProviderId] = useState<string>("");
  const [caseLimit, setCaseLimit] = useState(String(DEFAULT_CASE_LIMIT));

  const extractors = useMemo(
    () =>
      (providers.data ?? []).filter(
        (provider) => provider.enabled && provider.tasks.includes("DocumentExtraction"),
      ),
    [providers.data],
  );
  const items = useMemo(
    () =>
      extractors.map((provider) => ({
        value: provider.id,
        label: t("{0} · {1}", provider.name, provider.model),
      })),
    [extractors, t],
  );

  const selectedId = extractors.some((provider) => provider.id === providerId)
    ? providerId
    : (extractors[0]?.id ?? "");

  const limit = Number(caseLimit);
  const limitValid = Number.isInteger(limit) && limit >= 1 && limit <= MAX_CASE_LIMIT;

  const start = useApiMutation({
    mutationFn: () => startExtractionEvalRun({ providerId: selectedId, caseLimit: limit }),
    onSuccess: async (run) => {
      toast.success(t("Evaluation started"), {
        description: t(
          "{0, plural, one {# case is being scored} other {# cases are being scored}}",
          run.casesTotal,
        ),
      });
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: [EXTRACTION_EVAL_RUN_LIST_KEY] }),
        queryClient.invalidateQueries({ queryKey: [EXTRACTION_ACCURACY_KEY] }),
      ]);
      onOpenChange(false);
      onStarted(run.id);
    },
    resourceName: t("Evaluation run"),
  });

  const noCases = activeCases === 0;
  const noProviders = providers.isSuccess && extractors.length === 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("New evaluation run")}</DialogTitle>
          <DialogDescription>
            {t(
              "Runs one provider's model over the active cases and scores every field the way corrections are scored. The run spends from the organization's evaluation budget and stops when it is reached.",
            )}
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4">
          {noCases ? (
            <Alert variant="warning" size="sm">
              <AlertDescription>
                {t("There are no active cases yet. Add corrections to the evaluation set first.")}
              </AlertDescription>
            </Alert>
          ) : null}
          {noProviders ? (
            <Alert variant="warning" size="sm">
              <AlertDescription>
                {t("No enabled provider is assigned to document extraction.")}
              </AlertDescription>
            </Alert>
          ) : null}
          <div className="grid gap-1.5">
            <Label htmlFor={providerFieldId}>{t("Provider")}</Label>
            <Select
              items={items}
              value={selectedId || null}
              onValueChange={(value) => setProviderId(value ?? "")}
              disabled={items.length === 0}
            >
              <SelectTrigger id={providerFieldId} aria-label={t("Provider")}>
                <SelectValue placeholder={t("Choose a provider")} />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {items.map((item) => (
                    <SelectItem key={item.value} value={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor={limitFieldId}>{t("Cases to run")}</Label>
            <Input
              id={limitFieldId}
              type="number"
              inputMode="numeric"
              min={1}
              max={MAX_CASE_LIMIT}
              value={caseLimit}
              aria-invalid={!limitValid}
              onChange={(event) => setCaseLimit(event.target.value)}
            />
            <p className="text-muted-foreground text-xs">
              {t(
                "The newest active cases first, up to {0}. {1, plural, one {# case is active.} other {# cases are active.}}",
                MAX_CASE_LIMIT,
                activeCases,
              )}
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            disabled={noCases || selectedId === "" || !limitValid}
            isLoading={start.isPending}
            onClick={() => start.mutate(undefined)}
          >
            {t("Start run")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
