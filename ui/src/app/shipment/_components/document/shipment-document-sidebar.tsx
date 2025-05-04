import { CategoryCard } from "@/components/document-workflow/document-workflow-category-card";
import {
  CategoryListSkeleton,
  NoDocumentRequirements,
} from "@/components/document-workflow/document-workflow-skeleton";
import { ScrollArea } from "@/components/ui/scroll-area";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import type { CustomerDocumentRequirement } from "@/types/customer";
import { BillingReadinessBadge } from "./_components/billing-readiness-badge";

type DocumentCategory = {
  id: string;
  name: string;
  description: string;
  color: string;
  requirements: CustomerDocumentRequirement[];
  complete: boolean;
  documentsCount: number;
};

export function ShipmentDocumentSidebar({
  documentCategories,
  isLoadingRequirements,
  activeCategory,
  setActiveCategory,
  customerId,
}: {
  documentCategories: DocumentCategory[];
  isLoadingRequirements: boolean;
  activeCategory: string | null;
  setActiveCategory: (category: string) => void;
  customerId: ShipmentSchema["customerId"];
}) {
  return (
    <div className="w-1/4 bg-muted border-r border-border">
      <ShipmentDocumentSidebarHeader documentCategories={documentCategories} />
      <ScrollArea className="flex h-[calc(100%-140px)]">
        <div className="p-2">
          {isLoadingRequirements ? (
            <CategoryListSkeleton />
          ) : documentCategories.length > 0 ? (
            documentCategories.map((category) => (
              <CategoryCard
                key={category.id}
                category={category}
                isActive={category.id === activeCategory}
                onClick={() => setActiveCategory(category.id)}
              />
            ))
          ) : (
            <NoDocumentRequirements customerId={customerId} />
          )}
        </div>
      </ScrollArea>
    </div>
  );
}

function ShipmentDocumentSidebarHeader({
  documentCategories,
}: {
  documentCategories: DocumentCategory[];
}) {
  return (
    <ShipmentDocumentSidebarHeaderOuter>
      <BillingReadinessBadge documentCategories={documentCategories} />
    </ShipmentDocumentSidebarHeaderOuter>
  );
}

function ShipmentDocumentSidebarHeaderOuter({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="p-4 border-b border-border">
      <h2 className="text-lg font-semibold">Document Requirements</h2>
      <p className="text-sm text-muted-foreground">
        Complete all document requirements to process the shipment
      </p>
      {children}
    </div>
  );
}
