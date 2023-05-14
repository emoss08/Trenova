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

import { Order } from "@/types/order";
import React from "react";
import { Card, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

type Props = {
  order: Order;
};

const OrderDetails: React.FC<Props> = ({ order }) => {
  return (
    <>
      <div className="mx-auto max-w-7x1 sm:px-6 lg:px-8">
        <Card className="w-[350px]">
          <CardHeader>
            <CardTitle>Order Details</CardTitle>
            <CardDescription>Currently Viewing Order: {order.pro_number}</CardDescription>
          </CardHeader>
        </Card>
        <p>
          Mileage: {order.mileage} <br />
          Comment: {order.comment} <br />
          Pro Number: {order.pro_number} <br />
          Appoint Window Start: {order.origin_appointment_window_start} <br />
          Billed: {order.billed} <br />
          Temperature Min: {order.temperature_min} <br />
          ID: {order.id} <br />
          Transferred to Billing: {order.transferred_to_billing} <br />
          Destination Location: {order.destination_location} <br />
          Sub Total Currency: {order.sub_total_currency} <br />
          Destination Address: {order.destination_address} <br />
          Movements: {order.movements} <br />
          Equipment Type: {order.equipment_type} <br />
          Origin Appointment Window End: {order.origin_appointment_window_end} <br />
          Ready to Bill: {order.ready_to_bill} <br />
          Order Comments: {order.order_comments} <br />
          Freight Charge Amount: {order.freight_charge_amount} <br />
          Rate Method: {order.rate_method} <br />
          Commodity: {order.commodity} <br />
          Sub Total: {order.sub_total} <br />
          BOL Number: {order.bol_number} <br />
          Additional Charges: {order.additional_charges} <br />
          Entered By: {order.entered_by} <br />
          Billing Transfer Date: {order.billing_transfer_date} <br />
          Weight: {order.weight} <br />
          Temperature Max: {order.temperature_max} <br />
          Voided Commission: {order.voided_comm} <br />
          Origin Location: {order.origin_location} <br />
          Origin Address: {order.origin_address} <br />
          Freight Charge Amount Currency: {order.freight_charge_amount_currency} <br />
          Other Charge Amount: {order.other_charge_amount} <br />
          Order Documentation: {order.order_documentation} <br />
          Hazmat: {order.hazmat} <br />
          Status: {order.status} <br />
          Other Charge Amount Currency: {order.other_charge_amount_currency} <br />
          Destination Appointment Window Start: {order.destination_appointment_window_start} <br />
          Destination Appointment Window End: {order.destination_appointment_window_end} <br />
          Customer: {order.customer} <br />
          Pieces: {order.pieces} <br />
          Auto Rate: {order.auto_rate} <br />
          Order Type: {order.order_type} <br />
          Consignee Ref Number: {order.consignee_ref_number} <br />
          Bill Date: {order.bill_date} <br />
          Rate: {order.rate} <br />
          Revenue Code: {order.revenue_code} <br />
        </p>
      </div>
    </>
  );
};

export default OrderDetails;