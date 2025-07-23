/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Package } from "lucide-react";
import { ShipmentSelection } from "./shipment-selection";

interface ConsolidationFormProps {
  isEdit?: boolean;
  shipmentErrors?: string | null;
}

export function ConsolidationForm({
  isEdit = false,
  shipmentErrors,
}: ConsolidationFormProps) {
  return (
    <div className="space-y-6">
      {!isEdit && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Package className="h-5 w-5" />
              Shipment Selection
            </CardTitle>
            <CardDescription>
              Select shipments to include in this consolidation. You can add
              more shipments later.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <ShipmentSelection shipmentErrors={shipmentErrors} />
          </CardContent>
        </Card>
      )}
    </div>
  );
}
