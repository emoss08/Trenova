import { InputField } from "@/components/fields/input-field";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { Icon } from "@/components/ui/icons";
import { TURNSTILE_SITE_KEY } from "@/constants/env";
import { checkEmailSchema, CheckEmailSchema } from "@/lib/schemas/auth-schema";
import { api } from "@/services/api";
import { APIError } from "@/types/errors";
import { faEnvelope } from "@fortawesome/pro-regular-svg-icons";
import { zodResolver } from "@hookform/resolvers/zod";
import { Turnstile } from "@marsidev/react-turnstile";
import { useMutation } from "@tanstack/react-query";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

type CheckEmailFormProps = {
  onEmailVerified: (email: string) => void;
};

type TurnstileStatus = "error" | "expired" | "solved";

export function CheckEmailForm({ onEmailVerified }: CheckEmailFormProps) {
  const mutation = useMutation({
    mutationFn: async (values: CheckEmailSchema) =>
      await api.auth.checkEmail(values.emailAddress),
  });
  const [status, setStatus] = useState<TurnstileStatus | null>(null);

  console.info("Turnstile status", status);

  const {
    control,
    handleSubmit,
    setError,
    formState: { isSubmitting },
  } = useForm<CheckEmailSchema>({
    resolver: zodResolver(checkEmailSchema),
    defaultValues: {
      emailAddress: "",
    },
  });

  async function onSubmit(values: CheckEmailSchema) {
    try {
      const result = await mutation.mutateAsync(values);
      if (result.data?.valid) {
        onEmailVerified(values.emailAddress);
      }
    } catch (error) {
      const err = error as APIError;
      if (err.isValidationError()) {
        err.getFieldErrors().forEach((fieldError) => {
          const fieldName = fieldError.name as keyof CheckEmailSchema;
          setError(fieldName, {
            message: fieldError.reason,
          });
        });
      } else if (err.isAuthorizationError()) {
        toast.error(err.data?.detail);
      }
    }
  }

  return (
    <Form onSubmit={handleSubmit(onSubmit)}>
      <FormGroup cols={1}>
        <FormControl>
          <InputField
            icon={<Icon icon={faEnvelope} className="size-3.5" />}
            control={control}
            rules={{ required: true }}
            name="emailAddress"
            label="Email address"
            placeholder="Enter your email address"
          />
        </FormControl>
      </FormGroup>
      <Button
        size="lg"
        type="submit"
        className="w-full"
        isLoading={isSubmitting || status !== "solved"}
        loadingText="Verifying..."
        disabled={status !== "solved"}
      >
        Continue
      </Button>
      <div className="flex w-full justify-center mt-4">
        <Turnstile
          siteKey={TURNSTILE_SITE_KEY}
          options={{
            action: "email-verification",
            appearance: "execute",
            size: "flexible",
          }}
          onSuccess={() => setStatus("solved")}
          onError={() => setStatus("error")}
          onExpire={() => setStatus("expired")}
        />
      </div>
    </Form>
  );
}
