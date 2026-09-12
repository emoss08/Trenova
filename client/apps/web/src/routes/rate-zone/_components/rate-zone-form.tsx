import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { rateZoneKindChoices, statusChoices } from "@/lib/choices";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import type { RateZone } from "@trenova/shared/types/rate";
import { useFormContext } from "react-hook-form";
import { ZoneMemberEditor } from "./zone-member-editor";

export function RateZoneForm() {
  const t = useT();

  const { control } = useFormContext<RateZone>();

  return (
    <div className="space-y-6">
      <FormSection
        title={t("General Information")}
        description={t("How this zone is identified and whether it is live.")}
        className="border-b pb-4"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="status"
              label={t("Status")}
              placeholder={t("Status")}
              description={t(
                "An inactive zone stops matching, and every lane written against it stops with it",
              )}
              options={statusChoices}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="kind"
              label={t("Zone Kind")}
              placeholder={t("Select kind")}
              description={t(
                "What sort of area this is, which is how somebody else reads it later",
              )}
              options={rateZoneKindChoices}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="code"
              label={t("Code")}
              placeholder={t("SE")}
              description={t("The short name lanes and tariffs refer to")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="name"
              label={t("Name")}
              placeholder={t("Southeast")}
              description={t("What this area is called out loud")}
            />
          </FormControl>
          <FormControl cols="full">
            <TextareaField
              control={control}
              name="description"
              label={t("Description")}
              placeholder={t("Atlantic and Gulf states from Virginia through Louisiana")}
              description={t(
                "What the zone actually covers, for the next person deciding whether to reuse it",
              )}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Places")}
        description={t("The states, cities, postal codes, and locations this zone is a union of.")}
      >
        <ZoneMemberEditor />
      </FormSection>
    </div>
  );
}
