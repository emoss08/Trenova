import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import {
  EXTRACTION_SHADOW_REPORT_KEY,
  EXTRACTION_SHADOW_SETTINGS_KEY,
  updateExtractionShadowSettings,
  type ExtractionShadowSettings,
} from "@/lib/graphql/extraction-shadow";
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
import { useId, useMemo, useState } from "react";
import { toast } from "sonner";
import {
  hasProblems,
  MAX_DAILY_LIMIT,
  MAX_SAMPLE_PERCENT,
  shadowSettingsProblems,
  type ShadowSettingsDraft,
} from "./shadow-model";

type ShadowSettingsDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  settings: ExtractionShadowSettings;
};

function draftOf(settings: ExtractionShadowSettings): ShadowSettingsDraft {
  return {
    enabled: settings.enabled,
    providerId: settings.providerId ?? "",
    samplePercent: String(settings.samplePercent),
    dailyLimit: String(settings.dailyLimit),
  };
}

/**
 * Chooses the candidate that shadows production extraction. Only a provider
 * enabled for document extraction can be chosen; the one production already
 * routes to is never shadowed, because it would be compared with itself.
 */
export function ShadowSettingsDialog({ open, onOpenChange, settings }: ShadowSettingsDialogProps) {
  const t = useT();
  const enabledId = useId();
  const providerId = useId();
  const sampleId = useId();
  const limitId = useId();
  const queryClient = useQueryClient();
  const providers = useQuery({ ...queries.aiProvider.list(), enabled: open });
  const [draft, setDraft] = useState<ShadowSettingsDraft>(() => draftOf(settings));
  const [openedFor, setOpenedFor] = useState({ open, version: settings.version });

  if (openedFor.open !== open || openedFor.version !== settings.version) {
    setOpenedFor({ open, version: settings.version });
    if (open) {
      setDraft(draftOf(settings));
    }
  }

  const items = useMemo(
    () =>
      (providers.data ?? [])
        .filter(
          (provider) =>
            (provider.enabled && provider.tasks.includes("DocumentExtraction")) ||
            provider.id === settings.providerId,
        )
        .map((provider) => ({
          value: provider.id,
          label: t("{0} · {1}", provider.name, provider.model),
        })),
    [providers.data, settings.providerId, t],
  );

  const problems = shadowSettingsProblems(draft);
  const invalid = hasProblems(problems);

  const save = useApiMutation({
    mutationFn: () =>
      updateExtractionShadowSettings({
        enabled: draft.enabled,
        providerId: draft.providerId === "" ? null : draft.providerId,
        samplePercent: Number(draft.samplePercent.trim()),
        dailyLimit: Number(draft.dailyLimit.trim()),
        version: settings.version,
      }),
    onSuccess: async (saved) => {
      toast.success(saved.enabled ? t("Shadow traffic is on") : t("Shadow traffic is off"));
      queryClient.setQueryData([EXTRACTION_SHADOW_SETTINGS_KEY], saved);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: [EXTRACTION_SHADOW_SETTINGS_KEY] }),
        queryClient.invalidateQueries({ queryKey: [EXTRACTION_SHADOW_REPORT_KEY] }),
      ]);
      onOpenChange(false);
    },
    resourceName: t("Shadow settings"),
  });

  const noProviders = providers.isSuccess && items.length === 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Shadow traffic")}</DialogTitle>
          <DialogDescription>
            {t(
              "A share of production extractions is also sent to the candidate. Its answer is kept, never applied, and scored against the shipment a person confirms, beside production's answer for the same document. The calls spend from the evaluation budget.",
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
          <div className="flex items-center justify-between gap-3">
            <Label htmlFor={enabledId}>{t("Shadow production extraction")}</Label>
            <Switch
              id={enabledId}
              checked={draft.enabled}
              onCheckedChange={(enabled) => setDraft((current) => ({ ...current, enabled }))}
            />
          </div>
          <div className="grid gap-1.5">
            <Label htmlFor={providerId}>{t("Candidate provider")}</Label>
            <Select
              items={items}
              value={draft.providerId || null}
              onValueChange={(value) =>
                setDraft((current) => ({ ...current, providerId: value ?? "" }))
              }
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
                    "Give it a priority after the current extraction provider, so production never routes to it.",
                  )}
            </p>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="grid gap-1.5">
              <Label htmlFor={sampleId}>{t("Share of extractions (%)")}</Label>
              <Input
                id={sampleId}
                type="number"
                inputMode="numeric"
                min={1}
                max={MAX_SAMPLE_PERCENT}
                value={draft.samplePercent}
                aria-invalid={problems.samplePercent !== undefined}
                onChange={(event) =>
                  setDraft((current) => ({ ...current, samplePercent: event.target.value }))
                }
              />
              {problems.samplePercent ? (
                <p className="text-danger text-xs">{t(problems.samplePercent)}</p>
              ) : null}
            </div>
            <div className="grid gap-1.5">
              <Label htmlFor={limitId}>{t("Most per 24 hours")}</Label>
              <Input
                id={limitId}
                type="number"
                inputMode="numeric"
                min={1}
                max={MAX_DAILY_LIMIT}
                value={draft.dailyLimit}
                aria-invalid={problems.dailyLimit !== undefined}
                onChange={(event) =>
                  setDraft((current) => ({ ...current, dailyLimit: event.target.value }))
                }
              />
              {problems.dailyLimit ? (
                <p className="text-danger text-xs">{t(problems.dailyLimit)}</p>
              ) : null}
            </div>
          </div>
          <p className="text-muted-foreground text-xs">
            {t(
              "The same documents are chosen every time for a given share, so raising it adds documents rather than swapping them.",
            )}
          </p>
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            disabled={invalid}
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
