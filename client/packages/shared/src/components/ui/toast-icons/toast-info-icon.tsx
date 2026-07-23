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

export function ToastInfoIcon() {
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
        <path d="M12 16v-4" />
        <circle cx="12" cy="8" r="0.5" fill="currentColor" />
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
      <m.path d="M12 16v-4" variants={draw} custom={0.3} />
      <m.circle
        cx="12"
        cy="8"
        r="0.5"
        fill="currentColor"
        initial={{ scale: 0, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        transition={{ delay: 0.3, duration: 0.2, ease: "easeOut" }}
      />
    </m.svg>
  );
}
