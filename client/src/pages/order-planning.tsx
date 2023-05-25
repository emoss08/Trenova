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

import axios from "@/lib/axiosConfig";
import React, { useEffect } from "react";
import { useParams } from "react-router-dom";
import OrderDetails from "@/components/order-management/OrderDetails";

const OrderPlanning: React.FC = () => {
  const { orderId } = useParams<{ orderId: string }>();
  const [order, setOrder] = React.useState<any>(null);

  useEffect((): void => {
    const fetchOrder = async (): Promise<void> => {
      try {
        const response = await axios.get(`/orders/${orderId}/`);
        setOrder(response.data);
      } catch {
        console.log("error");
      }
    };
    fetchOrder().then((): void => {
    });
  }, [orderId]);

  return (
    <div>{order ? <OrderDetails order={order} /> : <div>Loading...</div>}</div>
  );
};

export default OrderPlanning;
