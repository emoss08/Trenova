import { TableSheetProps } from "@/types/tables";
import {
    Credenza,
    CredenzaBody,
    CredenzaContent,
    CredenzaDescription,
    CredenzaHeader,
    CredenzaTitle,
} from "../ui/credenza";

export function ShipmentAdvancedSearchDialog({
  onOpenChange,
  open,
}: TableSheetProps) {
  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>Advanced Search</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Search for shipments by status, date, and more.
        </CredenzaDescription>
        <CredenzaBody>Coming soon.</CredenzaBody>
      </CredenzaContent>
    </Credenza>
  );
}
