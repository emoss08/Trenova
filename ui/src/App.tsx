// @ts-expect-error fontsource-variable is not typed
import "@fontsource-variable/inter"; // Defaults to wght axis
// @ts-expect-error fontsource-variable is not typed
import "@fontsource/geist-mono";
import { isIE, isMobile, isSafari } from "react-device-detect";
import { RouterProvider } from "react-router";
import { ClientInformation } from "./components/client-information";
import { UnsupportedDevice } from "./components/unsupported-device";
import { router } from "./routing/router";

function App() {
  const unSupportedDevice = isMobile || isSafari || isIE;

  if (unSupportedDevice) {
    return <UnsupportedDevice device={{ isMobile, isSafari, isIE }} />;
  }
  return (
    <>
      <RouterProvider router={router} />
      <ClientInformation />
    </>
  );
}

export default App;
