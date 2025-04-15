import { MetaTags } from "@/components/meta-tags";
import { locationAutocomplete } from "@/services/google-maps";
import { useEffect, version } from "react";

export function Dashboard() {
  // * Test function that calls the autocomplete service
  const test = async () => {
    const result = await locationAutocomplete("3255 Maple Ave");
    console.log(result);
  };

  useEffect(() => {
    test();
  }, []);

  return (
    <>
      <MetaTags title="Dashboard" description="Dashboard" />
      <span>{`${version}`}</span>
    </>
  );
}
