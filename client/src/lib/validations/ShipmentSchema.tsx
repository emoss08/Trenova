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

import {
  ReasonCodeFormValues,
  ServiceTypeFormValues,
  ShipmentControlFormValues,
  ShipmentFormValues,
  ShipmentTypeFormValues,
} from "@/types/shipment";
import * as yup from "yup";
import { useShipmentControl } from "@/hooks/useQueries";
import {
  CodeTypeProps,
  ShipmentEntryMethodChoices,
  ShipmentStatusChoiceProps,
} from "@/lib/choices";
import { StatusChoiceProps } from "@/types";
import { User } from "@/types/accounts";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { stopSchema } from "./StopSchema";

export const shipmentControlSchema: yup.ObjectSchema<ShipmentControlFormValues> =
  yup.object().shape({
    autoRateShipment: yup.boolean().required("Auto Rate Shipments is required"),
    calculateDistance: yup.boolean().required("Calculate Distance is required"),
    enforceRevCode: yup.boolean().required("Enforce Rev Code is required"),
    enforceVoidedComm: yup
      .boolean()
      .required("Enforce Voided Comm is required"),
    generateRoutes: yup.boolean().required("Generate Routes is required"),
    enforceCommodity: yup.boolean().required("Enforce Commodity is required"),
    autoSequenceStops: yup
      .boolean()
      .required("Auto Sequence Stops is required"),
    autoShipmentTotal: yup
      .boolean()
      .required("Auto Shipment Total is required"),
    enforceOriginDestination: yup
      .boolean()
      .required("Enforce Origin Destination is required"),
    checkForDuplicateBol: yup
      .boolean()
      .required("Check for Duplicate BOL is required"),
    removeShipment: yup.boolean().required("Remove Shipment is required"),
    sendPlacardInfo: yup.boolean().required("Send Placard Info is required"),
    enforceHazmatSegRules: yup
      .boolean()
      .required("Enforce Hazmat Seg Rules is required"),
  });

export const serviceTypeSchema: yup.ObjectSchema<ServiceTypeFormValues> = yup
  .object()
  .shape({
    status: yup.string<StatusChoiceProps>().required("Status is required"),
    code: yup
      .string()
      .max(10, "Code must be at most 10 characters")
      .required("Code is required"),
    description: yup.string().notRequired(),
  });

export const reasonCodeSchema: yup.ObjectSchema<ReasonCodeFormValues> = yup
  .object()
  .shape({
    status: yup.string<StatusChoiceProps>().required("Status is required"),
    code: yup
      .string()
      .max(10, "Code must be at most 10 characters")
      .required("Code is required"),
    codeType: yup.string<CodeTypeProps>().required("Code type is required"),
    description: yup.string().required("Description is required"),
  });

export const shipmentTypeSchema: yup.ObjectSchema<ShipmentTypeFormValues> = yup
  .object()
  .shape({
    status: yup.string<StatusChoiceProps>().required("Status is required"),
    code: yup
      .string()
      .max(10, "Name must be at most 100 characters")
      .required("Code is required"),
    description: yup.string().notRequired(),
  });

export function useShipmentForm({ user }: { user: User }) {
  const { shipmentControlData, isLoading: isShipmentControlLoading } =
    useShipmentControl();

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
          : yup.string(),
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
      rate: yup.string(),
      mileage: yup.number(),
      otherChargeAmount: yup.string(),
      freightChargeAmount: yup.string(),
      rateMethod: yup.string(),
      customer: yup.string().required("Customer is required."),
      pieces: yup.number(),
      weight: yup.string(),
      readyToBill: yup.boolean().required("Ready to bill is required."),
      trailer: yup.string(),
      trailerType: yup.string().required("Trailer type is required."),
      tractorType: yup.string(),
      commodity:
        shipmentControlData && shipmentControlData.enforceCommodity
          ? yup.string().required("Commodity is required.")
          : yup.string(),
      hazardousMaterial: yup.string(),
      temperatureMin: yup.string(),
      temperatureMax: yup.string(),
      bolNumber: yup.string().required("BOL number is required."),
      consigneeRefNumber: yup.string(),
      comment: yup
        .string()
        .max(100, "Comment must be less than 100 characters."),
      voidedComm: yup.string(),
      autoRate: yup.boolean().required("Auto rate is required."),
      formulaTemplate: yup.string(),
      enteredBy: yup.string().required("Entered by is required."),
      subTotal: yup.string(),
      serviceTye: yup.string(),
      entryMethod: yup
        .string<ShipmentEntryMethodChoices>()
        .required("Entry method is required."),
      copyAmount: yup.number().required("Copy amount is required."),
      stops: yup.array().of(stopSchema),
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

  return { shipmentForm, isShipmentControlLoading, shipmentControlData };
}
