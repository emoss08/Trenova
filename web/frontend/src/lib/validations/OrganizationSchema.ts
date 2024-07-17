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

import {
  EnumDatabaseAction,
  EnumDeliveryMethod,
  type EmailProtocolChoiceProps,
  type RouteDistanceUnitProps,
  type RouteModelChoiceProps,
  type TimezoneChoices,
} from "@/lib/choices";
import { type StatusChoiceProps } from "@/types";
import type {
  EmailControlFormValues,
  EmailProfileFormValues,
  GoogleAPIFormValues,
  OrganizationFormValues,
  TableChangeAlertFormValues,
} from "@/types/organization";
import { ObjectSchema, boolean, mixed, number, object, string } from "yup";

export const organizationSchema: ObjectSchema<OrganizationFormValues> =
  object().shape({
    name: string().required("Name is required."),
    scacCode: string().required("SCAC Code is required."),
    dotNumber: string().required("DOT Number is required."),
    orgType: string().required("Organization Type is required."),
    timezone: string<TimezoneChoices>().required("Timezone is required."),
    logoUrl: string().notRequired(),
  });

export const tableChangeAlertSchema: ObjectSchema<TableChangeAlertFormValues> =
  object().shape({
    status: string<StatusChoiceProps>().required("Status is required."),
    name: string().required("Name is required."),
    databaseAction: mixed<EnumDatabaseAction>()
      .required("Database Action is required.")
      .oneOf(Object.values(EnumDatabaseAction)),
    topicName: string().required("Topic Name is required."),
    description: string(),
    emailProfile: string(),
    emailRecipients: string().test({
      name: "emailRecipients",
      message:
        "Email Recipients are only allowed if Delivery Method is Email. Please try again.",
      test: function (value) {
        const { deliveryMethod } = this.parent;
        if (deliveryMethod !== EnumDeliveryMethod.Email && value) {
          return false;
        }
        return true;
      },
    }),
    // conditionalLogic: object().nullable(),
    // Do not allow customSubject if the delivery method is not Email
    customSubject: string().test({
      name: "customSubject",
      message:
        "Custom Subject is only allowed if Delivery Method is Email. Please try again.",
      test: function (value) {
        const { deliveryMethod } = this.parent;
        if (deliveryMethod !== EnumDeliveryMethod.Email && value) {
          return false;
        }
        return true;
      },
    }),
    effectiveDate: string().nullable().notRequired(),
    deliveryMethod: mixed<EnumDeliveryMethod>()
      .required("Delivery Method is required.")
      .oneOf(Object.values(EnumDeliveryMethod)),
    expirationDate: string()
      .notRequired()
      .nullable()
      .when("effectiveDate", {
        is: (val: string) => val,
        then: (schema) =>
          schema.test(
            "is-after-effective-date",
            "Expiration Date must be after Effective Date. Please try again.",
            function (value) {
              const { effectiveDate } = this.parent;
              if (value && effectiveDate) {
                const effectiveDateObj = new Date(effectiveDate);
                const expirationDateObj = new Date(value);
                return expirationDateObj > effectiveDateObj;
              }
              return true;
            },
          ),
      }),
  });

export const emailControlSchema: ObjectSchema<EmailControlFormValues> =
  object().shape({
    billingEmailProfileId: string().notRequired(),
    rateExpirtationEmailProfileId: string().notRequired(),
  });

export const emailProfileSchema: ObjectSchema<EmailProfileFormValues> =
  object().shape({
    name: string().required("Name is required."),
    email: string().required("Email is required."),
    protocol: string<EmailProtocolChoiceProps>().notRequired(),
    host: string().notRequired(),
    port: number().notRequired(),
    username: string().notRequired(),
    password: string().notRequired(),
    isDefault: boolean().required("Default Profile is required."),
  });

export const googleAPISchema: ObjectSchema<GoogleAPIFormValues> =
  object().shape({
    apiKey: string().required("API Key is required."),
    mileageUnit: string<RouteDistanceUnitProps>().required(
      "Mileage Unit is required.",
    ),
    trafficModel: string<RouteModelChoiceProps>().required(
      "Traffic Model is required.",
    ),
    addCustomerLocation: boolean().required(
      "Add Customer Location is required.",
    ),
    addLocation: boolean().required("Add Location is required."),
    autoGeocode: boolean().required("Auto Geocode is required."),
  });
