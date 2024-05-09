import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { Image } from "@unpic/react";
import { useNavigate } from "react-router-dom";

export default function NewShipmentCard() {
  const navigate = useNavigate();

  return (
    <Card className="relative col-span-4 lg:col-span-1">
      <CardContent className="p-0">
        <div className="flex h-[40vh] flex-col items-center justify-center">
          <p className="text-muted-foreground mr-1 text-2xl font-semibold">
            Quick Access to
          </p>
          <p className="text-muted-foreground mb-5 text-xl">Add New Shipment</p>
          <Image
            src="https://keenthemes.com/assets/media/vectors/volume/preview/yellow-purple/delivery.png"
            alt="Add New Shipment"
            layout="constrained"
            width={250}
            height={250}
          />
        </div>
      </CardContent>
      <CardFooter className="flex justify-center gap-x-2">
        <Button
          size="sm"
          onClick={() => navigate("/shipment-management/new-shipment")}
        >
          Add Shipment
        </Button>
        <Button size="sm" variant="outline">
          Learn More
        </Button>
      </CardFooter>
    </Card>
  );
}
