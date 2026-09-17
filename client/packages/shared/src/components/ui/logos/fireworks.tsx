import { cn } from "@trenova/shared/lib/utils";

/**
 * Fireworks AI's three petals. Evenodd is what keeps them legible in one colour:
 * the vendor separates them with gradients, and filled flat they would merge
 * into a blob, so the overlaps are punched out instead.
 */
export function FireworksLogo({ className }: { className?: string }) {
  return (
    <svg
      className={cn("size-4", className)}
      viewBox="0 0 24 24"
      fill="currentColor"
      fillRule="evenodd"
      role="img"
      aria-hidden="true"
      focusable="false"
    >
      <path d="M7.95343 21.1394C4.89586 21.1304 2.25471 19.458 0.987296 16.2895C-0.280118 13.121 0.108924 9.16314 1.74363 5.61505C4.8012 5.62409 7.44235 7.29648 8.70976 10.465C9.97718 13.6335 9.58814 17.5914 7.95343 21.1394ZM7.31038 21.2574C11.3543 20.2215 14.8836 17.3754 16.6285 13.2361C18.3735 9.09671 17.9448 4.58749 15.8598 0.976291C11.8159 2.01214 8.2866 4.85826 6.54167 8.99762C4.79674 13.137 5.2254 17.6462 7.31038 21.2574ZM7.23368 21.2069C9.78906 23.2373 13.2102 23.9506 16.5772 22.8141C19.9441 21.6775 22.5058 18.9445 23.7304 15.6382C21.175 13.6078 17.7538 12.8944 14.3869 14.031C11.0199 15.1676 8.45822 17.9006 7.23368 21.2069Z" />
    </svg>
  );
}
