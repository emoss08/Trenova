import { useT } from "@trenova/shared/i18n/use-t";
import { TractorAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { IftaJurisdictionSelectField } from "@/components/fields/ifta-jurisdiction-select-field";
import { NumberField } from "@/components/fields/number-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { InfoPopover } from "@/components/info-popover";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import {
  IFTA_MILEAGE_SOURCE_LABELS,
  type IftaMileageSource,
} from "@trenova/shared/types/fuel-ifta-enums";
import {
  IFTA_MILEAGE_NOTES_MAX,
  IFTA_MILES_SCALE,
  type IftaMileageEntryFormValues,
} from "@trenova/shared/types/ifta-jurisdiction-mileage";
import { RouteIcon } from "lucide-react";
import { useFormContext } from "react-hook-form";

type IftaJurisdictionMileageFormProps = {
  isEdit: boolean;
  computed: boolean;
  source?: IftaMileageSource;
  shipmentMoveId?: string | null;
};

export function IftaJurisdictionMileageForm({
  isEdit,
  computed,
  source,
  shipmentMoveId,
}: IftaJurisdictionMileageFormProps) {
  const t = useT();

  const { control } = useFormContext<IftaMileageEntryFormValues>();

  return (
    <div className="flex flex-col gap-6">
      {isEdit && computed ? (
        <Alert>
          <RouteIcon className="size-4" />
          <AlertTitle>
            {t("Written by {0}", source ? IFTA_MILEAGE_SOURCE_LABELS[source].toLowerCase() : "the system")}
          </AlertTitle>
          <AlertDescription>
            {t("These miles came from the distance provider or telematics, not from a person, so they are read-only here. To correct a move's miles, add a manual entry for the same move and it replaces these rows on the return. {0}", shipmentMoveId ? ` Move ${shipmentMoveId}.` : "")}
          </AlertDescription>
        </Alert>
      ) : null}

      <FormSection
        title={t("Where and when")}
        description={t("Which tractor ran the miles, in which jurisdiction, and on what day.")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <TractorAutocompleteField<IftaMileageEntryFormValues>
              control={control}
              name="tractorId"
              label={t("Tractor")}
              rules={{ required: true }}
              placeholder={t("Select a tractor")}
              disabled={computed}
              description={t("The unit that ran the miles. Its fuel type decides which fleet MPG applies.")}
            />
          </FormControl>
          <FormControl>
            <IftaJurisdictionSelectField<IftaMileageEntryFormValues>
              control={control}
              name="jurisdictionId"
              label={t("Jurisdiction")}
              rules={{ required: true }}
              placeholder={t("Select a jurisdiction")}
              isReadOnly={computed}
              description={t("The state or province the miles were run in, not where the trip started.")}
            />
          </FormControl>
          <FormControl cols="full">
            <AutoCompleteDateField
              control={control}
              name="traveledAt"
              label={t("Travelled on")}
              rules={{ required: true }}
              placeholder={t("Day the miles were run")}
              readOnly={computed}
              description={t("Fixes the quarter the miles belong to. Cannot be in the future.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection title={t("Miles")} description={t("How far, and whether the tractor was under load.")}>
        <FormGroup cols={2}>
          <FormControl cols="full">
            <NumberField
              control={control}
              name="miles"
              valueType="string"
              decimalScale={IFTA_MILES_SCALE}
              thousandSeparator
              label={
                <span className="inline-flex items-center gap-1">
                  {t("Miles")}
                  <InfoPopover title={t("Manual miles and the return")}>
                    {t("Routed miles are captured from every completed move automatically. A manual entry is for travel the system did not see, such as bobtailing to a shop or repositioning between customers, and is added to the jurisdiction's line on the quarter's return at its next recompute. Manual miles are never subtracted, so do not enter miles a move already carries unless the entry names that move.")}
                  </InfoPopover>
                </span>
              }
              placeholder="412.50"
              rules={{ required: true }}
              readOnly={computed}
              description={t("Above zero, up to two decimals, as read from the odometer or trip sheet.")}
            />
          </FormControl>
          <FormControl cols="full">
            <SwitchField
              control={control}
              name="loaded"
              label={t("Under load")}
              description={t("Leave on when the trailer carried freight. Turn off for empty or bobtail miles.")}
              tooltip={t("Loaded and empty miles are both taxable; the split is kept for the fleet's own reporting.")}
              position="left"
              outlined
              readOnly={computed}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Notes")}
        description={t("Why these miles exist, so an auditor can see the reason without asking.")}
      >
        <FormGroup cols={1}>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="notes"
              label={t("Notes")}
              placeholder={t("e.g. Deadhead from Amarillo to the Dallas yard after the drop")}
              maxLength={IFTA_MILEAGE_NOTES_MAX}
              readOnly={computed}
              description={`Up to ${IFTA_MILEAGE_NOTES_MAX} characters.`}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
