import { Button } from "@/components/ui/button";
import { checkEmail } from "@/services/auth";
import { APIError } from "@/types/errors";
import { faEnvelope } from "@fortawesome/pro-regular-svg-icons";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import { InputField } from "../../../components/fields/input-field";
import { Form, FormControl, FormGroup } from "../../../components/ui/form";
import { Icon } from "../../../components/ui/icons";

const checkEmailSchema = z.object({
  emailAddress: z
    .string()
    .min(1, "Email is required")
    .email("Invalid email address"),
});

type CheckEmailFormValues = z.infer<typeof checkEmailSchema>;

type CheckEmailFormProps = {
  onEmailVerified: (email: string) => void;
};

export function CheckEmailForm({ onEmailVerified }: CheckEmailFormProps) {
  const mutation = useMutation({
    mutationFn: (values: CheckEmailFormValues) =>
      checkEmail(values.emailAddress),
  });

  const {
    control,
    handleSubmit,
    setError,
    formState: { isSubmitting },
  } = useForm<CheckEmailFormValues>({
    resolver: zodResolver(checkEmailSchema),
    defaultValues: {
      emailAddress: "",
    },
  });

  async function onSubmit(values: CheckEmailFormValues) {
    try {
      const result = await mutation.mutateAsync(values);
      if (result.data?.valid) {
        onEmailVerified(values.emailAddress);
      }
    } catch (error) {
      const err = error as APIError;
      if (err.isValidationError()) {
        err.getFieldErrors().forEach((fieldError) => {
          const fieldName = fieldError.name as keyof CheckEmailFormValues;
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
        type="submit"
        className="w-full"
        isLoading={isSubmitting}
        loadingText="Verifying..."
      >
        Continue
      </Button>
    </Form>
  );
}
