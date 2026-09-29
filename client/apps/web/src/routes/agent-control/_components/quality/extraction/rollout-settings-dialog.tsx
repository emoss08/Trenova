import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  EXTRACTION_ROLLOUT_KEY,
  EXTRACTION_ROLLOUT_REPORT_KEY,
  updateExtractionRollout,
  type ExtractionRollout,
} from "@/lib/graphql/extraction-rollout";
import { queries } from "@/lib/queries";
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
import { Switch } from "@trenova/shared/components/ui/switch";
import { useT } from "@trenova/shared/i18n/use-t";
import { useId, useMemo, useState, type ReactNode } from "react";
import { toast } from "sonner";
import {
  MAX_ACCURACY_DROP_POINTS,
  MAX_REJECTION_INCREASE_POINTS,
  MAX_ROLLOUT_PERCENT,
  MIN_ACCURACY_DROP_POINTS,
  MIN_REJECTION_INCREASE_POINTS,
  MIN_ROLLOUT_PERCENT,
  rolloutDraftOf,
  rolloutInput,
  rolloutProblems,
  type RolloutDraft,
} from "./rollout-model";

type RolloutSettingsDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  rollout: ExtractionRollout;
};

type NumberFieldProps = {
  label: string;
  min: number;
  max: number;
  value: string;
  problem?: string;
  hint?: ReactNode;
  onChange: (value: string) => void;
};

function NumberField({ label, min, max, value, problem, hint, onChange }: NumberFieldProps) {
  const t = useT();
  const id = useId();

  return (
    <div className="grid gap-1.5">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        type="number"
        inputMode="numeric"
        min={min}
        max={max}
        value={value}
        aria-invalid={problem !== undefined}
        onChange={(event) => onChange(event.target.value)}
      />
      {problem ? (
        <p className="text-danger text-xs">{t(problem)}</p>
      ) : hint ? (
        <p className="text-muted-foreground text-xs">{hint}</p>
      ) : null}
    </div>
  );
}

/**
 * Chooses the candidate that serves a share of real extractions, how large a
 * share, and how far it may fall behind production before a guard stops it.
 * Saving with the rollout on also starts it again after a guard stopped it.
 */
