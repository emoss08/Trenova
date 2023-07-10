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
export type OrderControl = {
  id: string;
  organization: string;
  auto_rate_orders: boolean;
  calculate_distance: boolean;
  enforce_rev_code: boolean;
  enforce_voided_comm: boolean;
  generate_routes: boolean;
  enforce_commodity: boolean;
  auto_sequence_stops: boolean;
  auto_order_total: boolean;
  enforce_origin_destination: boolean;
  check_for_duplicate_bol: boolean;
  remove_orders: boolean;
};

export type OrderControlFormValues = {
  auto_rate_orders: boolean;
  calculate_distance: boolean;
  enforce_rev_code: boolean;
  enforce_voided_comm: boolean;
  generate_routes: boolean;
  enforce_commodity: boolean;
  auto_sequence_stops: boolean;
  auto_order_total: boolean;
  enforce_origin_destination: boolean;
  check_for_duplicate_bol: boolean;
  remove_orders: boolean;
};

export type Order = {
  mileage: number;
  comment: string;
  pro_number: string;
  origin_appointment_window_start: string;
  billed: boolean;
  temperature_min: null | number;
  id: string;
  transferred_to_billing: boolean;
  destination_location: string;
  sub_total_currency: string;
  destination_address: string;
  movements: string[];
  equipment_type: string;
  origin_appointment_window_end: string;
  ready_to_bill: boolean;
  order_comments: string[];
  freight_charge_amount: string;
  rate_method: string;
  commodity: null | string;
  sub_total: string;
  bol_number: string;
  additional_charges: any[];
  entered_by: string;
  billing_transfer_date: null | string;
  weight: string;
  temperature_max: null | number;
  voided_comm: string;
  origin_address: string;
  freight_charge_amount_currency: string;
  other_charge_amount: string;
  order_documentation: any[];
  hazmat: null | string;
  status: string;
  other_charge_amount_currency: string;
  destination_appointment_window_start: string;
  destination_appointment_window_end: string;
  customer: string;
  pieces: number;
  auto_rate: boolean;
  order_type: string;
  consignee_ref_number: string;
  bill_date: null | string;
  rate: null | string;
  revenue_code: null | string;
  origin_location: string;
};
