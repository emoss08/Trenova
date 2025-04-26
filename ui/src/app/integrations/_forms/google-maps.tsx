import logo from "@/assets/logo.webp";
import { InputField } from "@/components/fields/input-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { LazyImage } from "@/components/ui/image";
import { ExternalLink } from "@/components/ui/link";
import {
  IntegrationType,
  type GoogleMapsConfigData,
} from "@/types/integration";
import { useFormContext } from "react-hook-form";
import { integrationImages } from "../_utils/integration";

export function GoogleMapsForm() {
  const { control } = useFormContext<GoogleMapsConfigData>();

  return (
    <div className="flex flex-col gap-6">
      <GoogleMapsFormHeader />
      <FormGroup cols={1}>
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
      </FormGroup>
    </div>
  );
}

export function GoogleMapsFormHeader() {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex gap-4 items-center justify-center">
        <LazyImage src={logo} alt="Google Maps Logo" className="size-8" />
        <div className="flex items-center justify-center gap-1">
          <div className="size-1 rounded-full bg-muted-foreground" />
          <div className="size-1 rounded-full bg-muted-foreground" />
          <div className="size-1 rounded-full bg-muted-foreground" />
        </div>
        <LazyImage
          src={integrationImages[IntegrationType.GoogleMaps]}
          className="size-8"
        />
      </div>
      <div className="flex flex-col text-center gap-1">
        <h3 className="text-lg font-semibold">Connect with Google Maps</h3>
        <div className="flex justify-center gap-1">
          <p className="text-xs text-muted-foreground">
            To get a Google Maps API key, visit the
          </p>
          <ExternalLink
            href="https://console.cloud.google.com/google/maps-apis/overview"
            className="text-xs"
          >
            Google Cloud Console.
          </ExternalLink>
        </div>
      </div>
    </div>
  );
}
