import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { TextareaField } from "@/components/common/fields/textarea";
import { Skeleton } from "@/components/ui/skeleton";
import { TitleWithTooltip } from "@/components/ui/title-with-tooltip";
import { useUsers } from "@/hooks/useQueries";
import { entryMethodChoices } from "@/lib/choices";
import { validateBOLNumber } from "@/services/ShipmentRequestService";
import { ShipmentControl, ShipmentFormValues } from "@/types/order";
import { debounce } from "lodash-es";
import { useEffect } from "react";
import { useFormContext } from "react-hook-form";
import { useTranslation } from "react-i18next";

export default function DispatchInformation({
  shipmentControlData,
  isShipmentControlLoading,
}: {
  shipmentControlData: ShipmentControl;
  isShipmentControlLoading: boolean;
}) {
  const { t } = useTranslation("shipment.addshipment");
  const { control, watch, setError } = useFormContext<ShipmentFormValues>();

  const {
    selectUsersData,
    isError: isUserError,
    isLoading: isUsersLoading,
  } = useUsers();

  const bolValue = watch("bolNumber");

  // Check for duplicate BOL number if shipmentControlData.checkForDuplicateBol is true
  useEffect(() => {
    const debounceValidation = debounce(async () => {
      try {
        const response = await validateBOLNumber(bolValue);
        if (response.valid === false) {
          setError("bolNumber", {
            type: "manual",
            message: response.message,
          });
        }
      } catch (error) {
        console.error("[Trenova] Error validating BOL number", error);
      }
    }, 500);

    if (bolValue && shipmentControlData.checkForDuplicateBol) {
      debounceValidation();
    }

    return () => {
      debounceValidation.cancel();
    };
  }, [bolValue]);

  if (isShipmentControlLoading) {
    return <Skeleton className="h-[40vh]" />;
  }

  return (
    <div className="rounded-md border border-border bg-card">
      <div className="flex justify-center rounded-t-md border-b border-border bg-background p-2">
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
