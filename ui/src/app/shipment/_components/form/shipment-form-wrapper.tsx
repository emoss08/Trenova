import { Form } from "@/components/ui/form";
import { type ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { useFormContext } from "react-hook-form";

interface ShipmentFormWrapperProps {
  onSubmit: (values: ShipmentSchema) => Promise<void>;
  children: React.ReactNode;
  className?: string;
}

export function ShipmentFormWrapper({
  onSubmit,
  children,
  className = "space-y-0 p-0",
}: ShipmentFormWrapperProps) {
  const form = useFormContext<ShipmentSchema>();

  return (
    <Form className={className} onSubmit={form.handleSubmit(onSubmit)}>
      {children}
    </Form>
  );
}
