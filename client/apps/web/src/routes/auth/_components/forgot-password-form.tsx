import { useApiMutation } from "@/hooks/use-api-mutation";
import { authService } from "@trenova/shared/services/auth";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { AuthCardBody } from "./auth-card";
import { AuthErrorText, AuthSubmit, AuthTextField } from "./auth-field";
import { AuthTray, StepCrumbs, StepHeading } from "./auth-primitives";

export const forgotPasswordSchema = z.object({
  emailAddress: z.email({ error: "Please enter a valid email address" }),
});

export type ForgotPasswordRequest = z.infer<typeof forgotPasswordSchema>;

/**
 * Step 1 of recovery: ask for the address.
 *
 * The confirmation this shows on success is deliberately non-committal — "if that
 * address has an account". The server answers the same way for an address that
 * belongs to nobody as for one that does, and a screen that said "sent!" only for
 * real accounts would give away exactly what the server refuses to.
 */
export function ForgotPasswordForm({
  defaultEmail,
  onBack,
}: {
  defaultEmail?: string;
  onBack: () => void;
}) {
  const form = useForm<ForgotPasswordRequest>({
    resolver: zodResolver(forgotPasswordSchema),
    defaultValues: { emailAddress: defaultEmail ?? "" },
  });
  const { control, handleSubmit, formState } = form;
  const rootError = formState.errors.root?.message;

  const { mutateAsync, isPending, isSuccess } = useApiMutation({
    mutationFn: (data: ForgotPasswordRequest) => authService.forgotPassword(data.emailAddress),
    form,
    resourceName: "Password reset",
  });

  if (isSuccess) {
    return <ForgotPasswordSent onBack={onBack} />;
  }

  return (
    <AuthCardBody>
      <StepCrumbs left="Account recovery" right="Secure sign-in" />
      <StepHeading title="Reset your password">
        Enter the address you sign in with and we&apos;ll send you a link to choose a new password.
      </StepHeading>

      <form
        className="mt-4 flex flex-col gap-3.5"
        noValidate
        onSubmit={handleSubmit((data) => void mutateAsync(data))}
      >
        <AuthTextField
          name="emailAddress"
          control={control}
          label="Email address"
          type="email"
          required
          placeholder="name@work-email.com"
          autoComplete="username"
          disabled={isPending}
        />
        {rootError && <AuthErrorText>{rootError}</AuthErrorText>}
        <AuthSubmit type="submit" isLoading={isPending} loadingText="Sending link">
          Send reset link
        </AuthSubmit>
      </form>

      <AuthTray onBack={onBack} hints={<span>Remembered it? Go back.</span>} />
    </AuthCardBody>
  );
}

function ForgotPasswordSent({ onBack }: { onBack: () => void }) {
  return (
    <AuthCardBody>
      <StepCrumbs left="Account recovery" right="Link sent" />
      <StepHeading title="Check your inbox">
        If that address has an account, a reset link is on its way. The link works once and expires
        shortly, so use it soon.
      </StepHeading>

      <p className="text-subtle-foreground mt-4 mb-0 text-[11.5px]">
        Nothing arrived? Check spam, then try again — and confirm you used the address your
        administrator set the account up with.
      </p>

      <div className="mt-4">
        <AuthSubmit onClick={onBack}>Back to sign in</AuthSubmit>
      </div>
    </AuthCardBody>
  );
}
