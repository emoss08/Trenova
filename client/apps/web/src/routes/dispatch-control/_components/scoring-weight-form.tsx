import { useT } from "@trenova/shared/i18n/use-t";
import { NumberField } from "@/components/fields/number-field";
import {
  PRESET_SCORING_WEIGHTS,
  SCORING_FACTOR_META,
  type DispatchControl,
} from "@/types/dispatch-control";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useFormContext, useWatch } from "react-hook-form";

export function ScoringWeightForm() {
  const t = useT();

  const { control } = useFormContext<DispatchControl>();

  const strategy = useWatch({
    control,
    name: "autoAssignmentStrategy",
  });

  const presets = PRESET_SCORING_WEIGHTS[strategy] ?? PRESET_SCORING_WEIGHTS.Proximity;

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Candidate Scoring Weights")}</CardTitle>
        <CardDescription>
          {t("Fine-tune how heavily each factor counts when ranking drivers for a shipment. Weights run from 0 to 10 and are relative to each other; a factor set to 0 is ignored entirely. Leave a field empty to inherit the {0} strategy's preset, shown as the placeholder.", strategy)}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={2}>
          {SCORING_FACTOR_META.map((factor) => (
            <FormControl key={factor.key} className="min-h-[3em]">
              <NumberField
                control={control}
                name={`scoringWeights.${factor.key}`}
                label={t(factor.label)}
                description={t(factor.description)}
                placeholder={presets[factor.key].toFixed(1)}
                decimalScale={1}
              />
            </FormControl>
          ))}
        </FormGroup>
      </CardContent>
    </Card>
  );
}
