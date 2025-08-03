/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { FormCreateModal } from "@/components/ui/form-create-modal";
import {
  AuthType,
  emailProfileSchema,
  EncryptionType,
  ProviderType,
} from "@/lib/schemas/email-profile-schema";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { EmailProfileForm } from "./email-profile-form";

export function CreateEmailProfileModal({
  open,
  onOpenChange,
}: TableSheetProps) {
  const form = useForm({
    resolver: zodResolver(emailProfileSchema),
    defaultValues: {
      status: Status.Active,
      name: "",
      authType: AuthType.enum.Plain,
      providerType: ProviderType.enum.SMTP,
      encryptionType: EncryptionType.enum.None,
      host: "",
      description: "",
      fromAddress: "",
      fromName: "",
      replyTo: "",
      isDefault: false,
      maxConnections: 5,
      timeoutSeconds: 30,
      retryCount: 3,
      rateLimitPerMinute: 60,
      rateLimitPerHour: 1000,
      rateLimitPerDay: 10000,
      metadata: undefined,
      username: "",
      port: undefined,
      oauth2ClientId: undefined,
      oauth2TenantId: undefined,
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Email Profile"
      formComponent={<EmailProfileForm />}
      form={form}
      url="/email-profiles/"
      queryKey="email-profile-list"
    />
  );
}
