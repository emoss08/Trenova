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
import {
  Customer,
  CustomerEmailProfile,
  CustomerRuleProfile,
} from "@/types/customer";

export async function getCustomers(): Promise<ReadonlyArray<Customer>> {
  const response = await axios.get("customers/");
  return response.data.results;
}

/**
 * Fetches the details of the customer with the specified ID.
 * @param id The ID of the customer to fetch details for.
 * @returns A promise that resolves to a customer details.
 */
export async function getCustomerDetails(id: string): Promise<Customer> {
  const response = await axios.get(`customers/${id}/`);
  return response.data;
}

/**
 * Fetaches the details of the customer including metrics with the specified ID.
 * @param id The ID of the customer to fetch details for.
 * @returns A promise that resolves to a customer details.
 */
export async function getCustomerDetailsWithMetrics(
  id: string,
): Promise<Customer> {
  const response = await axios.get(`customers/${id}/?expand_metrics=True`);
  return response.data;
}

/**
 * Fetches the customer rule profiles from the server.
 * @returns A promise that resolves to a list of customer rule profiles.
 */
export async function getCustomerRuleProfiles(): Promise<
  CustomerRuleProfile[]
> {
  const response = await axios.get("customer_rule_profiles/");
  return response.data.results;
}

/**
 * Fetches the customer email profiles from the server.
 * @returns A promise that resolves to a list of customer email profiles.
 */
export async function getCustomerEmailProfile(
  id: string,
): Promise<CustomerEmailProfile> {
  const response = await axios.get(
    `customer_email_profiles/get_by_customer_id/?customer_id=${id}`,
  );
  return response.data;
}

export async function getCustomerRuleProfile(
  id: string,
): Promise<CustomerRuleProfile> {
  const response = await axios.get(
    `customer_rule_profiles/get_by_customer_id/?customer_id=${id}`,
  );
  return response.data;
}
