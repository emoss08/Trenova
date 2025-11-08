/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Button } from "@/components/ui/button";
import { FormCreateModal } from "@/components/ui/form-create-modal";
import { Icon } from "@/components/ui/icons";
import { userSchema } from "@/lib/schemas/user-schema";
import { TIMEZONES } from "@/lib/timezone/timezone";
import { useNotice } from "@/stores/user-preference-store";
import { useUser } from "@/stores/user-store";
import { Status } from "@/types/common";
import type { TableSheetProps } from "@/types/data-table";
import { TimeFormat } from "@/types/user";
import { faInfoCircle, faXmark } from "@fortawesome/pro-regular-svg-icons";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { UserForm } from "./user-form";

export function CreateUserModal({ open, onOpenChange }: TableSheetProps) {
  const user = useUser();

  const form = useForm({
    resolver: zodResolver(userSchema),
    defaultValues: {
      status: Status.Active,
      currentOrganizationId: user?.currentOrganizationId,
      emailAddress: "",
      username: "",
      name: "",
      mustChangePassword: true,
      timezone: TIMEZONES[0].value,
      isLocked: false,
      thumbnailUrl: undefined,
      timeFormat: TimeFormat.TimeFormat24Hour,
      profilePicUrl: undefined,
      version: undefined,
      id: undefined,
      createdAt: undefined,
      updatedAt: undefined,
      lastLoginAt: undefined,
      organizations: [],
      organizationMemberships: [],
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="User"
      formComponent={<UserForm />}
      form={form}
      url="/users/"
      queryKey="user-list"
      className="max-w-[500px]"
      notice={<UserCreateNotice />}
    />
  );
}

function UserCreateNotice() {
  const { isDismissed, dismiss } = useNotice("user-create-notice");

  return !isDismissed ? (
    <div className="bg-purple-500/20 px-4 py-3 text-purple-500">
      <div className="flex gap-2">
        <div className="flex grow gap-3">
          <Icon
            icon={faInfoCircle}
            className="mt-0.5 shrink-0 text-purple-500"
            aria-hidden="true"
          />
          <div className="flex grow flex-col justify-between gap-2 md:flex-row">
            <span className="text-sm">
              Upon creation, the user will receive an email with a temporary
              password. Upon login, the user will be required to change their
              password.
            </span>
          </div>
        </div>
        <Button
          variant="ghost"
          className="group -my-1.5 -me-2 size-8 shrink-0 p-0 hover:bg-transparent"
          onClick={dismiss}
          aria-label="Close banner"
        >
          <Icon
            icon={faXmark}
            className="opacity-60 transition-opacity group-hover:opacity-100"
            aria-hidden="true"
          />
        </Button>
      </div>
    </div>
  ) : null;
}
