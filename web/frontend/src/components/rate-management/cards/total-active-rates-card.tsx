import { Card, CardContent } from "@/components/ui/card";

export default function TotalActiveRatesCard() {
  return (
    <Card className="relative col-span-4 lg:col-span-1">
      <CardContent className="p-0">
        <div className="flex size-full flex-col items-center justify-center">
          Total Active Rates
        </div>
      </CardContent>
    </Card>
  );
}
