/**
 * Placeholder book imagery drawn in code (no photos exist yet).
 * Swap BookPhoto for real cover scans when they exist.
 */

import Image from "next/image";
import { Book, type BookVariant } from "@/components/ui/Book";

/** Maps the palette hexes used across the pages to a Book photo variant. */
const COLOR_VARIANT: Record<string, BookVariant> = {
  "#7a2620": "red", "#5c1a16": "red", "#3f110e": "red", "#93352b": "red",
  "#2c4436": "forest", "#24382e": "forest", "#1a2a22": "forest",
  "#3c1d0e": "taupe", "#64331e": "taupe", "#8e613c": "gold",
};

type CoverProps = { title: string; author: string; color: string; className?: string };

/** Front of a leather-bound book. Picks the photo variant closest to `color`. `plain` drops the lettering (for art behind headlines). */
export function BookCover({ title, author, color, plain = false, eager = false, className = "" }: CoverProps & { plain?: boolean; eager?: boolean }) {
  return <Book title={title} author={author} variant={COLOR_VARIANT[color] ?? "forest"} plain={plain} eager={eager} className={className} />;
}

/** A cover photo on a transparent background. `tone` is unused and kept only for existing call sites. */
export function BookPhoto({ title, author, color, className = "" }: CoverProps & { tone?: "dark" | "light" }) {
  return (
    <div aria-hidden="true" className={`relative size-full ${className}`}>
      <div className="absolute left-1/2 top-1/2 w-[60%] -translate-x-1/2 -translate-y-1/2">
        <BookCover title={title} author={author} color={color} />
      </div>
    </div>
  );
}

/** Crops a padded PNG (size w x h) to its opaque box and fills the wrapper with it. */
function Cropped({ src, w, h, box, sizes, className }: { src: string; w: number; h: number; box: { x: number; y: number; w: number; h: number }; sizes: string; className: string }) {
  return (
    <div aria-hidden="true" className={`relative overflow-hidden ${className}`} style={{ aspectRatio: `${box.w} / ${box.h}` }}>
      <Image
        src={src}
        alt=""
        width={w}
        height={h}
        sizes={sizes}
        className="absolute max-w-none"
        style={{ width: `${(w / box.w) * 100}%`, height: "auto", left: `${(-box.x / box.w) * 100}%`, top: `${(-box.y / box.h) * 100}%` }}
      />
    </div>
  );
}

/** A stack of books seen from the side (public/images/book-stack.png). */
export function SpineStack({ className = "" }: { className?: string }) {
  return <Cropped src="/images/book-stack.png" w={627} h={398} box={{ x: 90, y: 56, w: 447, h: 277 }} sizes="(max-width: 768px) 60vw, 400px" className={className} />;
}

/** An open book (public/images/book-open.png). */
export function OpenBook({ className = "" }: { className?: string }) {
  return <Cropped src="/images/book-open.png" w={593} h={421} box={{ x: 42, y: 36, w: 509, h: 351 }} sizes="(max-width: 768px) 80vw, 460px" className={className} />;
}
