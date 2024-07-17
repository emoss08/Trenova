/**
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



import type { StatusChoiceProps } from "@/types";
import {
  CustomerContactFormValues,
  CustomerEmailProfileFormValues,
  CustomerFormValues,
  CustomerRuleProfileFormValues,
  DeliverySlotFormValues,
  EnumBillingCycleChoices,
  EnumDayOfWeekChoices,
  EnumEmailFormatChoices,
} from "@/types/customer";
import { array, boolean, mixed, object, string, type ObjectSchema } from "yup";

/** Customer Email Profile Schema */
export const customerEmailProfileSchema: ObjectSchema<CustomerEmailProfileFormValues> =
  object().shape({
    subject: string().max(100),
    emailProfileId: string().nullable(),
    emailRecipients: string().required("Email Recipients is required"),
    attachmentName: string().optional(),
    emailCcRecipients: string().optional(),
    emailFormat: mixed<EnumEmailFormatChoices>()
      .required("Email Format is required")
      .oneOf(Object.values(EnumEmailFormatChoices)),
  });

export const customerRuleProfileSchema: ObjectSchema<CustomerRuleProfileFormValues> =
  object().shape({
    docClassIds: array()
      .of(string().required())
      .min(1, "At Least one document class is required.")
      .required("Document Class is required"),
    billingCycle: mixed<EnumBillingCycleChoices>()
      .required("Billing Cycle is required")
      .oneOf(Object.values(EnumBillingCycleChoices)),
  });

const deliverySlotSchema: ObjectSchema<DeliverySlotFormValues> = object().shape(
  {
    dayOfWeek: mixed<EnumDayOfWeekChoices>()
      .required("Day of Week is required")
      .oneOf(Object.values(EnumDayOfWeekChoices)),
    startTime: string()
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
    endTime: string()
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
    locationId: string().required("Location is required"),
  },
);

const customerContactSchema: ObjectSchema<CustomerContactFormValues> =
  object().shape({
    name: string().required("Name is required"),
    email: string().when("isPayableContact", {
      is: true,
      then: (schema) => schema.required("Email is required"),
      otherwise: (schema) => schema.notRequired(),
    }),
    title: string().optional(),
    phoneNumber: string().optional(),
    isPayableContact: boolean().required(),
  });

/** Customer Schema */
export const customerSchema: ObjectSchema<CustomerFormValues> = object().shape({
  status: string<StatusChoiceProps>().required("Status is required"),
  code: string().optional(), // Code is generated on the server.
  name: string().required("Name is required"),
  addressLine1: string().required("Address Line 1 is required"),
  addressLine2: string().optional(),
  city: string().required("City is required"),
  stateId: string().required("State is required"),
  postalCode: string().required("Postal Code is required"),
  hasCustomerPortal: boolean(),
  autoMarkReadyToBill: boolean(),
  ruleProfile: customerRuleProfileSchema,
  emailProfile: customerEmailProfileSchema,
  deliverySlots: array().of(deliverySlotSchema),
  contacts: array().of(customerContactSchema),
});
