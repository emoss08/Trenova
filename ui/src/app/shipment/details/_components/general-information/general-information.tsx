import { AutocompleteField } from "@/components/fields/autocomplete";
import { DoubleClickInput } from "@/components/fields/input-field";
import { ColorOptionValue } from "@/components/fields/select-components";
import { PlainShipmentStatusBadge } from "@/components/status-badge";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Icon } from "@/components/ui/icons";
import { ScrollArea, ScrollBar } from "@/components/ui/scroll-area";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { formatDate, toDate } from "@/lib/date";
import { ServiceTypeSchema } from "@/lib/schemas/service-type-schema";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { ShipmentTypeSchema } from "@/lib/schemas/shipment-type-schema";
import { calculateShipmentMileage } from "@/lib/shipment/utils";
import { cn } from "@/lib/utils";
import { Shipment } from "@/types/shipment";
import {
  faChartLine,
  faFolder,
  faHexagonNodes,
  faRoad,
  faSquareRing,
} from "@fortawesome/pro-regular-svg-icons";
import { Path, useFormContext } from "react-hook-form";
import { MoveInformation } from "./move-information";

interface DetailItemProps {
  label: string;

  fieldName?: Path<ShipmentSchema>;
  value?: React.ReactNode;
  className?: string;
}

function DetailItem({ label, fieldName, value, className }: DetailItemProps) {
  const { control } = useFormContext<ShipmentSchema>();

  return (
    <div className={cn("space-y-1", className)}>
      <dt className="text-sm font-medium text-muted-foreground uppercase">
        {label}
      </dt>
      <dd className="text-sm text-foreground max-h-4">
        {fieldName ? (
          <DoubleClickInput
            control={control}
            name={fieldName}
            className="max-w-[100px]"
            displayClassName="text-foreground"
          />
        ) : (
          value
        )}
      </dd>
    </div>
  );
}

export default function GeneralInformation() {
  return (
    <div>
      <ShipmentDetails />
    </div>
  );
}

function ShipmentDetails() {
  const { getValues } = useFormContext<ShipmentSchema>();
  const { proNumber } = getValues();

  return (
    <div className="flex flex-col gap-4">
      <h3 className="text-4xl font-semibold">{proNumber ?? "-"}</h3>
      <div className="flex flex-col gap-4">
        <ShipmentStats />
        <ShipmentServiceDetails />
        <ShipmentTabs />
        {/* <TrailerCapacity /> */}
      </div>
    </div>
  );
}
function ShipmentStats() {
  const { getValues } = useFormContext<Shipment>();
  const { createdAt, status } = getValues();

  const createdAtDate = toDate(createdAt);
  const formatedCreatedAt = createdAtDate ? formatDate(createdAtDate) : "-";

  const mileage = calculateShipmentMileage(getValues());

  return (
    <dl className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
      <div className="space-y-1">
        <dt className="text-sm font-medium text-muted-foreground uppercase">
          Status
        </dt>
        <dd className="text-sm text-foreground max-h-4">
          <PlainShipmentStatusBadge status={status} />
        </dd>
      </div>
      <DetailItem fieldName="bol" label="BOL Number" />
      <DetailItem label="Total Distance" value={`${mileage} mi.`} />
      <DetailItem label="Created At" value={formatedCreatedAt} />
    </dl>
  );
}

