import Image from "next/image";

export type BookVariant = "forest" | "green" | "red" | "gold" | "navy" | "taupe";

type BookProps = { title: string; author: string; variant?: BookVariant; plain?: boolean; className?: string };

/** Leather book photo (public/images/book-<variant>.png) with the title laid over the cover, absolutely positioned. */
export function Book({ title, author, variant = "forest", plain = false, className = "" }: BookProps) {
  return (
    <div
      aria-hidden="true"
      className={`relative aspect-[315/433] overflow-hidden [container-type:inline-size] ${className}`}
      style={{ filter: "drop-shadow(0 24px 30px rgba(0,0,0,.55))" }}
    >
      {/* book-*.png are 436x573 with padding; crop to the book body */}
      <Image
        src={`/images/book-${variant}.png`}
        alt=""
        width={436}
        height={573}
        sizes="(max-width: 768px) 70vw, 420px"
        className="absolute max-w-none"
        style={{ width: "138.4%", height: "auto", left: "-19%", top: "-16.6%" }}
      />
      {!plain && (
        <div className="absolute inset-y-[14%] left-[20%] right-[12%] flex overflow-hidden flex-col items-center justify-center gap-[4cqw] text-center">
          <span className="line-clamp-5 break-words font-serif text-[min(7cqw,26px)] uppercase leading-[1.4] tracking-[0.04em] text-gold-300 [text-wrap:balance]">
            {title}
          </span>
          <span className="line-clamp-2 break-words font-serif text-[min(5cqw,18px)] italic text-gold-400">{author}</span>
        </div>
      )}
    </div>
  );
}
