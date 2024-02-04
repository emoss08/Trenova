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

import { CodeTypeProps, ShipmentStatusChoiceProps } from "@/lib/choices";
import { StatusChoiceProps } from "@/types";
import {
  ReasonCodeFormValues,
  ServiceTypeFormValues,
  ShipmentFormValues,
  ShipmentTypeFormValues,
} from "@/types/order";
import * as yup from "yup";

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

{
  /* if not origin location then origin address is required and vice versa */
}
export const shipmentSchema: yup.ObjectSchema<ShipmentFormValues> = yup
  .object()
  .shape({
    proNumber: yup.string().required("Pro number is required."),
    shipmentType: yup.string().required("Shipment type is required."),
    serviceType: yup.string().notRequired(),
    status: yup
      .string<ShipmentStatusChoiceProps>()
      .required("Status is required."),
    revenueCode: yup.string().notRequired(),
    originLocation: yup.string().test({
      name: "originLocation",
      test: function (value) {
        if (!value) {
          return this.parent.originAddress !== "";
        }
        return true;
      },
      message: "Origin location is required.",
    }),
    originAddress: yup.string().test({
      name: "originAddress",
      test: function (value) {
        if (!value) {
          return this.parent.originLocation !== "";
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
    destinationLocation: yup.string().test({
      name: "destinationLocation",
      test: function (value) {
        if (!value) {
          return this.parent.destinationAddress !== "";
        }
        return true;
      },
      message: "Destination location is required.",
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
    otherChargeAmount: yup
      .string()
      .required("Other charge amount is required."),
    freightChargeAmount: yup.string().notRequired(),
    rateMethod: yup.string().notRequired(),
    customer: yup.string().required("Customer is required."),
    pieces: yup.number().required("Pieces is required."),
    weight: yup.string().required("Weight is required."),
    readyToBill: yup.boolean().required("Ready to bill is required."),
    trailer: yup.string().notRequired(),
    trailerType: yup.string().required("Trailer type is required."),
    tractorType: yup.string().notRequired(),
    commodity: yup.string().notRequired(),
    hazardousMaterial: yup.string().notRequired(),
    temperatureMin: yup.string().notRequired(),
    temperatureMax: yup.string().notRequired(),
    bolNumber: yup.string().required("BOL number is required."),
    consigneeRefNumber: yup.string().notRequired(),
    comment: yup.string().notRequired(),
    voidedComm: yup.string().notRequired(),
    autoRate: yup.boolean().required("Auto rate is required."),
    formulaTemplate: yup.string().notRequired(),
    enteredBy: yup.string().required("Entered by is required."),
    subTotal: yup.string().required("Sub total is required."),
    serviceTye: yup.string().notRequired(),
    entryMethod: yup.string().required("Entry method is required."),
  });
