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

export function ToastWarningIcon() {
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
        <path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z" />
        <path d="M12 9v4" />
        <circle cx="12" cy="17" r="0.5" fill="currentColor" />
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
      <m.path
        d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z"
        variants={draw}
        custom={0}
      />
      <m.path d="M12 9v4" variants={draw} custom={0.35} />
      <m.circle
        cx="12"
        cy="17"
        r="0.5"
        fill="currentColor"
        initial={{ scale: 0, opacity: 0 }}
        animate={{ scale: [0, 1.2, 1], opacity: 1 }}
        transition={{ delay: 0.5, duration: 0.3, ease: "easeOut" }}
      />
    </m.svg>
  );
}
