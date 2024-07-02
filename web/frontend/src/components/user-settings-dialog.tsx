import { User } from "@/types/accounts";
import { Credenza, CredenzaBody, CredenzaContent } from "./ui/credenza";
import UserProfile from "./user-settings/profile-page";

type UserSettingsDialogProps = {
  onOpenChange: () => void;
  open: boolean;
  user: User;
};

export function UserSettingsDialog({
  onOpenChange,
  open,
  user,
}: UserSettingsDialogProps) {
  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent className="max-w-[60em]">
        <CredenzaBody>
          <UserProfile user={user} />
        </CredenzaBody>
      </CredenzaContent>
    </Credenza>
  );
}
