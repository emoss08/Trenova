/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { ComponentLoader } from "@/components/ui/component-loader";

export function ShipmentDetailsSkeleton() {
  return (
    <div className="flex items-center justify-center h-full">
      <ComponentLoader message="Loading shipment details..." />
    </div>
  );
}
