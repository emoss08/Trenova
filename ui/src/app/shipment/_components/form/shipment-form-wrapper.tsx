/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Form } from "@/components/ui/form";
import { type ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { memo } from "react";
import { useFormContext } from "react-hook-form";

interface ShipmentFormWrapperProps {
  onSubmit: (values: ShipmentSchema) => Promise<void>;
  children: React.ReactNode;
  className?: string;
}

function ShipmentFormWrapperComponent({
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

export const ShipmentFormWrapper = memo(ShipmentFormWrapperComponent);