function ShipmentTabs() {
  return (
    <Tabs defaultValue="tab-1">
      <ScrollArea>
        <TabsList className="w-full mb-3 h-auto gap-2 rounded-none border-b border-border bg-transparent px-0 py-1 text-foreground justify-start">
          <TabsTrigger
            value="tab-1"
            className="relative after:absolute after:inset-x-0 after:bottom-0 after:-mb-1.5 after:h-0.5 hover:bg-accent hover:text-foreground data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:after:bg-primary data-[state=active]:hover:bg-accent"
          >
            <Icon
              icon={faRoad}
              className="-ms-0.5 me-1.5 opacity-60 size-4"
              aria-hidden="true"
            />
            Moves
          </TabsTrigger>
          <TabsTrigger
            value="tab-2"
            className="relative after:absolute after:inset-x-0 after:bottom-0 after:-mb-1.5 after:h-0.5 hover:bg-accent hover:text-foreground data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:after:bg-primary data-[state=active]:hover:bg-accent"
          >
            <Icon
              icon={faFolder}
              className="-ms-0.5 me-1.5 opacity-60 size-4"
              aria-hidden="true"
            />
            Documents
            <Badge
              className="ms-1.5 min-w-5 bg-primary/15"
              variant="secondary"
              withDot={false}
            >
              3
            </Badge>
          </TabsTrigger>
          <TabsTrigger
            value="tab-3"
            className="relative after:absolute after:inset-x-0 after:bottom-0 after:-mb-1.5 after:h-0.5 hover:bg-accent hover:text-foreground data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:after:bg-primary data-[state=active]:hover:bg-accent"
          >
            <Icon
              icon={faHexagonNodes}
              className="-ms-0.5 me-1.5 opacity-60 size-4"
              aria-hidden="true"
            />
            EDI
            <Badge variant="purple" className="ms-1.5">
              New
            </Badge>
          </TabsTrigger>
          <TabsTrigger
            value="tab-4"
            className="relative after:absolute after:inset-x-0 after:bottom-0 after:-mb-1.5 after:h-0.5 hover:bg-accent hover:text-foreground data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:after:bg-primary data-[state=active]:hover:bg-accent"
          >
            <Icon
              icon={faChartLine}
              className="-ms-0.5 me-1.5 opacity-60 size-4"
              aria-hidden="true"
            />
            History
          </TabsTrigger>
        </TabsList>
        <ScrollBar orientation="horizontal" />
      </ScrollArea>
      <TabsContent value="tab-1">
        <p className="pt-1">
          <MoveInformation />
        </p>
      </TabsContent>
      <TabsContent value="tab-2">
        <p className="pt-1">Content for Tab 2</p>
      </TabsContent>
      <TabsContent value="tab-3">
        <p className="pt-1 text-center text-xs text-muted-foreground">
          Content for Tab 3
        </p>
      </TabsContent>
      <TabsContent value="tab-4">
        <p className="pt-1 text-center text-xs text-muted-foreground">
          Content for Tab 4
        </p>
      </TabsContent>

      <TabsContent value="tab-5">
        <p className="pt-1 text-center text-xs text-muted-foreground">
          Content for Tab 5
        </p>
      </TabsContent>
    </Tabs>
  );
}

function ShipmentServiceDetails() {
  const { control } = useFormContext<ShipmentSchema>();
  return (
    <Card className="w-full mt-4">
      <CardHeader className="p-4">
        <CardTitle className="flex items-center space-x-2">
          <div className="border-border flex size-10 items-center justify-center rounded-lg border">
            <Icon icon={faSquareRing} className="size-5" />
          </div>
          <div className="flex flex-col">
            <h3 className="text-lg font-semibold">Service Information</h3>
            <p className="text-muted-foreground text-xs font-normal">
              Information about the service type and shipment type for the
              shipment.
            </p>
          </div>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <FormGroup cols={2}>
          <FormControl>
            <AutocompleteField<ShipmentTypeSchema, ShipmentSchema>
              name="shipmentTypeId"
              control={control}
              link="/shipment-types/"
              label="Shipment Type"
              rules={{ required: true }}
              placeholder="Select Shipment Type"
              description="Select the shipment type for the shipment."
              getOptionValue={(option) => option.id || ""}
              getDisplayValue={(option) => (
                <ColorOptionValue color={option.color} value={option.code} />
              )}
              renderOption={(option) => (
                <ColorOptionValue color={option.color} value={option.code} />
              )}
            />
          </FormControl>
          <FormControl>
            <AutocompleteField<ServiceTypeSchema, ShipmentSchema>
              name="serviceTypeId"
              control={control}
              link="/service-types/"
              label="Service Type"
              rules={{ required: true }}
              placeholder="Select Service Type"
              description="Select the service type for the shipment."
              getOptionValue={(option) => option.id || ""}
              getDisplayValue={(option) => (
                <ColorOptionValue color={option.color} value={option.code} />
              )}
              renderOption={(option) => (
                <ColorOptionValue color={option.color} value={option.code} />
              )}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}
