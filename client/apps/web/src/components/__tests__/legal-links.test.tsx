import { LegalAgreementNote } from "@/routes/auth/_components/legal-agreement-note";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { publicConfigQueryOptions } from "@trenova/shared/hooks/use-public-config";
import { SELF_HOSTED_PUBLIC_CONFIG } from "@trenova/shared/types/platform";
import { describe, expect, it } from "vitest";
import { LegalLinks } from "../legal-links";

function renderNote(config: { termsUrl: string; privacyUrl: string }) {
  const queryClient = new QueryClient();
  queryClient.setQueryData(publicConfigQueryOptions.queryKey, {
    ...SELF_HOSTED_PUBLIC_CONFIG,
    ...config,
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <LegalAgreementNote />
    </QueryClientProvider>,
  );
}

describe("LegalLinks", () => {
  it("joins both documents", () => {
    render(
      <p>
        <LegalLinks
          urls={{
            termsUrl: "https://example.com/terms",
            privacyUrl: "https://example.com/privacy",
          }}
        />
      </p>,
    );

    expect(screen.getByRole("link", { name: "Terms of Service" })).toHaveAttribute(
      "href",
      "https://example.com/terms",
    );
    expect(screen.getByRole("link", { name: "Privacy policy" })).toHaveAttribute(
      "href",
      "https://example.com/privacy",
    );
  });

  it("names only the document the install publishes", () => {
    const { container } = render(
      <p>
        <LegalLinks urls={{ termsUrl: "", privacyUrl: "https://example.com/privacy" }} />
      </p>,
    );

    expect(screen.queryByRole("link", { name: "Terms of Service" })).not.toBeInTheDocument();
    expect(container.textContent).toBe("Privacy policy");
  });
});

describe("LegalAgreementNote", () => {
  it("says nothing when the install publishes neither document", () => {
    const { container } = renderNote({ termsUrl: "", privacyUrl: "" });

    expect(container).toBeEmptyDOMElement();
  });

  it("links the documents the server's public config names", () => {
    renderNote({ termsUrl: "https://example.com/terms", privacyUrl: "" });

    expect(screen.getByText(/By continuing you agree to our/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Terms of Service" })).toHaveAttribute(
      "href",
      "https://example.com/terms",
    );
    expect(screen.queryByRole("link", { name: "Privacy policy" })).not.toBeInTheDocument();
  });
});
