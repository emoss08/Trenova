import { InputField } from "@/components/fields/input-field";
import { SensitiveInputField } from "@/components/fields/sensitive-input-field";
import { CircleCheckBigIcon } from "@/components/ui/animate-icons";
import { Button } from "@/components/ui/button";
import { FormControl, FormGroup } from "@/components/ui/form";
import { EmailProfileSchema } from "@/lib/schemas/email-profile-schema";
import { api } from "@/services/api";
import { useMutation } from "@tanstack/react-query";
import { useFormContext } from "react-hook-form";
import { toast } from "sonner";

export function ResendFormFields() {
  const { control, getValues } = useFormContext<EmailProfileSchema>();

  const { mutate: testConnection, isSuccess } = useMutation({
    mutationFn: async () => {
      const values = getValues();
      return await api.emailProfile.testConnection({
        providerType: values.providerType,
        host: values.host || "",
        port: values.port || 0,
        username: values.username || "",
        password: values.password || "",
        apiKey: values.apiKey || "",
      });
    },
    onSuccess: () => {
      toast.success("Connection test successful", {
        description: "The connection to the Resend API was successful.",
      });
    },
    onError: () => {
      toast.error("Connection test failed", {
        description:
          "The connection to the Resend API failed. Check your API key and try again.",
      });
    },
  });

  return (
    <div className="flex flex-col gap-4 border-t pt-4">
      <div className="flex items-center justify-between">
        <div className="flex flex-col gap-1">
          <h3
            id="resend-configuration"
            className="font-semibold leading-none tracking-tight text-sm"
          >
            Resend Configuration
          </h3>
          <p className="text-xs text-muted-foreground">
            Configure Resend settings for email delivery
          </p>
        </div>
        <Button
          variant={isSuccess ? "green" : "outline"}
          size="sm"
          type="button"
          onClick={() => testConnection()}
        >
          {isSuccess ? (
            <CircleCheckBigIcon className="size-4" startAnimation={isSuccess} />
          ) : (
            "Test Connection"
          )}
        </Button>
      </div>
      <FormGroup cols={1}>
        <FormControl cols="full">
          <InputField
            control={control}
            rules={{ required: true }}
            name="fromAddress"
            label="From Address"
            placeholder="From Address"
            description="Default sender email address for outgoing messages"
          />
        </FormControl>
        <FormControl>
          <SensitiveInputField
            control={control}
            name="apiKey"
            rules={{ required: true }}
            label="Resend API Key"
            placeholder="Resend API Key"
            description="The API key for your Resend account."
          />
        </FormControl>
      </FormGroup>
    </div>
  );
}