export function RolloutSettingsDialog({ open, onOpenChange, rollout }: RolloutSettingsDialogProps) {
  const t = useT();
  const enabledId = useId();
  const providerId = useId();
  const queryClient = useQueryClient();
  const providers = useQuery({ ...queries.aiProvider.list(), enabled: open });
  const [draft, setDraft] = useState<RolloutDraft>(() => rolloutDraftOf(rollout));
  const [openedFor, setOpenedFor] = useState({ open, version: rollout.version });

  if (openedFor.open !== open || openedFor.version !== rollout.version) {
    setOpenedFor({ open, version: rollout.version });
    if (open) {
      setDraft(rolloutDraftOf(rollout));
    }
  }

  const items = useMemo(
    () =>
      (providers.data ?? [])
        .filter(
          (provider) =>
            (provider.enabled && provider.tasks.includes("DocumentExtraction")) ||
            provider.id === rollout.providerId,
        )
        .map((provider) => ({
          value: provider.id,
          label: t("{0} · {1}", provider.name, provider.model),
        })),
    [providers.data, rollout.providerId, t],
  );

  const problems = rolloutProblems(draft);
  const input = rolloutInput(draft, rollout.version);
  const set = (patch: Partial<RolloutDraft>) => setDraft((current) => ({ ...current, ...patch }));

  const save = useApiMutation({
    mutationFn: () => {
      if (!input) {
        throw new Error("The rollout settings have problems");
      }
      return updateExtractionRollout(input);
    },
    onSuccess: async (saved) => {
      toast.success(
        saved.serving ? t("The candidate is serving its share") : t("The rollout is off"),
      );
      queryClient.setQueryData([EXTRACTION_ROLLOUT_KEY], saved);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: [EXTRACTION_ROLLOUT_KEY] }),
        queryClient.invalidateQueries({ queryKey: [EXTRACTION_ROLLOUT_REPORT_KEY] }),
      ]);
      onOpenChange(false);
    },
    resourceName: t("Rollout"),
  });

  const noProviders = providers.isSuccess && items.length === 0;
  const restarting = rollout.haltedAt !== null && rollout.haltedAt !== undefined && draft.enabled;
  const everything = draft.enabled && draft.percent.trim() === String(MAX_ROLLOUT_PERCENT);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Gradual rollout")}</DialogTitle>
          <DialogDescription>
            {t(
              "A share of real documents is read by the candidate and its answer fills the shipment draft. When the candidate cannot answer, production reads the document instead. The guards stop the rollout and send every document back to production when the candidate does worse.",
            )}
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4">
          {noProviders ? (
            <Alert variant="warning" size="sm">
              <AlertDescription>
                {t("No enabled provider is assigned to document extraction.")}
              </AlertDescription>
            </Alert>
          ) : null}
          {restarting ? (
            <Alert variant="info" size="sm">
              <AlertDescription>
                {t(
                  "Saving starts the rollout again and begins a new comparison; what happened before the guard stopped it is no longer counted.",
                )}
              </AlertDescription>
            </Alert>
          ) : null}
          <div className="flex items-center justify-between gap-3">
            <Label htmlFor={enabledId}>{t("Serve the candidate")}</Label>
            <Switch
              id={enabledId}
              checked={draft.enabled}
              onCheckedChange={(enabled) => set({ enabled })}
            />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor={providerId}>{t("Candidate provider")}</Label>
            <Select
              items={items}
              value={draft.providerId || null}
              onValueChange={(value) => set({ providerId: value ?? "" })}
              disabled={items.length === 0}
            >
              <SelectTrigger
                id={providerId}
                aria-label={t("Candidate provider")}
                aria-invalid={problems.providerId !== undefined}
              >
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
            <p className="text-muted-foreground text-xs">
              {problems.providerId
                ? t(problems.providerId)
                : t(
                    "Choose one that did better than production in shadow traffic. Changing it starts a new comparison.",
                  )}
            </p>
          </div>
          <NumberField
            label={t("Share of documents (%)")}
            min={MIN_ROLLOUT_PERCENT}
            max={MAX_ROLLOUT_PERCENT}
            value={draft.percent}
            problem={problems.percent}
            hint={t(
              "The same documents are chosen every time for a given share, so raising it adds documents rather than swapping them.",
            )}
            onChange={(percent) => set({ percent })}
          />
          <div className="grid gap-4 sm:grid-cols-2">
            <NumberField
              label={t("Stop below production's accuracy by (pts)")}
              min={MIN_ACCURACY_DROP_POINTS}
              max={MAX_ACCURACY_DROP_POINTS}
              value={draft.maxAccuracyDropPoints}
              problem={problems.maxAccuracyDropPoints}
              onChange={(maxAccuracyDropPoints) => set({ maxAccuracyDropPoints })}
            />
            <NumberField
              label={t("Stop above production's unusable answers by (pts)")}
              min={MIN_REJECTION_INCREASE_POINTS}
              max={MAX_REJECTION_INCREASE_POINTS}
              value={draft.maxRejectionIncreasePoints}
              problem={problems.maxRejectionIncreasePoints}
              onChange={(maxRejectionIncreasePoints) => set({ maxRejectionIncreasePoints })}
            />
          </div>
          {everything ? (
            <Alert variant="warning" size="sm">
              <AlertDescription>
                {t(
                  "At 100 percent no document is left on production to compare against, so the guards cannot stop the rollout.",
                )}
              </AlertDescription>
            </Alert>
          ) : null}
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            disabled={input === null}
            isLoading={save.isPending}
            onClick={() => save.mutate(undefined)}
          >
            {t("Save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
