/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { CheckboxInput } from "@/components/common/fields/checkbox";
import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { ErrorLoadingData } from "@/components/common/table/data-table-components";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useDispatchControl } from "@/hooks/useQueries";
import { serviceIncidentControlChoices } from "@/lib/choices";
import { dispatchControlSchema } from "@/lib/validations/DispatchSchema";
import type {
  DispatchControlFormValues,
  DispatchControl as DispatchControlType,
} from "@/types/dispatch";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";

function DispatchControlForm({
  dispatchControl,
}: {
  dispatchControl: DispatchControlType;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { t } = useTranslation(["admin.dispatchcontrol", "common"]);

  const { control, handleSubmit, reset } = useForm<DispatchControlFormValues>({
    resolver: yupResolver(dispatchControlSchema),
    defaultValues: dispatchControl,
  });

  const mutation = useCustomMutation<DispatchControlFormValues>(
    control,
    {
      method: "PUT",
      path: `/dispatch-control/${dispatchControl.id}/`,
      successMessage: t("formSuccessMessage"),
      queryKeysToInvalidate: ["dispatchControl"],
      errorMessage: t("formErrorMessage"),
    },
    () => setIsSubmitting(false),
  );

  const onSubmit = (values: DispatchControlFormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);

    reset(values);
  };

  return (
    <form
      className="border-border bg-card m-4 border sm:rounded-xl md:col-span-2"
      onSubmit={handleSubmit(onSubmit)}
    >
      <div className="px-4 py-6 sm:p-8">
        <div className="grid max-w-3xl grid-cols-1 gap-x-6 gap-y-8 sm:grid-cols-1 md:grid-cols-2 lg:grid-cols-6">
          <div className="col-span-3">
            <SelectInput
              name="recordServiceIncident"
              control={control}
              options={serviceIncidentControlChoices}
              rules={{ required: true }}
              label={t("fields.recordServiceIncident.label")}
              placeholder={t("fields.recordServiceIncident.placeholder")}
              description={t("fields.recordServiceIncident.description")}
            />
          </div>
          <div className="col-span-3">
            <InputField
              name="gracePeriod"
              control={control}
              type="number"
              rules={{ required: true }}
              label={t("fields.gracePeriod.label")}
              placeholder={t("fields.gracePeriod.placeholder")}
              description={t("fields.gracePeriod.description")}
            />
          </div>
          <div className="col-span-3">
            <InputField
              name="deadheadTarget"
              type="number"
              control={control}
              rules={{ required: true }}
              label={t("fields.deadheadTarget.label")}
              placeholder={t("fields.deadheadTarget.placeholder")}
              description={t("fields.deadheadTarget.description")}
            />
          </div>
          <div className="col-span-3">
            <InputField
              name="maxShipmentWeightLimit"
              type="number"
              control={control}
              rules={{ required: true }}
              label={t("fields.maxShipmentWeightLimit.label")}
              placeholder={t("fields.maxShipmentWeightLimit.placeholder")}
              description={t("fields.maxShipmentWeightLimit.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="enforceWorkerAssign"
              control={control}
              label={t("fields.enforceWorkerAssign.label")}
              description={t("fields.enforceWorkerAssign.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="trailerContinuity"
              control={control}
              label={t("fields.trailerContinuity.label")}
              description={t("fields.trailerContinuity.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="dupeTrailerCheck"
              control={control}
              label={t("fields.dupeTrailerCheck.label")}
              description={t("fields.dupeTrailerCheck.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="regulatoryCheck"
              control={control}
              label={t("fields.regulatoryCheck.label")}
              description={t("fields.regulatoryCheck.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="prevShipmentOnHold"
              control={control}
              label={t("fields.prevShipmentsOnHold.label")}
              description={t("fields.prevShipmentsOnHold.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="workerTimeAwayRestriction"
              control={control}
              label={t("fields.workerTimeAwayRestriction.label")}
              description={t("fields.workerTimeAwayRestriction.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="tractorWorkerFleetConstraint"
              control={control}
              label={t("fields.tractorWorkerFleetConstraint.label")}
              description={t("fields.tractorWorkerFleetConstraint.description")}
            />
          </div>
          <div className="col-span-3">
            <CheckboxInput
              name="maintenanceCompliance"
              control={control}
              label={t("fields.maintenanceCompliance.label")}
              description={t("fields.maintenanceCompliance.description")}
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
          variant="outline"
          disabled={isSubmitting}
        >
          {t("buttons.cancel", { ns: "common" })}
        </Button>
        <Button type="submit" isLoading={isSubmitting}>
          {t("buttons.save", { ns: "common" })}
        </Button>
      </div>
    </form>
  );
}

export default function DispatchControl() {
  const { data, isLoading, isError } = useDispatchControl();
  const { t } = useTranslation("admin.dispatchcontrol");

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
          <ErrorLoadingData />
        </div>
      ) : (
        data && <DispatchControlForm dispatchControl={data} />
      )}
    </div>
  );
}
