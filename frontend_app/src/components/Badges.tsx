interface BadgeProps {
  text: string;
  bgColor?: string;
  textColor?: string;
  dotColor?: string;
}

function Badge(props: BadgeProps) {
  return (
    <>
      <span
        className={`inline-flex items-center rounded ${
          props.bgColor || "bg-indigo-100"
        } px-2 py-0.5 text-xs font-medium ${props.textColor || "text-white"}`}
      >
        <svg
          className={`mr-1.5 h-2 w-2 ${props.dotColor || "text-indigo-400"}`}
          fill="currentColor"
          viewBox="0 0 8 8"
        >
          <circle cx={4} cy={4} r={3} />
        </svg>
        {props.text}
      </span>
    </>
  );
}

export default Badge;
