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

/** Customer Order Metric Type */
type TotalOrderMetricsType = {
    total_orders: number;
    last_month_diff: number;
    month_before_last_diff: number;
};

type TotalRevenueMetricsType = {
    total_revenue: number;
    last_month_diff: number;
    month_before_last_diff: number;
};

type PerformanceMetricType = {
    this_month_on_time_percentage: number;
    last_month_on_time_percentage: number;
    on_time_diff: number;
    this_month_early_percentage: number;
    last_month_early_percentage: number;
    early_diff: number;
    this_month_late_percentage: number;
    last_month_late_percentage: number;
    late_diff: number;
};

type TotalMileageMetricsType = {
    this_month_miles: number;
    last_month_miles: number;
    mileage_diff: number;
};

/** Customer Shipment Metric Type */
type CustomerShipmentMetrics = {
    last_bill_date?: string | null;
    last_shipment_date?: string | null;
};

/** Customer Type */
export type Customer = {
    id: string;
    organization: string;
    status: string;
    code: string;
    name: string;
    address_line_1: string;
    address_line_2: string;
    city: string;
    zip_code: string;
    has_customer_portal: boolean;
    auto_mark_ready_to_bill: boolean;
    created: string;
    modified: string;
    customer_shipment_metrics: CustomerShipmentMetrics;
    total_order_metrics: TotalOrderMetricsType;
    total_revenue_metrics: TotalRevenueMetricsType;
    on_time_performance: PerformanceMetricType;
    total_mileage_metrics: TotalMileageMetricsType;
    credit_balance: number;
};

/** Customer Form Values Type */
export type CustomerFormValues = Omit<
    Customer,
    "id" | "organization" | "created" | "modified"
>;
