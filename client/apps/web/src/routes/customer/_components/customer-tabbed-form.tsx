import { useT } from "@trenova/shared/i18n/use-t";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Tabs, TabsContent, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { CreditCardIcon, MailIcon, RadarIcon, UserIcon } from "lucide-react";
import { parseAsString, useQueryState } from "nuqs";
import { CustomerBillingProfileForm } from "./customer-billing-profile-form";
import { CustomerBrokerIntelligence } from "./customer-broker-intelligence";
import { CustomerEmailProfileForm } from "./customer-email-profile-form";
import { CustomerForm } from "./customer-form";

export type CustomerTabbedFormProps = {
  customerId?: string;
};

export function CustomerTabbedForm({ customerId }: CustomerTabbedFormProps) {
  const t = useT();

  const [activeTab, setActiveTab] = useQueryState("tab", parseAsString.withDefault("general"));

  return (
    <Tabs
      value={activeTab}
      onValueChange={(value) => setActiveTab(value as string)}
      className="-m-4 flex flex-1 flex-col overflow-hidden"
    >
      <div className="border-border border-b px-4">
        <TabsList variant="underline">
          <TabsTab value="general">
            <UserIcon className="size-4" />
            {t("General")}
          </TabsTab>
          <TabsTab value="billing">
            <CreditCardIcon className="size-4" />
            {t("Billing profile")}
          </TabsTab>
          <TabsTab value="email">
            <MailIcon className="size-4" />
            {t("Email profile")}
          </TabsTab>
          {customerId ? (
            <TabsTab value="broker-vetting">
              <RadarIcon className="size-4" />
              {t("Broker vetting")}
            </TabsTab>
          ) : null}
        </TabsList>
      </div>
      <ScrollArea className="flex-1">
        <TabsContent value="general" className="p-4">
          <CustomerForm />
        </TabsContent>
        <TabsContent value="billing" className="p-4">
          <div className="space-y-6">
            <CustomerBillingProfileForm />
          </div>
        </TabsContent>
        <TabsContent value="email" className="p-4">
          <CustomerEmailProfileForm />
        </TabsContent>
        {customerId ? (
          <TabsContent value="broker-vetting" className="p-4">
            <CustomerBrokerIntelligence customerId={customerId} />
          </TabsContent>
        ) : null}
      </ScrollArea>
    </Tabs>
  );
}
