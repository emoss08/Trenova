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

import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { useShipmentControl } from "@/hooks/useQueries";
import { ShipmentStatusChoiceProps } from "@/lib/choices";
import { cleanObject } from "@/lib/utils";
import { useUserStore } from "@/stores/AuthStore";
import { ShipmentFormValues, ShipmentPageTab } from "@/types/order";
import {
  faBoxTaped,
  faCommentQuote,
  faFile,
  faMoneyBillTransfer,
  faOctagon,
  faWebhook,
} from "@fortawesome/pro-duotone-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { yupResolver } from "@hookform/resolvers/yup";
import { Suspense, lazy, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { Skeleton } from "../ui/skeleton";

import { stopSchema } from "@/lib/validations/StopSchema";
import * as yup from "yup";
import { ShipmentAsideMenu } from "./add-shipment/nav/nav-bar";

// Lazy load the tabs
const GeneralTab = lazy(
  () =>
    import("@/components/shipment-management/add-shipment/general-info-tab"),
);
const BillingTab = lazy(
  () =>
    import("@/components/shipment-management/add-shipment/billing-info-tab"),
);
const StopsTab = lazy(
  () => import("@/components/shipment-management/add-shipment/stop-info-tab"),
);

const tabs: Record<string, ShipmentPageTab> = {
  general: {
    name: "General Information",
    component: GeneralTab,
    icon: <FontAwesomeIcon icon={faBoxTaped} />,
    description: "General information about the shipment",
  },
  stops: {
    name: "Additional Stops",
    component: StopsTab,
    icon: <FontAwesomeIcon icon={faOctagon} />,
    description: "Stops for the shipment",
  },
  billing: {
    name: "Billing Information",
    component: BillingTab,
    icon: <FontAwesomeIcon icon={faMoneyBillTransfer} />,
    description: "Billing information for the shipment",
  },
  comments: {
    name: "Comments",
    component: () => <div>Comments</div>,
    icon: <FontAwesomeIcon icon={faCommentQuote} />,
    description: "Comments about the shipment",
  },
  documents: {
    name: "Documents",
    component: () => <div>Documents</div>,
    icon: <FontAwesomeIcon icon={faFile} />,
    description: "Documents for the shipment",
  },
  edi: {
    name: "EDI Information",
    component: () => <div>EDI Information</div>,
    icon: <FontAwesomeIcon icon={faWebhook} />,
    description: "EDI information for the shipment",
  },
};

export default function AddShipment() {
  const [activeTab, setActiveTab] = useState<string>("general");
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);
  const [user] = useUserStore.use("user");
  const { shipmentControlData, isLoading: isShipmentControlLoading } =
    useShipmentControl();

  if (isShipmentControlLoading && !shipmentControlData && !user) {
    return <Skeleton className="h-[100vh] w-full" />;
  }

  // Shipment Form validation schema
  const shipmentSchema: yup.ObjectSchema<ShipmentFormValues> = yup
    .object()
    .shape({
      proNumber: yup.string().required("Pro number is required."),
      shipmentType: yup.string().required("Shipment type is required."),
      serviceType: yup.string().required("Service type is required."),
      status: yup
        .string<ShipmentStatusChoiceProps>()
        .required("Status is required."),
      revenueCode:
        shipmentControlData && shipmentControlData.enforceRevCode
          ? yup.string().required("Revenue code is required.")
          : yup.string().notRequired(),
      originLocation: yup
        .string()
        .test({
          name: "originLocation",
          test: function (value) {
            if (!value) {
              return this.parent.originAddress !== "";
            }
            return true;
          },
          message: "Origin location is required.",
        })
        .test({
          name: "originLocation",
          test: function (value) {
            if (
              shipmentControlData &&
              shipmentControlData.enforceOriginDestination
            ) {
              if (value === this.parent.destinationLocation) {
                return false;
              }
            }
            return true;
          },
          message: "Origin and Destination locations cannot be the same.",
        }),
      originAddress: yup.string().test({
        name: "originAddress",
        test: function (value) {
          if (!value) {
            return false;
          }
          return true;
        },
        message: "Origin address is required.",
      }),
      originAppointmentWindowStart: yup
        .string()
        .required("Origin appointment window start is required."),
      originAppointmentWindowEnd: yup
        .string()
        .required("Origin appointment window end is required."),
      destinationLocation: yup
        .string()
        .test({
          name: "destinationLocation",
          test: function (value) {
            if (!value) {
              return this.parent.destinationAddress !== "";
            }
            return true;
          },
          message: "Destination location is required.",
        })
        .test({
          name: "destinationLocation",
          test: function (value) {
            if (
              shipmentControlData &&
              shipmentControlData.enforceOriginDestination
            ) {
              if (value === this.parent.originLocation) {
                return false;
              }
            }
            return true;
          },
          message: "Origin and Destination locations cannot be the same.",
        }),
      destinationAddress: yup.string().test({
        name: "destinationAddress",
        test: function (value) {
          if (!value) {
            return this.parent.destinationLocation !== "";
          }
          return true;
        },
        message: "Destination address is required.",
      }),
      destinationAppointmentWindowStart: yup
        .string()
        .required("Destination appointment window start is required."),
      destinationAppointmentWindowEnd: yup
        .string()
        .required("Destination appointment window end is required."),
      ratingUnits: yup.number().required("Rating units is required."),
      rate: yup.string().notRequired(),
      mileage: yup.number().notRequired(),
      otherChargeAmount: yup.string().notRequired(),
      freightChargeAmount: yup.string().notRequired(),
      rateMethod: yup.string().notRequired(),
      customer: yup.string().required("Customer is required."),
      pieces: yup.number().notRequired(),
      weight: yup.string().notRequired(),
      readyToBill: yup.boolean().required("Ready to bill is required."),
      trailer: yup.string().notRequired(),
      trailerType: yup.string().required("Trailer type is required."),
      tractorType: yup.string().notRequired(),
      commodity:
        shipmentControlData && shipmentControlData.enforceCommodity
          ? yup.string().required("Commodity is required.")
          : yup.string().notRequired(),
      hazardousMaterial: yup.string().notRequired(),
      temperatureMin: yup.string().notRequired(),
      temperatureMax: yup.string().notRequired(),
      bolNumber: yup.string().required("BOL number is required."),
      consigneeRefNumber: yup.string().notRequired(),
      comment: yup
        .string()
        .max(100, "Comment must be less than 100 characters.")
        .notRequired(),
      voidedComm: yup.string().notRequired(),
      autoRate: yup.boolean().required("Auto rate is required."),
      formulaTemplate: yup.string().notRequired(),
      enteredBy: yup.string().required("Entered by is required."),
      subTotal: yup.string().notRequired(),
      serviceTye: yup.string().notRequired(),
      entryMethod: yup.string().required("Entry method is required."),
      copyAmount: yup.number().required("Copy amount is required."),
      stops: yup.array().of(stopSchema).notRequired(),
    });

  // Form state and methods
  const shipmentForm = useForm<ShipmentFormValues>({
    resolver: yupResolver(shipmentSchema),
    defaultValues: {
      status: "N",
      proNumber: "",
      originLocation: "",
      originAddress: "",
      destinationLocation: "",
      destinationAddress: "",
      bolNumber: "",
      entryMethod: "MANUAL",
      comment: "",
      ratingUnits: 1,
      autoRate: false,
      readyToBill: false,
      copyAmount: 0,
      enteredBy: user?.id || "",
      commodity: "",
      temperatureMin: "",
      temperatureMax: "",
      hazardousMaterial: "",
      tractorType: "",
      trailerType: "",
      stops: [],
    },
  });

  const { control, reset, handleSubmit } = shipmentForm;

  // Mutation
  const mutation = useCustomMutation<ShipmentFormValues>(
    control,
    {
      method: "POST",
      path: "/shipments/",
      successMessage: "Shipment created successfully.",
      queryKeysToInvalidate: ["shipments"],
      closeModal: true,
      errorMessage: "Failed to create new shipment.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: ShipmentFormValues) => {
    const cleanedValues = cleanObject(values);
    setIsSubmitting(true);
    console.info("Form values", cleanedValues);
    mutation.mutate(cleanedValues);
    console.info("Resetting form");
    reset();
  };

  const ActiveTabComponent = tabs[activeTab].component;

  return (
    <div className="flex-1 items-start md:grid md:grid-cols-[220px_minmax(0,1fr)] md:gap-6 lg:grid-cols-[300px_minmax(0,1fr)] lg:gap-10">
      <ShipmentAsideMenu
        tabs={tabs}
        activeTab={activeTab}
        setActiveTab={setActiveTab}
      />
      <div className="relative mb-10 lg:gap-10">
        <FormProvider {...shipmentForm}>
          <form
            onSubmit={handleSubmit(onSubmit)}
            className="flex h-full flex-col overflow-y-visible"
          >
            <Suspense fallback={<Skeleton className="h-[100vh] w-full" />}>
              <ActiveTabComponent />
            </Suspense>
            <div className="mt-4 flex flex-col-reverse pt-4 sm:flex-row sm:justify-end sm:space-x-2">
              <Button disabled type="button" variant="outline">
                Save & Add Another
              </Button>
              <Button type="submit" isLoading={isSubmitting}>
                Save
              </Button>
            </div>
          </form>
        </FormProvider>
      </div>
    </div>
  );
}
