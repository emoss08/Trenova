import { ScrollArea } from "@/components/ui/scroll-area";
import { Tabs, TabsContent, TabsList, TabsTab } from "@/components/ui/tabs";
import { CreditCardIcon, MailIcon, UserIcon } from "lucide-react";
import { parseAsString, useQueryState } from "nuqs";
import { CustomerBillingProfileForm } from "./customer-billing-profile-form";
import { CustomerEmailProfileForm } from "./customer-email-profile-form";
import { CustomerForm } from "./customer-form";

export function CustomerTabbedForm() {
  const [activeTab, setActiveTab] = useQueryState("tab", parseAsString.withDefault("general"));

  return (
    <Tabs
      value={activeTab}
      onValueChange={(value) => setActiveTab(value as string)}
      className="-m-4 flex flex-1 flex-col overflow-hidden"
    >
      <div className="border-b border-border px-4">
        <TabsList variant="underline">
          <TabsTab value="general">
            <UserIcon className="size-4" />
            General
          </TabsTab>
          <TabsTab value="billing">
            <CreditCardIcon className="size-4" />
            Billing Profile
          </TabsTab>
          <TabsTab value="email">
            <MailIcon className="size-4" />
            Email Profile
          </TabsTab>
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
      </ScrollArea>
    </Tabs>
  );
}
