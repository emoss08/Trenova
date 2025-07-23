/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Icon } from "@/components/ui/icons";
import {
  faChevronLeft,
  faCircleXmark,
} from "@fortawesome/pro-regular-svg-icons";

export function ShipmentNotFoundOverlay() {
  return (
    <div className="relative size-full rounded-md">
      <div className="flex absolute inset-0 items-center justify-center bg-background/50 backdrop-blur-[2px]">
        <Card className="w-[400px] p-6 shadow-lg border">
          <div className="flex flex-col items-center mb-4">
            <div className="relative mb-2">
              <div className="absolute -inset-1 rounded-full bg-destructive/20 motion-safe:animate-pulse" />
              <Icon
                icon={faCircleXmark}
                className="size-12 text-destructive relative"
              />
            </div>
            <h3 className="text-xl font-semibold text-destructive">
              Shipment Not Found
            </h3>
          </div>
          <Button variant="outline" size="sm" className="w-full gap-2">
            <Icon icon={faChevronLeft} className="size-4" />
            Return to Shipments
          </Button>
        </Card>
      </div>
    </div>
  );
}
