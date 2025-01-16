import { Button } from "@/components/ui/button";
import { login } from "@/services/auth";
import { useAuthStore } from "@/stores/user-store";
import { APIError } from "@/types/errors";
import { faEnvelope, faLock } from "@fortawesome/pro-regular-svg-icons";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { useNavigate, useSearchParams } from "react-router";
import { toast } from "sonner";
import { z } from "zod";
import { InputField } from "../../../components/fields/input-field";
import { PasswordField } from "../../../components/fields/sensitive-input-field";
import { Checkbox } from "../../../components/ui/checkbox";
import { Form, FormControl, FormGroup } from "../../../components/ui/form";
import { Icon } from "../../../components/ui/icons";
import { Label } from "../../../components/ui/label";

const loginSchema = z.object({
  emailAddress: z
    .string()
    .min(1, "Email is required")
    .email("Invalid email address"),
  password: z.string().min(1, "Password is required"),
  rememberMe: z.optional(z.boolean()),
});

type LoginFormValues = z.infer<typeof loginSchema>;

type LoginFormProps = {
  email: string;
  onForgotPassword: () => void;
};

export function LoginForm({ email, onForgotPassword }: LoginFormProps) {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const setUser = useAuthStore((state) => state.setUser);

  const mutation = useMutation({
    mutationFn: async (values: LoginFormValues) => {
      const response = await login(values);
      return response.data;
    },
    onSuccess: (data) => {
      setUser(data.user);

      // Redirect to the original destination or dashboard
      const from = searchParams.get("from") || "/";
      navigate(from, { replace: true });
    },
    onError: (error: APIError) => {
      if (error.isValidationError()) {
        error.getFieldErrors().forEach((fieldError) => {
          setError(fieldError.name as keyof LoginFormValues, {
            message: fieldError.reason,
          });
        });
      } else {
        toast.error(error.message || "Failed to sign in");
      }
    },
  });

  const {
    control,
    handleSubmit,
    setError,
    formState: { isSubmitting },
  } = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      emailAddress: email,
      password: "",
      rememberMe: false,
    },
  });

  async function onSubmit(values: LoginFormValues) {
    await mutation.mutateAsync(values);
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
            placeholder="Email address"
          />
        </FormControl>

        <FormControl>
          <PasswordField
            onPasswordReset={onForgotPassword}
            icon={<Icon icon={faLock} className="size-3.5" />}
            control={control}
            rules={{ required: true }}
            name="password"
            label="Password"
            placeholder="Password"
          />
        </FormControl>

        <div className="flex items-center space-x-2">
          <Checkbox id="rememberMe" />
          <Label>Remember me</Label>
        </div>
      </FormGroup>
      <Button
        type="submit"
        className="w-full"
        isLoading={isSubmitting}
        loadingText="Signing in..."
      >
        Sign in
      </Button>
    </Form>
  );
}
