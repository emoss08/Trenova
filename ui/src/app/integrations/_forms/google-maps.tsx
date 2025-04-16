import { InputField } from "@/components/fields/input-field";
import { FormControl } from "@/components/ui/form";
import { Icon } from "@/components/ui/icons";
import type { GoogleMapsConfigData } from "@/types/integrations/google-maps";
import { faInfoCircle } from "@fortawesome/pro-regular-svg-icons";
import { useFormContext } from "react-hook-form";

export function GoogleMapsForm() {
  const { control, formState } = useFormContext<GoogleMapsConfigData>();
  const { dirtyFields } = formState;

  return (
    <>
      <FormControl>
        <InputField
          control={control}
          name="apiKey"
          label="API Key"
          rules={{ required: true }}
          placeholder="Enter your Google Maps API Key"
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
      {Object.keys(dirtyFields).length > 0 && (
        <div className="mt-2 rounded-md bg-blue-50 p-2">
          <p className="text-xs text-blue-700">
            <Icon icon={faInfoCircle} className="mr-1 h-3 w-3" />
            Changes will be saved when you click Update Configuration.
          </p>
        </div>
      )}
    </>
  );
}
