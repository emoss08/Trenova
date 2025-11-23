import { TourProvider } from "@/components/tour/tour-provider";
import { Icon } from "@/components/ui/icons";
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
} from "@/components/ui/sidebar";
import { checkSectionErrors } from "@/lib/form";
import { type CustomerSchema } from "@/lib/schemas/customer-schema";
import { cn } from "@/lib/utils";
import {
  faCreditCard,
  faEnvelope,
  faUser,
} from "@fortawesome/pro-regular-svg-icons";
import { useState } from "react";
import { type FieldValues, type Path, useFormContext } from "react-hook-form";
import { CustomerBillingProfile } from "./customer-billing-profile";
import { CustomerEmailProfile } from "./customer-email-profile";
import { CustomerGeneralInformation } from "./customer-general-information";

function createNavigationItems<T extends FieldValues>({
  isEdit = false,
}: {
  isEdit?: boolean;
}) {
  return [
    {
      id: "general",
      name: "General Information",
      description: "Essential customer identification details.",
      icon: <Icon icon={faUser} />,
      component: <CustomerGeneralInformation isEdit={isEdit} />,
      validateSection: (errors: Partial<T>) =>
        checkSectionErrors(errors, [
          "status",
          "code",
          "name",
          "description",
          "addressLine1",
          "addressLine2",
          "city",
          "stateId",
          "postalCode",
          "externalId",
        ] as Path<T>[]),
    },
    {
      id: "billing-profile",
      name: "Billing Profile",
      description: "Configure billing settings for the customer.",
      icon: <Icon icon={faCreditCard} />,
      component: <CustomerBillingProfile />,
      validateSection: (errors: Partial<T>) =>
        checkSectionErrors(errors, [
          "billingProfile.billingCycleType",
          "billingProfile.documentTypes",
          "billingProfile.hasOverrides",
          "billingProfile.enforceCustomerBillingReq",
          "billingProfile.validateCustomerRates",
          "billingProfile.paymentTerm",
          "billingProfile.autoTransfer",
          "billingProfile.autoMarkReadyToBill",
          "billingProfile.autoBill",
          "billingProfile.specialInstructions",
          "billingProfile.revenueAccountId",
          "billingProfile.arAccountId",
          "billingProfile.allowInvoiceConsolidation",
          "billingProfile.consolidationPeriodDays",
        ] as Path<T>[]),
    },
    {
      id: "email-profile",
      name: "Email Profile",
      description: "Configure email settings for the customer.",
      icon: <Icon icon={faEnvelope} />,
      component: <CustomerEmailProfile />,
      validateSection: (errors: Partial<T>) =>
        checkSectionErrors(errors, [
          "emailProfile.subject",
          "emailProfile.comment",
          "emailProfile.fromEmail",
          "emailProfile.blindCopy",
          "emailProfile.readReceipt",
          "emailProfile.attachmentName",
        ] as Path<T>[]),
    },
  ];
}

export function CustomerForm({ isEdit = false }: { isEdit?: boolean }) {
  const {
    formState: { errors },
  } = useFormContext<CustomerSchema>();

  const [activeSection, setActiveSection] = useState("general");
  const navigationItems = createNavigationItems<CustomerSchema>({ isEdit });
  const activeComponent = navigationItems.find(
    (item) => item.id === activeSection,
  )?.component;

  return (
    <TourProvider>
      <CustomerFormOuter>
        <SidebarProvider className="h-auto min-h-[750px] w-56 shrink-0 items-start">
          <Sidebar
            collapsible="none"
            className="hidden w-56 rounded-tl-lg border-r border-input/50 md:flex"
          >
            <SidebarContent>
              <SidebarGroup>
                <SidebarGroupContent>
                  <SidebarMenu>
                    {navigationItems.map((item) => {
                      const hasError = item.validateSection(errors as any);

                      return (
                        <SidebarMenuItem key={item.id}>
                          <SidebarMenuButton
                            asChild
                            isActive={activeSection === item.id}
                            onClick={() => setActiveSection(item.id)}
                            className={cn(
                              "hover:bg-transparent text-muted-foreground size-full gap-0.5",
                              hasError && "hover:text-red-500 text-red-600",
                            )}
                          >
                            <div className="flex flex-col items-start">
                              <div className="flex items-center gap-2">
                                {item.icon}
                                {item.name}
                              </div>
                              <div className="w-[190px] truncate text-2xs text-muted-foreground">
                                {item.description}
                              </div>
                            </div>
                          </SidebarMenuButton>
                        </SidebarMenuItem>
                      );
                    })}
                  </SidebarMenu>
                </SidebarGroupContent>
              </SidebarGroup>
            </SidebarContent>
          </Sidebar>
        </SidebarProvider>
        <CustomerFormContent>{activeComponent}</CustomerFormContent>
      </CustomerFormOuter>
    </TourProvider>
  );
}

function CustomerFormOuter({ children }: { children: React.ReactNode }) {
  return <div className="flex size-full flex-1">{children}</div>;
}

function CustomerFormContent({ children }: { children: React.ReactNode }) {
  return <main className="flex size-full">{children}</main>;
}
