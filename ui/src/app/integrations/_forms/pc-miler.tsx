import { InputField } from "@/components/fields/input-field";
import { FormControl } from "@/components/ui/form";
import type { PCMilerConfigData } from "@/types/integration";
import { useFormContext } from "react-hook-form";

export function PCMilerForm() {
  const { control } = useFormContext<PCMilerConfigData>();

  return (
    <>
      <FormControl>
        <InputField
          control={control}
          name="username"
          label="Username"
          rules={{ required: true }}
          placeholder="Enter your PCMiler username"
          description="Your PCMiler account username."
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="password"
          label="Password"
          rules={{ required: true }}
          placeholder="Enter your PCMiler password"
          type="password"
          description="Your PCMiler account password."
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="licenseKey"
          label="License Key"
          rules={{ required: true }}
          placeholder="Enter your PCMiler license key"
          description="The license key for your PCMiler subscription."
        />
      </FormControl>
    </>
  );
}
