/** Hand-drawn wicker basket (replaces the lucide shopping bag). Inherits color and size from the parent. */
export function BasketIcon({ size = 22, className = "" }: { size?: number; className?: string }) {
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 24 24"
      width={size}
      height={size}
      fill="none"
      stroke="currentColor"
      strokeWidth="1.4"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
    >
      <defs>
        <filter id="basket-ink" x="-10%" y="-10%" width="120%" height="120%">
          <feTurbulence type="fractalNoise" baseFrequency=".9" numOctaves="2" seed="4" result="n" />
          <feDisplacementMap in="SourceGraphic" in2="n" scale="0.9" />
        </filter>
      </defs>
      <g filter="url(#basket-ink)">
        {/* arched handle */}
        <path d="M6.6 10.2C6.2 3.6 17.8 3.4 17.4 10.2" />
        {/* rim + body */}
        <path d="M3.4 10.4h17.2" />
        <path d="M4.2 12.6h15.6" />
        <path d="M4.8 12.6 6 19.2c3.8 1.4 8.2 1.4 12 0l1.2-6.6" />
        {/* weave */}
        <path d="M8.4 12.8l.6 6.6M12 12.8v7M15.6 12.8l-.6 6.6" />
        <path d="M5.3 16.1q6.7 1.2 13.4 0" />
      </g>
    </svg>
  );
}
