import { Button } from "@/components/ui/button";
import { FormCreateModal } from "@/components/ui/form-create-modal";
import { Icon } from "@/components/ui/icons";
import { USER_CREATE_NOTICE_KEY } from "@/constants/env";
import { userSchema } from "@/lib/schemas/user-schema";
import { TIMEZONES } from "@/lib/timezone/timezone";
import { useUser } from "@/stores/user-store";
import { Status } from "@/types/common";
import type { TableSheetProps } from "@/types/data-table";
import { faInfoCircle, faXmark } from "@fortawesome/pro-regular-svg-icons";
import { zodResolver } from "@hookform/resolvers/zod";
import { useLocalStorage } from "@uidotdev/usehooks";
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
      id: undefined,
      thumbnailUrl: undefined,
      profilePicUrl: undefined,
      version: undefined,
      createdAt: undefined,
      updatedAt: undefined,
      timeFormat: undefined,
      lastLoginAt: undefined,
      roles: [],
      organizations: [],
      currentOrganization: undefined,
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
  const [noticeVisible, setNoticeVisible] = useLocalStorage(
    USER_CREATE_NOTICE_KEY,
    true,
  );

  const handleClose = () => {
    setNoticeVisible(false);
  };

  return noticeVisible ? (
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
          onClick={handleClose}
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
