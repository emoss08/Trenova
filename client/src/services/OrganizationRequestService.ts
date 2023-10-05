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

import axios from "@/lib/AxiosConfig";
import { JobTitle } from "@/types/accounts";
import {
  Department,
  EmailControl,
  EmailProfile,
  Organization,
} from "@/types/organization";
import { BillingControl } from "@/types/billing";
import { DispatchControl } from "@/types/dispatch";
import { InvoiceControl } from "@/types/invoicing";
import { ShipmentControl } from "@/types/order";
import { RouteControl } from "@/types/route";

/**
 * Fetches organizations from the server.
 * @returns A promise that resolves to an array of organizations.
 */
export async function getOrganizations(): Promise<Organization[]> {
  const response = await axios.get("/organizations/");
  return response.data.results;
}

/**
 * Fetches the details of the organization with the specified ID.
 * @param id - The ID of the organization to fetch details for.
 * @returns A promise that resolves to the organization's details.
 */
export async function getOrganizationDetails(
  id: string,
): Promise<Organization> {
  const response = await axios.get(`/organizations/${id}/`);
  return response.data;
}

/**
 * Fetches departments from the server.
 * @returns A promise that resolves to an array of departments.
 */
export async function getDepartments(): Promise<Department[]> {
  const response = await axios.get("/departments/");
  return response.data.results;
}

/**
 * Fetches the details of the department with the specified ID.
 * @param id - The ID of the department to fetch details for.
 * @returns A promise that resolves to the department's details.
 */
export async function getDepartmentDetails(id: string): Promise<Department> {
  const response = await axios.get(`/departments/${id}/`);
  return response.data;
}

/**
 * Fetches job titles from the server.
 * @returns A promise that resolves to an array of job titles.
 */
export async function getJobTitles(): Promise<JobTitle[]> {
  const response = await axios.get("/job_titles/");
  return response.data.results;
}

/**
 * Fetches the details of the job title with the specified ID.
 * @param id - The ID of the job title to fetch details for.
 * @returns A promise that resolves to the job title's details.
 */
export async function getJobTitleDetails(id: string): Promise<JobTitle> {
  const response = await axios.get(`/job_titles/${id}/`);
  return response.data;
}

/**
 * Fetches billing control from the server.
 * @returns A promise that resolves to an array of billing control.
 * @note This should only return one result.
 */
export async function getBillingControl(): Promise<BillingControl[]> {
  const response = await axios.get("/billing_control/");
  return response.data.results;
}

/**
 * Fetches dispatch control from the server.
 * @returns A promise that resolves to an array of dispatch control.
 * @note This should only return one result.
 */
export async function getDispatchControl(): Promise<DispatchControl[]> {
  const response = await axios.get("/dispatch_control/");
  return response.data.results;
}

/**
 * Fetches invoice control from the server.
 * @returns A promise that resolves to an array of invoice control.
 * @note This should only return one result.
 */
export async function getInvoiceControl(): Promise<InvoiceControl[]> {
  const response = await axios.get("/invoice_control/");
  return response.data.results;
}

/**
 * Fetches order control from the server.
 * @returns A promise that resolves to an array of order control.
 * @note This should only return one result.
 */
export async function getShipmentControl(): Promise<
  ReadonlyArray<ShipmentControl>
> {
  const response = await axios.get("/shipment_control/");
  return response.data.results;
}

/**
 * Fetches email profiles from the server.
 * @returns A promise that resolves to an array of email profiles.
 * @note This should only return one result.
 */
export async function getEmailProfiles(): Promise<EmailProfile[]> {
  const response = await axios.get("/email_profiles/");
  return response.data.results;
}

/**
 * Fetches email control from the server.
 * @returns A promise that resolves to an array of email control.
 * @note This should only return one result.
 */
export async function getEmailControl(): Promise<EmailControl[]> {
  const response = await axios.get("/email_control/");
  return response.data.results;
}

/**
 * Fetches route control from the server.
 * @returns A promise that resolves to an array of route control.
 * @note This should only return one result.
 */
export async function getRouteControl(): Promise<RouteControl[]> {
  const response = await axios.get("/route_control/");
  return response.data.results;
}
