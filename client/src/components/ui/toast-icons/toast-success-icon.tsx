import { m, useReducedMotion } from "motion/react";

const draw = {
  hidden: { pathLength: 0, opacity: 0 },
  visible: (delay: number) => ({
    pathLength: 1,
    opacity: 1,
    transition: {
      pathLength: { delay, duration: 0.4, ease: "easeOut" as const },
      opacity: { delay, duration: 0.01 },
    },
  }),
};

export function ToastSuccessIcon() {
  const reduced = useReducedMotion();

  if (reduced) {
    return (
      <svg
        width="16"
        height="16"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="12" cy="12" r="10" />
        <path d="m9 12 2 2 4-4" />
      </svg>
    );
  }

  return (
    <m.svg
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      initial="hidden"
      animate="visible"
    >
      <m.circle cx="12" cy="12" r="10" variants={draw} custom={0} />
      <m.path d="m9 12 2 2 4-4" variants={draw} custom={0.3} />
    </m.svg>
  );
}
