import { Skeleton } from "@/components/ui/skeleton";
import { useUserOrganization } from "@/hooks/useQueries";
import { Image } from "@unpic/react";
import { Link } from "react-router-dom";

export function OrganizationLogo() {
  const { userOrganizationData, userOrganizationLoading } =
    useUserOrganization();

  if (userOrganizationLoading) {
    return <Skeleton className="h-10 w-40" />;
  }

  if (userOrganizationData && userOrganizationData.logoUrl) {
    return (
      <LogoLink src={userOrganizationData.logoUrl} alt="Organization Logo" />
    );
  }

  return (
    <span className="max-w-[200px] truncate">
      <Link
        className="text-accent-foreground text-xl font-semibold"
        to="/"
        title={userOrganizationData?.name}
      >
        {userOrganizationData?.name}
      </Link>
    </span>
  );
}

export function OrganizationNameLogo() {
  const { userOrganizationData, userOrganizationLoading } =
    useUserOrganization();

  if (userOrganizationLoading) {
    return <Skeleton className="h-10 w-40" />;
  }

  return (
    <span className="max-w-[200px] truncate">
      <Link
        className="text-accent-foreground text-xl font-bold"
        to="/"
        title={userOrganizationData?.name}
      >
        {userOrganizationData?.name}
      </Link>
    </span>
  );
}

function LogoLink({ src, alt }: { src: string; alt: string }) {
  return (
    <Link to="/" style={{ textDecoration: "none" }}>
      <Image
        src={src}
        layout="constrained"
        className="object-contain"
        width={40}
        height={40}
        alt={alt}
      />
    </Link>
  );
}
