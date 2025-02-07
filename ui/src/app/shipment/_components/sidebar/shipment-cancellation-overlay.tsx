import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Icon } from "@/components/ui/icons";
import { Separator } from "@/components/ui/separator";
import { toDate } from "@/lib/date";
import {
  faChevronLeft,
  faCircleXmark,
} from "@fortawesome/pro-regular-svg-icons";

interface CanceledShipmentOverlayProps {
  children: React.ReactNode;
  onBack: () => void;
  canceledAt: number;
  canceledBy?: string;
  cancelReason?: string;
}

export function CanceledShipmentOverlay({
  children,
  onBack,
  canceledAt,
  canceledBy,
  cancelReason,
}: CanceledShipmentOverlayProps) {
  const canceledAtDate = toDate(canceledAt);

  const cancledAtDateString = canceledAtDate?.toLocaleDateString("en-US", {
    weekday: "long",
    year: "numeric",
    month: "long",
    day: "numeric",
  });

  const canceledAtTimeString = canceledAtDate?.toLocaleTimeString("en-US", {
    hour: "numeric",
    minute: "2-digit",
    hour12: true,
  });

  return (
    <div className="relative size-full rounded-md">
      <div className="opacity-30">{children}</div>
      <div className="absolute inset-0 flex items-center justify-center bg-background/50 backdrop-blur-[2px]">
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
              Shipment Canceled
            </h3>
          </div>
          <Separator className="my-4" />
          <div className="space-y-3 mb-6">
            <div className="space-y-1">
              <p className="text-sm font-medium text-muted-foreground">
                Canceled by
              </p>
              <p className="text-sm">{canceledBy ?? "-"}</p>
            </div>
            <div className="space-y-1">
              <p className="text-sm font-medium text-muted-foreground">
                Cancellation Time
              </p>
              <p className="text-sm">
                {cancledAtDateString ?? "-"}
                {" at "}
                {canceledAtTimeString ?? "-"}
              </p>
            </div>
            <div className="space-y-1">
              <p className="text-sm font-medium text-muted-foreground">
                Reason for Cancellation
              </p>
              <p className="text-sm text-pretty">{cancelReason ?? "-"}</p>
            </div>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={onBack}
            className="w-full gap-2"
          >
            <Icon icon={faChevronLeft} className="size-4" />
            Return to Shipments
          </Button>
        </Card>
      </div>
    </div>
  );
}
