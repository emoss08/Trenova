import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Tabs, TabsContent, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { LandmarkIcon, ReceiptIcon, ShieldCheckIcon, TruckIcon, UsersIcon } from "lucide-react";
import { parseAsString, useQueryState } from "nuqs";
import { CarrierComplianceForm } from "./carrier-compliance-form";
import { CarrierContactsForm } from "./carrier-contacts-form";
import { CarrierForm } from "./carrier-form";
import { CarrierRemittanceForm } from "./carrier-remittance-form";
import { CarrierTaxForm } from "./carrier-tax-form";

export function CarrierTabbedForm() {
  const [activeTab, setActiveTab] = useQueryState("tab", parseAsString.withDefault("identity"));

  return (
    <Tabs
      value={activeTab}
      onValueChange={(value) => setActiveTab(value as string)}
      className="-m-4 flex flex-1 flex-col overflow-hidden"
    >
      <div className="border-border border-b px-4">
        <TabsList variant="underline">
          <TabsTab value="identity">
            <TruckIcon className="size-4" />
            Identity
          </TabsTab>
          <TabsTab value="compliance">
            <ShieldCheckIcon className="size-4" />
            Compliance & Insurance
          </TabsTab>
          <TabsTab value="tax">
            <ReceiptIcon className="size-4" />
            Tax
          </TabsTab>
          <TabsTab value="remittance">
            <LandmarkIcon className="size-4" />
            Remittance
          </TabsTab>
          <TabsTab value="contacts">
            <UsersIcon className="size-4" />
            Contacts
          </TabsTab>
        </TabsList>
      </div>
      <ScrollArea className="flex-1">
        <TabsContent value="identity" className="p-4">
          <CarrierForm />
        </TabsContent>
        <TabsContent value="compliance" className="p-4">
          <CarrierComplianceForm />
        </TabsContent>
        <TabsContent value="tax" className="p-4">
          <CarrierTaxForm />
        </TabsContent>
        <TabsContent value="remittance" className="p-4">
          <CarrierRemittanceForm />
        </TabsContent>
        <TabsContent value="contacts" className="p-4">
          <CarrierContactsForm />
        </TabsContent>
      </ScrollArea>
    </Tabs>
  );
}
