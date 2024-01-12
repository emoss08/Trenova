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

import axios from "@/lib/axiosConfig";
import { Shipment } from "@/types/order";

/**
 * Fetches the shipments from the server.
 * @returns A promise that resolves to a FeasibilityToolControl object.
 */
export async function getShipments(
  searchQuery?: string,
  statusFilter?: string,
): Promise<Shipment[]> {
  const response = await axios.get("/shipments/", {
    params: {
      search: searchQuery,
      status: statusFilter,
    },
  });
  return response.data.results;
}

type ShipmentCount = {
  status: string;
  count: number;
};

type ShipmentsByStatusResponse = {
  results: ShipmentCount[];
  totalCount: number;
};

export async function getShipmentCountByStatus(
  searchQuery?: string,
): Promise<ShipmentsByStatusResponse> {
  const response = await axios.get("/shipments/get_shipment_count_by_status/", {
    params: {
      search: searchQuery,
    },
  });
  return response.data;
}
