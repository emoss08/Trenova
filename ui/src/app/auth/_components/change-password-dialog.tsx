/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Icon } from "@/components/ui/icons";
import { VisuallyHidden } from "@/components/ui/visually-hidden";
import type { TableSheetProps } from "@/types/data-table";
import { faInfoCircle } from "@fortawesome/pro-solid-svg-icons";
import ChangePasswordForm from "./change-password-form";

type ChangePasswordDialogProps = {
  mustChangePassword: boolean;
} & TableSheetProps;

export function ChangePasswordDialog({
  mustChangePassword,
  ...props
}: ChangePasswordDialogProps) {
  const { onOpenChange } = props;

  return (
    <Dialog
      {...props}
      onOpenChange={mustChangePassword ? undefined : onOpenChange}
    >
      <DialogContent
        withClose={false}
        onPointerDownOutside={
          mustChangePassword ? (e) => e.preventDefault() : undefined
        }
        onEscapeKeyDown={
          mustChangePassword ? (e) => e.preventDefault() : undefined
        }
        onInteractOutside={
          mustChangePassword ? (e) => e.preventDefault() : undefined
        }
      >
        <VisuallyHidden>
          <DialogHeader>
            <DialogTitle>Change Password</DialogTitle>
            <DialogDescription>
              You must change your password to continue.
            </DialogDescription>
          </DialogHeader>
        </VisuallyHidden>
        <DialogBody className="flex flex-col gap-1">
          {mustChangePassword && <MustChangePasswordNotice />}
          <ChangePasswordForm onOpenChange={onOpenChange} />
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}

function MustChangePasswordNotice() {
  return (
    <div className="bg-red-500/20 px-4 py-3 text-red-500 border border-red-500 rounded-md mb-4">
      <div className="flex gap-2">
        <div className="flex grow gap-3">
          <Icon
            icon={faInfoCircle}
            className="mt-0.5 shrink-0 text-red-500"
            aria-hidden="true"
          />
          <div className="flex grow flex-col">
            <h3 className="text-sm font-bold uppercase">
              Password Change Required
            </h3>
            <span className="text-sm">
              Please update your password to continue. This is required due to
              password expiration or new account setup.
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}
