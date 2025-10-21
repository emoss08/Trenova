import { Button } from "@/components/ui/button";
import { FormCreateModal } from "@/components/ui/form-create-modal";
import { Icon } from "@/components/ui/icons";
import { ExternalLink } from "@/components/ui/link";
import { RESEND_ALERT_KEY } from "@/constants/env";
import { useLocalStorage } from "@/hooks/use-local-storage";
import {
  AuthType,
  emailProfileSchema,
  EncryptionType,
  ProviderType,
} from "@/lib/schemas/email-profile-schema";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { faInfoCircle, faXmark } from "@fortawesome/pro-regular-svg-icons";
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
      className="sm:max-w-[500px]"
      notice={<ResendAlert />}
    />
  );
}

function ResendAlert() {
  const [noticeVisible, setNoticeVisible] = useLocalStorage(
    RESEND_ALERT_KEY,
    true,
  );

  const handleClose = () => {
    setNoticeVisible(false);
  };

  return noticeVisible ? (
    <div className="bg-muted px-4 py-3 text-foreground">
      <div className="flex gap-2">
        <div className="flex grow gap-3">
          <Icon
            icon={faInfoCircle}
            className="mt-0.5 shrink-0 text-foreground"
            aria-hidden="true"
          />
          <div className="flex grow flex-col justify-between gap-2 md:flex-row">
            <span className="text-sm">
              We highly recommend using <strong>Resend</strong> for your email
              needs. They provide a generous free tier and are a great option
              for sending emails. You can sign up for a free account at{" "}
              <ExternalLink href="https://resend.com">Resend</ExternalLink>.
            </span>
          </div>
        </div>
        <Button
          variant="secondary"
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
