/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import licenseConfig from "@/lib/license";
import { useEffect, useState } from "react";
import ReactMarkdown from "react-markdown";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "./ui/dialog";

export function LicenseInformation({
  open,
  onOpenChange,
  licenseUrl = "https://raw.githubusercontent.com/emoss08/Trenova/refs/heads/master/LICENSE.md",
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  licenseUrl?: string;
}) {
  const [licenseMarkdown, setLicenseMarkdown] = useState<string>("");
  const [isLoading, setIsLoading] = useState<boolean>(true);

  useEffect(() => {
    const fetchLicense = async () => {
      try {
        const response = await fetch(licenseUrl);
        if (!response.ok) {
          throw new Error(`Failed to fetch license: ${response.status}`);
        }

        const text = await response.text();
        setLicenseMarkdown(text);
      } catch (error) {
        console.error("Error fetching license:", error);
        setLicenseMarkdown(
          "# Error\nFailed to load license content. Please try again later.",
        );
      } finally {
        setIsLoading(false);
      }
    };

    fetchLicense();
  }, [licenseUrl]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[625px]">
        <DialogHeader>
          <DialogTitle>Trenova License Information</DialogTitle>
          <DialogDescription>
            Transportation Management System software license details
          </DialogDescription>
        </DialogHeader>
        <DialogBody>
          {isLoading ? (
            <div className="flex items-center justify-center h-40">
              <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary"></div>
            </div>
          ) : (
            <div className="space-y-4 text-sm license-markdown">
              <ReactMarkdown>{licenseMarkdown}</ReactMarkdown>

              <h3 className="text-lg font-semibold mt-6">
                Third-Party Components
              </h3>
              <p className="text-muted-foreground">
                Trenova includes several third-party open source components,
                each with their own licenses:
              </p>
              <div className="space-y-4">
                {licenseConfig.thirdPartyLicenses.map((license) => (
                  <div key={license.name}>
                    <h4 className="font-medium">{license.name}</h4>
                    <a
                      target="_blank"
                      rel="noreferrer"
                      href={license.url}
                      className="text-xs text-muted-foreground hover:underline cursor-pointer"
                    >
                      {license.license} - {license.copyright}
                    </a>
                  </div>
                ))}
              </div>

              <p className="text-muted-foreground mt-6">
                For the complete text of these licenses, please visit the
                respective project websites or view the license files included
                with the software.
              </p>
            </div>
          )}
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}

// Add some custom styles for the Markdown content
const styles = `
  .license-markdown {
    /* Add any custom styles for your markdown content here */
  }
  
  .license-markdown h1 {
    font-size: 1.5rem;
    font-weight: 600;
    margin-bottom: 1rem;
  }
  
  .license-markdown h2 {
    font-size: 1.25rem;
    font-weight: 600;
    margin-top: 1rem;
    margin-bottom: 0.75rem;
  }
  
  .license-markdown p {
    margin-bottom: 0.75rem;
  }
  
  .license-markdown ul, .license-markdown ol {
    padding-left: 1.5rem;
    margin-bottom: 0.75rem;
  }
  
  .license-markdown li {
    margin-bottom: 0.25rem;
  }
  
  .license-markdown code {
    background-color: rgba(0, 0, 0, 0.05);
    padding: 0.2rem 0.4rem;
    border-radius: 0.25rem;
    font-size: 0.875rem;
  }
`;

// Inject the styles into the document
if (typeof document !== "undefined") {
  const styleElement = document.createElement("style");
  styleElement.innerHTML = styles;
  document.head.appendChild(styleElement);
}
