import { useEffect, useState } from "react";

/**
 * An object URL for a blob, made when the blob arrives and revoked when it changes or the
 * component goes. It is made in an effect, not during render, so React's development
 * double mount revokes one URL and makes the next instead of leaving a revoked one in use.
 */
export function useObjectUrl(blob: Blob | null | undefined): string | null {
  const [url, setUrl] = useState<string | null>(null);

  useEffect(() => {
    if (!blob) {
      setUrl(null);
      return;
    }
    const next = URL.createObjectURL(blob);
    setUrl(next);
    return () => URL.revokeObjectURL(next);
  }, [blob]);

  return url;
}
