import { SettingsReadout } from "@/components/settings/settings-section";
import type { SequenceConfig, SequenceConfigDocument } from "@/types/sequence-config";
import { AlertTriangleIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useFormContext, useWatch, type FieldPath } from "react-hook-form";
import { defaultConfigForType, locationComponentLabels } from "./sequence-config-constants";
import {
  numbersCanRepeat,
  segmentsText,
  sequenceSegments,
  type PreviewSegment,
  type SequencePart,
} from "./sequence-preview.utils";

const partLabels: Record<Exclude<SequencePart, "component" | "text" | "separator">, string> =
  defineLabels({
    prefix: "Prefix",
    businessUnit: "Business unit",
    year: "Year",
    month: "Month",
    week: "Week",
    day: "Day",
    location: "Location",
    counter: "Counter",
    random: "Random",
    checkDigit: "Check",
  });

function segmentLabel(segment: PreviewSegment): string | null {
  if (segment.part === "component") {
    return segment.component ? locationComponentLabels[segment.component] : null;
  }
  if (segment.part === "text" || segment.part === "separator") return null;
  return partLabels[segment.part];
}

/**
 * A sequence's next number, drawn piece by piece with each piece named under it. It is the
 * sequence's own surface: it changes as the settings under it change, and pressing a piece
 * goes to the setting that puts it there.
 */
export function SequencePreview({ index, now }: { index: number; now: Date }) {
  const t = useT();
  const { control, setFocus, setValue, getValues } = useFormContext<SequenceConfigDocument>();
  const config = useWatch({ control, name: `configs.${index}` }) as SequenceConfig | undefined;

  if (!config) return null;

  const segments = sequenceSegments(config, now);
  const isLocation = config.sequenceType === "location_code";
  const repeats = numbersCanRepeat(config);

  const restoreDefault = () => {
    const current = getValues(`configs.${index}`);
    setValue(
      `configs.${index}`,
      defaultConfigForType(current.sequenceType, {
        id: current.id,
        organizationId: current.organizationId,
        businessUnitId: current.businessUnitId,
        version: current.version,
        createdAt: current.createdAt,
        updatedAt: current.updatedAt,
      }),
      { shouldDirty: true, shouldValidate: true },
    );
  };

  return (
    <SettingsReadout>
      <div className="flex items-start justify-between gap-4">
        <figure className="flex min-w-0 flex-col gap-1">
          <figcaption className="sr-only">{t("Next number: {0}", segmentsText(segments))}</figcaption>
          <ol
            data-slot="sequence-preview"
            aria-hidden
            className="flex min-w-0 flex-wrap items-start font-mono text-xl leading-7 font-medium tracking-tight"
          >
            {segments.map((segment, position) => {
              const label = segmentLabel(segment);
              if (!label || !segment.field) {
                return (
                  <li
                    key={position}
                    className={cn(
                      "whitespace-pre",
                      segment.part === "separator" && "px-px text-foreground-subtle",
                    )}
                  >
                    {segment.text}
                  </li>
                );
              }
              const field = segment.field;
              return (
                <li key={position}>
                  <button
                    type="button"
                    tabIndex={-1}
                    title={label}
                    onClick={() =>
                      setFocus(`configs.${index}.${field}` as FieldPath<SequenceConfigDocument>)
                    }
                    className="flex flex-col items-start rounded-sm px-0.5 transition-colors hover:bg-surface-hover"
                  >
                    <span className="whitespace-pre tabular-nums">{segment.text}</span>
                    <span className="font-sans text-2xs leading-4 font-normal tracking-normal text-foreground-subtle">
                      {label}
                    </span>
                  </button>
                </li>
              );
            })}
          </ol>
        </figure>
        <Button type="button" variant="ghost" size="xs" className="shrink-0" onClick={restoreDefault}>
          {t("Use the default")}
        </Button>
      </div>
      {repeats ? (
        <Alert variant="warning" size="sm">
          <AlertTriangleIcon />
          <AlertDescription>
            {t(
              "Numbers will repeat. The counter starts again at 1 every month, so a number needs both the year and the month to stay unique.",
            )}
          </AlertDescription>
        </Alert>
      ) : (
        <p className="text-xs text-muted-foreground">
          {isLocation
            ? t(
                "The first code for a location named Acme Warehouse in Dallas, TX 75201. Press a part to change it.",
              )
            : t(
                "The first number this month. The counter starts again at 1 every month. Press a part to change it.",
              )}
        </p>
      )}
    </SettingsReadout>
  );
}
