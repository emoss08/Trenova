import { InputField } from "@/components/fields/input-field";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { Icon } from "@/components/ui/icons";
import { checkEmailSchema, CheckEmailSchema } from "@/lib/schemas/auth-schema";
import { checkEmail } from "@/services/auth";
import { APIError } from "@/types/errors";
import { faEnvelope } from "@fortawesome/pro-regular-svg-icons";
import { yupResolver } from "@hookform/resolvers/yup";
import { useMutation } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

type CheckEmailFormProps = {
  onEmailVerified: (email: string) => void;
};

export function CheckEmailForm({ onEmailVerified }: CheckEmailFormProps) {
  const mutation = useMutation({
    mutationFn: (values: CheckEmailSchema) => checkEmail(values.emailAddress),
  });

  const {
    control,
    handleSubmit,
    setError,
    formState: { isSubmitting },
  } = useForm<CheckEmailSchema>({
    resolver: yupResolver(checkEmailSchema),
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
        isLoading={isSubmitting}
        loadingText="Verifying..."
      >
        Continue
      </Button>
    </Form>
  );
}
