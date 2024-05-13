import type { StatusChoiceProps } from "@/types";
import {
  BillingCycleChoices,
  CustomerContactFormValues,
  CustomerEmailProfileFormValues,
  CustomerFormValues,
  CustomerRuleProfileFormValues,
  DeliverySlotFormValues,
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
    documentClass: array()
      .of(string().required())
      .min(1, "At Least one document class is required.")
      .required("Document Class is required"),
    billingCycle: mixed<BillingCycleChoices>()
      .required("Billing Cycle is required")
      .oneOf(Object.values(BillingCycleChoices)),
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
    status: string<StatusChoiceProps>().required(),
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
