import Link from "next/link";

type DotButtonProps = {
  href: string;
  children: React.ReactNode;
  variant?: "outline" | "solid";
  className?: string;
};

/** Hand-inked brush-stroke link with a small fleuron that nudges on hover. */
export function DotButton({ href, children, variant = "outline", className = "" }: DotButtonProps) {
  return (
    <Link
      href={href}
      className={`dot-btn ${variant === "solid" ? "dot-btn--solid" : ""} ${className}`.trim()}
    >
      {children}
      <span className="dot-btn__dot" aria-hidden="true">
        ❧
      </span>
    </Link>
  );
}
