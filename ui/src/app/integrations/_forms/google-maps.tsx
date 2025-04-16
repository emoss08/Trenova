import { InputField } from "@/components/fields/input-field";
import { FormControl } from "@/components/ui/form";
import type { GoogleMapsConfigData } from "@/types/integrations/google-maps";
import { useFormContext } from "react-hook-form";

export function GoogleMapsForm() {
  const { control } = useFormContext<GoogleMapsConfigData>();

  return (
    <>
      <FormControl>
        <InputField
          control={control}
          name="apiKey"
          label="API Key"
          rules={{ required: true }}
          placeholder="Enter your Google Maps API Key"
          autoComplete="off"
          type="password"
          description="Enter your Google Maps API key from the Google Cloud Console."
        />
      </FormControl>
      <p className="text-xs text-muted-foreground">
        To get a Google Maps API key, visit the{" "}
        <a
          href="https://console.cloud.google.com/google/maps-apis/overview"
          target="_blank"
          rel="noopener noreferrer"
          className="font-medium underline"
        >
          Google Cloud Console
        </a>
      </p>
    </>
  );
}
