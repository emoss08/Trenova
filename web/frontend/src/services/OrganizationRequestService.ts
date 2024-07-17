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

import axios from "@/lib/axiosConfig";
import { BillingControl } from "@/types/billing";
import { DispatchControl } from "@/types/dispatch";
import { InvoiceControl } from "@/types/invoicing";
import {
  Depot,
  EmailControl,
  EmailProfile,
  GoogleAPI,
  Organization,
  OrganizationFeatureFlag,
  Topic,
} from "@/types/organization";
import { RouteControl } from "@/types/route";
import { ShipmentControl } from "@/types/shipment";

/**
 * Featches the details of the user currently assigned organization.
 * @returns A promise that resolves to the organization's details.
 * @note This only returns parts of the organization
 */
export async function getUserOrganizationDetails(): Promise<Organization> {
  const response = await axios.get("/organizations/me/");
  return response.data;
}

/**
 * Featches the details of the user currently assigned organization.
 * @returns A promise that resolves to the organization's details.
 */
export async function getOrganizationDetails(): Promise<Organization> {
  const response = await axios.get("/organizations/details/");
  return response.data;
}

/**
 * Fetches billing control from the server.
 * @returns A promise that resolves to an array of billing control.
 * @note This should only return one result.
 */
export async function getBillingControl(): Promise<BillingControl> {
  const response = await axios.get("/billing-control/");
  return response.data;
}

/**
 * Fetches dispatch control from the server.
 * @returns A promise that resolves to an array of dispatch control.
 * @note This should only return one result.
 */
export async function getDispatchControl(): Promise<DispatchControl> {
  const response = await axios.get("/dispatch-control/");
  return response.data;
}

/**
 * Fetches invoice control from the server.
 * @returns A promise that resolves to an array of invoice control.
 * @note This should only return one result.
 */
export async function getInvoiceControl(): Promise<InvoiceControl> {
  const response = await axios.get("/invoice-control/");
  return response.data;
}

/**
 * Fetches order control from the server.
 * @returns A promise that resolves to an array of order control.
 * @note This should only return one result.
 */
export async function getShipmentControl(): Promise<ShipmentControl> {
  const response = await axios.get("/shipment-control/");
  return response.data;
}

/**
 * Fetches email profiles from the server.
 * @returns A promise that resolves to an array of email profiles.
 * @note This should only return one result.
 */
export async function getEmailProfiles(): Promise<EmailProfile[]> {
  const response = await axios.get("/email-profiles/");
  return response.data.results;
}

/**
 * Fetches email control from the server.
 * @returns A promise that resolves to an array of email control.
 * @note This should only return one result.
 */
export async function getEmailControl(): Promise<EmailControl> {
  const response = await axios.get("/email-control/");
  return response.data;
}

/**
 * Fetches route control from the server.
 * @returns A promise that resolves to an array of route control.
 * @note This should only return one result.
 */
export async function getRouteControl(): Promise<RouteControl> {
  const response = await axios.get("/route-control/");
  return response.data;
}

/**
 * Fetches depots from the server.
 * @returns A promise that resolves to an array of depots.
 */
export async function getDepots(): Promise<Depot[]> {
  const response = await axios.get("/depots/");
  return response.data.results;
}

/**
 * Fetches feature flags for the organization from the server.
 * @returns A promise that resolves to an array of feature flags.
 */
export async function getFeatureFlags(): Promise<OrganizationFeatureFlag[]> {
  const response = await axios.get("/feature-flags/");
  return response.data.results;
}

/**
 * Fetches the Google api information for the organization from the server.
 * @returns A promise that resolves to an array of Google api information.
 */
export async function getGoogleApiInformation(): Promise<GoogleAPI> {
  const response = await axios.get("/google-api/");
  return response.data;
}

/**
 * Fetches topic values from the server.
 * @returns A promise that resolves to an array of Table Names.
 */
export async function getTableNames(): Promise<
  { value: string; label: string }[]
> {
  const response = await axios.get("/table-change-alerts/table-names/");
  return response.data.results;
}

/**
 * Fetches topic values from the server.
 * @returns A promise that resolves to an array of Table Names.
 */
export async function getTopicNames(): Promise<Topic[]> {
  const response = await axios.get("/table-change-alerts/topics/");
  return response.data.results;
}

/**
 * Posts a user profile picture to the server.
 * @param profilePicture Profile picture to be uploaded
 * @returns A promise that resolves to the user's details.
 */
export async function postOrganizationLogo(logo: File): Promise<Organization> {
  const formData = new FormData();
  formData.append("logo", logo);
  const response = await axios.post("organizations/upload-logo", formData, {
    headers: {
      "Content-Type": "multipart/form-data",
    },
  });

  return response.data;
}

export async function clearOrganizationLogo(): Promise<void> {
  await axios.post("organizations/clear-logo");
}
