import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { TextareaField } from "@/components/common/fields/textarea";
import { TitleWithTooltip } from "@/components/ui/title-with-tooltip";
import { useUsers } from "@/hooks/useQueries";
import { entryMethodChoices } from "@/lib/choices";
import { ShipmentFormValues } from "@/types/order";
import { Control } from "react-hook-form";
import { useTranslation } from "react-i18next";

export function DispatchInformation({
  control,
}: {
  control: Control<ShipmentFormValues>;
}) {
  const { t } = useTranslation("shipment.addshipment");

  const {
    selectUsersData,
    isError: isUserError,
    isLoading: isUsersLoading,
  } = useUsers();

  return (
    <div className="border-border bg-card rounded-md border">
      <div className="border-border bg-background flex justify-center rounded-t-md border-b p-2">
        <TitleWithTooltip
          title={t("card.additionalInfo.label")}
          tooltip={t("card.additionalInfo.description")}
        />
      </div>
      <div className="grid grid-cols-1 gap-x-6 gap-y-4 p-4 md:grid-cols-2">
        <div className="col-span-1">
          <InputField
            name="consigneeRefNumber"
            control={control}
            label={t("card.additionalInfo.fields.consigneeRefNumber.label")}
            placeholder={t(
              "card.additionalInfo.fields.consigneeRefNumber.placeholder",
            )}
            description={t(
              "card.additionalInfo.fields.consigneeRefNumber.description",
            )}
          />
        </div>
        <div className="col-span-1">
          <InputField
            name="bolNumber"
            control={control}
            rules={{ required: true }}
            label={t("card.additionalInfo.fields.bolNumber.label")}
            placeholder={t("card.additionalInfo.fields.bolNumber.placeholder")}
            description={t("card.additionalInfo.fields.bolNumber.description")}
          />
        </div>
        <div className="col-span-1">
          <SelectInput
            name="entryMethod"
            control={control}
            options={entryMethodChoices}
            isReadOnly
            rules={{ required: true }}
            label={t("card.additionalInfo.fields.entryMethod.label")}
            placeholder={t(
              "card.additionalInfo.fields.entryMethod.placeholder",
            )}
            description={t(
              "card.additionalInfo.fields.entryMethod.description",
            )}
          />
        </div>
        <div className="col-span-1">
          <SelectInput
            name="enteredBy"
            options={selectUsersData}
            isLoading={isUsersLoading}
            isFetchError={isUserError}
            control={control}
            isReadOnly
            rules={{ required: true }}
            label={t("card.additionalInfo.fields.enteredBy.label")}
            placeholder={t("card.additionalInfo.fields.enteredBy.placeholder")}
            description={t("card.additionalInfo.fields.enteredBy.description")}
          />
        </div>
        <div className="col-span-2">
          <TextareaField
            name="comment"
            control={control}
            label={t("card.additionalInfo.fields.comment.label")}
            placeholder={t("card.additionalInfo.fields.comment.placeholder")}
            description={t("card.additionalInfo.fields.comment.description")}
          />
        </div>
      </div>
    </div>
  );
}
