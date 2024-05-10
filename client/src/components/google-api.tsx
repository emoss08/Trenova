import { CheckboxInput } from "@/components/common/fields/checkbox";
import { PasswordField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { ErrorLoadingData } from "@/components/common/table/data-table-components";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useGoogleAPI } from "@/hooks/useQueries";
import { routeDistanceUnitChoices, routeModelChoices } from "@/lib/choices";
import { googleAPISchema } from "@/lib/validations/OrganizationSchema";
import type {
  GoogleAPIFormValues,
  GoogleAPI as GoogleAPIType,
} from "@/types/organization";
import { yupResolver } from "@hookform/resolvers/yup";
import { InfoCircledIcon } from "@radix-ui/react-icons";
import { useForm } from "react-hook-form";
import { Trans, useTranslation } from "react-i18next";
import { ExternalLink } from "./ui/link";

function GoogleAPIAlert() {
  const { t } = useTranslation("admin.googleapi");

  return (
    <Alert className="mb-5">
      <InfoCircledIcon />
      <AlertTitle>{t("alert.title")}</AlertTitle>
      <AlertDescription>
        <ul className="list-disc">
          <li>
            <Trans
              components={[<strong />]}
              i18nKey="alert.list.apiKey.description"
              t={t}
            />
            <ExternalLink href="https://developers.google.com/maps/documentation/javascript/get-api-key">
              {t("alert.list.apiKey.link")}
            </ExternalLink>
          </li>
          <li>
            <Trans
              components={[<strong />]}
              i18nKey="alert.list.mileageUnit.description"
              t={t}
            />
            <ExternalLink href="https://support.google.com/merchants/answer/14156166?hl=en">
              {t("alert.list.mileageUnit.link")}
            </ExternalLink>
          </li>
          <li>
            <Trans
              components={[<strong />]}
              i18nKey="alert.list.trafficModel.description"
              t={t}
            />
            <ExternalLink href="https://developers.google.com/maps/documentation/distance-matrix/distance-matrix#traffic_model">
              {t("alert.list.trafficModel.link")}
            </ExternalLink>
          </li>
        </ul>
      </AlertDescription>
    </Alert>
  );
}

function GoogleApiForm({ googleApi }: { googleApi: GoogleAPIType }) {
  const { t } = useTranslation(["admin.googleapi", "common"]);

  const { control, handleSubmit, reset } = useForm<GoogleAPIFormValues>({
    resolver: yupResolver(googleAPISchema),
    defaultValues: googleApi,
  });

  const mutation = useCustomMutation<GoogleAPIFormValues>(control, {
    method: "PUT",
    path: `/google-api/${googleApi.id}`, // Does not require an ID
    successMessage: t("formSuccessMessage"),
    queryKeysToInvalidate: "googleAPI",
    reset,
    errorMessage: t("formErrorMessage"),
  });

  const onSubmit = (values: GoogleAPIFormValues) => {
    mutation.mutate(values);
    reset(values);
  };

  return (
    <form
      className="border-border bg-card m-4 border sm:rounded-xl md:col-span-2"
      onSubmit={handleSubmit(onSubmit)}
    >
      <div className="px-4 py-6 sm:p-8">
        <GoogleAPIAlert />
        <div className="grid max-w-3xl grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-6">
          <div className="col-span-3">
            <SelectInput
              name="mileageUnit"
              control={control}
              options={routeDistanceUnitChoices}
              rules={{ required: true }}
              label={t("fields.mileageUnit.label")}
              placeholder={t("fields.mileageUnit.placeholder")}
              description={t("fields.mileageUnit.description")}
            />
          </div>
          <div className="col-span-3">
            <SelectInput
              name="trafficModel"
              control={control}
              options={routeModelChoices}
              rules={{ required: true }}
              label={t("fields.trafficModel.label")}
              placeholder={t("fields.trafficModel.placeholder")}
              description={t("fields.trafficModel.description")}
            />
          </div>
          <div className="col-span-4">
            <PasswordField
              name="apiKey"
              control={control}
              rules={{ required: true }}
              label={t("fields.apiKey.label")}
              placeholder={t("fields.apiKey.placeholder")}
              description={t("fields.apiKey.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="addCustomerLocation"
              control={control}
              label={t("fields.addCustomerLocation.label")}
              description={t("fields.addCustomerLocation.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="addLocation"
              control={control}
              label={t("fields.addLocation.label")}
              description={t("fields.addLocation.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="autoGeocode"
              control={control}
              label={t("fields.autoGeocode.label")}
              description={t("fields.autoGeocode.description")}
            />
          </div>
        </div>
      </div>
      <div className="border-muted flex items-center justify-end gap-x-4 border-t p-4 sm:px-8">
        <Button
          onClick={(e) => {
            e.preventDefault();
            reset();
          }}
          type="button"
          variant="ghost"
          disabled={mutation.isPending}
        >
          {t("buttons.cancel", { ns: "common" })}
        </Button>
        <Button type="submit" isLoading={mutation.isPending}>
          {t("buttons.save", { ns: "common" })}
        </Button>
      </div>
    </form>
  );
}

export default function GoogleApi() {
  const { t } = useTranslation("admin.googleapi");
  const { googleAPIData, isLoading, isError } = useGoogleAPI();

  return (
    <div className="grid grid-cols-1 gap-8 md:grid-cols-3">
      <div className="px-4 sm:px-0">
        <h2 className="text-foreground text-base font-semibold leading-7">
          {t("title")}
        </h2>
        <p className="text-muted-foreground mt-1 text-sm leading-6">
          {t("subTitle")}
        </p>
      </div>
      {isLoading ? (
        <div className="bg-background ring-muted m-4 ring-1 sm:rounded-xl md:col-span-2">
          <Skeleton className="h-screen w-full" />
        </div>
      ) : isError ? (
        <div className="bg-background ring-muted m-4 p-8 ring-1 sm:rounded-xl md:col-span-2">
          <ErrorLoadingData message="Failed to load Google API control." />
        </div>
      ) : (
        (googleAPIData as GoogleAPIType) && (
          <GoogleApiForm googleApi={googleAPIData as GoogleAPIType} />
        )
      )}
    </div>
  );
}
