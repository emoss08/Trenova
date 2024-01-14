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
  EmailProtocolChoiceProps,
  RouteDistanceUnitProps,
  RouteModelChoiceProps,
} from "@/lib/choices";
import {
  EmailControlFormValues,
  EmailProfileFormValues,
  GoogleAPIFormValues,
  OrganizationFormValues,
} from "@/types/organization";
import * as Yup from "yup";
import { ObjectSchema } from "yup";

export const organizationSchema: ObjectSchema<OrganizationFormValues> = Yup.object().shape({
  name: Yup.string().required("Name is required."),
  scacCode: Yup.string().required("SCAC Code is required."),
  dotNumber: Yup.number().required("DOT Number is required."),
  addressLine1: Yup.string().required("Address Line 1 is required."),
  addressLine2: Yup.string().notRequired(),
  city: Yup.string().required("City is required."),
  state: Yup.string().required("State is required."),
  zipCode: Yup.string().required("Zip Code is required."),
  phoneNumber: Yup.string().notRequired(),
  website: Yup.string().notRequired(),
  orgType: Yup.string().required("Organization Type is required."),
  timezone: Yup.string().required("Timezone is required."),
  language: Yup.string().required("Language is required."),
  currency: Yup.string().required("Currency is required."),
  dateFormat: Yup.string().required("Date Format is required."),
  timeFormat: Yup.string().required("Time Format is required."),
  logo: Yup.string().notRequired(),
  tokenExpirationDays: Yup.number().required(
    "Token Expiration Days is required.",
  ),
});



export const emailControlSchema: ObjectSchema<EmailControlFormValues> =
  Yup.object().shape({
    billingEmailProfile: Yup.string().notRequired(),
    rateExpirationEmailProfile: Yup.string().notRequired(),
  });

export const emailProfileSchema: ObjectSchema<EmailProfileFormValues> =
  Yup.object().shape({
    name: Yup.string().required("Name is required."),
    email: Yup.string().required("Email is required."),
    protocol: Yup.string<EmailProtocolChoiceProps>().notRequired(),
    host: Yup.string().notRequired(),
    port: Yup.number().notRequired(),
    username: Yup.string().notRequired(),
    password: Yup.string().notRequired(),
  });

export const googleAPISchema: ObjectSchema<GoogleAPIFormValues> =
  Yup.object().shape({
    apiKey: Yup.string().required("API Key is required."),
    mileageUnit: Yup.string<RouteDistanceUnitProps>().required(
      "Mileage Unit is required.",
    ),
    trafficModel: Yup.string<RouteModelChoiceProps>().required(
      "Traffic Model is required.",
    ),
    addCustomerLocation: Yup.boolean().required(
      "Add Customer Location is required.",
    ),
    addLocation: Yup.boolean().required("Add Location is required."),
    autoGeocode: Yup.boolean().required("Auto Geocode is required."),
  });
