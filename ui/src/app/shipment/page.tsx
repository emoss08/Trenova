import { FormSaveProvider } from "@/components/form";
import { MetaTags } from "@/components/meta-tags";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { SuspenseLoader } from "@/components/ui/component-loader";
import { Icon } from "@/components/ui/icons";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ShipmentFilterSchema } from "@/lib/schemas/shipment-filter-schema";
import {
  faBoxes,
  faClock,
  faExclamationTriangle,
  faTruck,
} from "@fortawesome/pro-regular-svg-icons";
import { FormProvider, useForm } from "react-hook-form";
import ShipmentTable from "./_components/shipment-table";

interface StatCardProps {
  title: string;
  value: string | number;
  icon: typeof faBoxes;
  trend?: number;
  color: string;
}

function StatCard({ title, value, icon, trend, color }: StatCardProps) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        <Icon icon={icon} className={`size-4 ${color}`} />
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{value}</div>
        {trend !== undefined && (
          <p className="text-xs text-muted-foreground">
            {trend > 0 ? "+" : ""}
            {trend}% from last month
          </p>
        )}
      </CardContent>
    </Card>
  );
}

interface ShipmentStats {
  totalActive: number;
  inTransit: number;
  delayed: number;
  pendingDelivery: number;
}

export function Shipment() {
  const form = useForm<ShipmentFilterSchema>({
    defaultValues: {
      search: undefined,
      status: undefined,
    },
  });

  // Mock stats - replace with actual data later
  const stats: ShipmentStats = {
    totalActive: 1234,
    inTransit: 856,
    delayed: 23,
    pendingDelivery: 156,
  };

  return (
    <FormSaveProvider>
      <div className="space-y-6 p-6">
        <MetaTags title="Shipments" description="Shipments" />

        {/* Header Section */}
        <div className="flex justify-between items-center">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Shipments</h1>
            <p className="text-muted-foreground">
              Manage and track all shipments in your system
            </p>
          </div>
        </div>

        {/* Stats Grid */}
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <StatCard
            title="Total Active Shipments"
            value={stats.totalActive}
            icon={faBoxes}
            trend={12}
            color="text-blue-600"
          />
          <StatCard
            title="In Transit"
            value={stats.inTransit}
            icon={faTruck}
            trend={8}
            color="text-green-600"
          />
          <StatCard
            title="Delayed"
            value={stats.delayed}
            icon={faExclamationTriangle}
            trend={-2}
            color="text-yellow-600"
          />
          <StatCard
            title="Pending Delivery"
            value={stats.pendingDelivery}
            icon={faClock}
            trend={5}
            color="text-purple-600"
          />
        </div>

        {/* Main Content Tabs */}
        <Tabs defaultValue="all" className="w-full">
          <TabsList>
            <TabsTrigger value="all">All Shipments</TabsTrigger>
            <TabsTrigger value="active">Active</TabsTrigger>
            <TabsTrigger value="completed">Completed</TabsTrigger>
            <TabsTrigger value="delayed">Delayed</TabsTrigger>
          </TabsList>

          <TabsContent value="all" className="mt-6">
            <SuspenseLoader>
              <FormProvider {...form}>
                <ShipmentTable />
              </FormProvider>
            </SuspenseLoader>
          </TabsContent>

          <TabsContent value="active">
            <Card>
              <CardContent className="p-6">
                <div className="flex items-center justify-center h-64 text-muted-foreground">
                  Active shipments will be displayed here
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="completed">
            <Card>
              <CardContent className="p-6">
                <div className="flex items-center justify-center h-64 text-muted-foreground">
                  Completed shipments will be displayed here
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="delayed">
            <Card>
              <CardContent className="p-6">
                <div className="flex items-center justify-center h-64 text-muted-foreground">
                  Delayed shipments will be displayed here
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </FormSaveProvider>
  );
}
