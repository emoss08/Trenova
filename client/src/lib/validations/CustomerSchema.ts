/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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

import { StatusChoiceProps } from "@/types";
import {
  CustomerContactFormValues,
  CustomerEmailProfileFormValues,
  CustomerFormValues,
  CustomerRuleProfileFormValues,
  DeliverySlotFormValues,
} from "@/types/customer";
import * as Yup from "yup";
import { ObjectSchema } from "yup";

/** Customer Email Profile Schema */
export const customerEmailProfileSchema: ObjectSchema<CustomerEmailProfileFormValues> =
  Yup.object().shape({
    subject: Yup.string().notRequired().max(100),
    comment: Yup.string().notRequired().max(100),
    fromAddress: Yup.string().notRequired(),
    blindCopy: Yup.string().notRequired(),
    readReceipt: Yup.boolean().required(),
    readReceiptTo: Yup.string().when("read_receipt", {
      is: true,
      then: (schema) => schema.required("Read Receipt To is required"),
      otherwise: (schema) => schema.notRequired(),
    }),
    attachmentName: Yup.string().notRequired(),
  });

export const customerRuleProfileSchema: ObjectSchema<CustomerRuleProfileFormValues> =
  Yup.object().shape({
    name: Yup.string().required("Name is required"),
    documentClass: Yup.array()
      .of(Yup.string().required())
      .min(1, "At Least one document class is required.")
      .required("Document Class is required"),
  });

const deliverySlotSchema: Yup.ObjectSchema<DeliverySlotFormValues> =
  Yup.object().shape({
    dayOfWeek: Yup.number().required("Day of Week is required"),
    startTime: Yup.string()
      .required("Start Time is required")
      .test(
        "is-before-end-time",
        "Start Time must be before End Time",
        function (value) {
          const { endTime } = this.parent;
          if (value && endTime) {
            const [startHours, startMinutes, startSeconds] = value
              .split(":")
              .map(Number);
            const [endHours, endMinutes, endSeconds] = endTime
              .split(":")
              .map(Number);
            const startDate = new Date(
              0,
              0,
              0,
              startHours,
              startMinutes,
              startSeconds,
            );
            const endDate = new Date(0, 0, 0, endHours, endMinutes, endSeconds);
            return startDate < endDate;
          }
          return true;
        },
      ),
    endTime: Yup.string()
      .required("End Time is required")
      .test(
        "is-after-start-time",
        "End Time must be after Start Time",
        function (value) {
          const { startTime } = this.parent;
          if (value && startTime) {
            const [startHours, startMinutes, startSeconds] = startTime
              .split(":")
              .map(Number);
            const [endHours, endMinutes, endSeconds] = value
              .split(":")
              .map(Number);
            const startDate = new Date(
              0,
              0,
              0,
              startHours,
              startMinutes,
              startSeconds,
            );
            const endDate = new Date(0, 0, 0, endHours, endMinutes, endSeconds);
            return endDate > startDate;
          }
          return true;
        },
      ),
    location: Yup.string().required("Location is required"),
  });
const customerContactSchema: ObjectSchema<CustomerContactFormValues> =
  Yup.object().shape({
    isActive: Yup.boolean().required(),
    name: Yup.string().required("Name is required"),
    email: Yup.string().notRequired(),
    title: Yup.string().notRequired(),
    phone: Yup.string().notRequired(),
    isPayableContact: Yup.boolean().required(),
  });

/** Customer Schema */
export const customerSchema: ObjectSchema<CustomerFormValues> =
  Yup.object().shape({
    status: Yup.string<StatusChoiceProps>().required("Status is required"),
    code: Yup.string().required("Code is required"),
    name: Yup.string().required("Name is required"),
    addressLine1: Yup.string().required("Address Line 1 is required"),
    addressLine2: Yup.string().notRequired(),
    city: Yup.string().required("City is required"),
    state: Yup.string().required("State is required"),
    zipCode: Yup.string().required("Zip Code is required"),
    hasCustomerPortal: Yup.boolean(),
    autoMarkReadyToBill: Yup.boolean(),
    advocate: Yup.string().notRequired(),
    deliverySlots: Yup.array().of(deliverySlotSchema).notRequired(),
    contacts: Yup.array().of(customerContactSchema).notRequired(),
    ruleProfile: customerRuleProfileSchema.notRequired(),
    emailProfile: customerEmailProfileSchema.notRequired(),
  });
