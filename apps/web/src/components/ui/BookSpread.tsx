"use client";

import Image from "next/image";
import Link from "next/link";
import { useLayoutEffect, useRef, type CSSProperties, type MouseEventHandler, type RefObject } from "react";
import { BookCover, bookVariant } from "@/components/ui/BookArt";
import type { Book } from "@/lib/catalogue";

export type OpenableBook = Pick<
  Book,
  "slug" | "title" | "author" | "year" | "color" | "edition" | "condition" | "subjectName" | "description" | "publisher" | "language" | "pages"
>;

type More = { href: string; onClick?: MouseEventHandler<HTMLAnchorElement> };

type SpreadProps = {
  book: OpenableBook;
  /** The front board has swung round the spine onto the left-hand page. */
  open: boolean;
  /** Shuts at the closing speed (the catalogue preview flying back to the shelf). */
  closing?: boolean;
  /** Keeps the introduction blank while the board is open (the book is still in flight). */
  hush?: boolean;
  /** id for the "Introduction" heading, when a dialog is labelled by it. */
  titleId?: string;
  /** Adds a "View this copy" link at the foot of the page. */
  more?: More;
  style?: CSSProperties;
};

/** Smallest type scale Introduction shrinks to; anything longer is cut at the foot of the page. */
const MIN_FIT = 0.5;

/**
 * Sets `--fit` on the page to the largest type scale (MIN_FIT..1) at which its text fits without scrolling,
 * again whenever the page is resized or the web fonts arrive.
 */
function useFitText(ref: RefObject<HTMLElement | null>, deps: unknown[]) {
  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    const fits = (scale: number) => {
      el.style.setProperty("--fit", String(scale));
      return el.scrollHeight <= el.clientHeight + 1;
    };
    const fit = () => {
      if (el.clientHeight === 0 || fits(1)) return;
      let lo = MIN_FIT, hi = 1;
      for (let i = 0; i < 7; i++) {
        const mid = (lo + hi) / 2;
        if (fits(mid)) lo = mid;
        else hi = mid;
      }
      fits(lo);
    };
    fit();
    let width = el.clientWidth, height = el.clientHeight;
    const ro = new ResizeObserver(() => {
      if (el.clientWidth === width && el.clientHeight === height) return;
      width = el.clientWidth;
      height = el.clientHeight;
      fit();
    });
    ro.observe(el);
    let live = true;
    document.fonts?.ready.then(() => live && fit());
    return () => {
      live = false;
      ro.disconnect();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);
}

/** The introduction set on a page: running head, heading, text (or a title page), folio. The type shrinks to fit the page. */
function Introduction({ book, titleId, more }: { book: OpenableBook; titleId?: string; more?: More }) {
  const scrollRef = useRef<HTMLDivElement>(null);
  useFitText(scrollRef, [book, Boolean(more)]);
  const paragraphs = book.description?.split(/\n+/).map((p) => p.trim()).filter(Boolean) ?? [];
  const facts = [book.condition, book.language, book.pages ? `${book.pages} pages` : undefined, book.subjectName].filter(Boolean);

  return (
    <div ref={scrollRef} className="bookopen__scroll">
      <p className="bookopen__running small-upper">{book.title}</p>
      <h2 id={titleId} className="bookopen__heading">
        Introduction
      </h2>
      <span className="bookopen__fleuron" aria-hidden="true">
        ❦
      </span>

      {paragraphs.length > 0 ? (
        <div className="bookopen__text">
          {paragraphs.map((p, i) => (
            <p key={i}>{p}</p>
          ))}
        </div>
      ) : (
        // No introduction written yet: set the title page from what the catalogue knows.
        <div className="bookopen__titlepage">
          <p className="bookopen__title">{book.title}</p>
          <p className="bookopen__by">by {book.author}</p>
          <p className="bookopen__imprint small-upper">{[book.publisher, book.year].filter(Boolean).join(" · ")}</p>
          {book.edition !== book.publisher && <p className="bookopen__imprint small-upper">{book.edition}</p>}
          {facts.length > 0 && <p className="bookopen__facts">{facts.join(" — ")}</p>}
          <p className="bookopen__note">The keeper has not yet written an introduction for this copy.</p>
        </div>
      )}

      <footer className="bookopen__foot">
        {more ? (
          <Link href={more.href} onClick={more.onClick} className="bookopen__more small-upper under-hover">
            View this copy →
          </Link>
        ) : (
          <span />
        )}
        <span className="bookopen__folio" aria-hidden="true">
          — i —
        </span>
      </footer>
    </div>
  );
}

/**
 * A book seen from the front, Western binding. Fills its parent, which has the closed book's size and
 * proportions (315:433). Closed, it is the front board (`BookCover`), lying on the right-hand page of the
 * open-book photo (`BookPlate`), spine on its gutter. Opening swings the board right-to-left round that
 * gutter: the right-hand page shows as it lifts, its inside shows once past upright, and laid flat over
 * the left-hand page (hidden until then) it dissolves into it. See `.bookopen__cover` in globals.css.
 *
 * One component for every open book (catalogue preview, the book in flight, the book page), so the
 * same markup and CSS draw each of them and one can stand in for another without a visible change.
 */
export function BookSpread({ book, open, closing = false, hush = false, titleId, more, style }: SpreadProps) {
  const state = [open && "is-open", closing && "is-closing", hush && "is-hushed"].filter(Boolean).join(" ");

  return (
    <div className={`bookopen__spread ${state}`} style={style}>
      <BookPlate book={book} titleId={titleId} more={more} />

      <div className="bookopen__cover" aria-hidden="true">
        <div className="bookopen__front">
          <BookCover title={book.title} author={book.author} color={book.color} eager />
        </div>
        {/* The board's inside is the photo's own left-hand page, placed where it lies once the board is over */}
        <div className="bookopen__back">
          <Image src={openBookPhoto(book)} alt="" width={PHOTO.w} height={PHOTO.h} sizes="(min-width: 1024px) 46vw, 100vw" loading="eager" />
        </div>
      </div>
    </div>
  );
}

/** Size of the open-book photos (public/images/open-book-<variant>.png); all variants share one layout. */
const PHOTO = { w: 1426, h: 1103 };
const openBookPhoto = (book: OpenableBook) => `/images/open-book-${bookVariant(book.color)}.png`;
/** Bounding box of the blank right-hand page in the photo, in photo pixels (`.bookplate__page` in globals.css). */
const PAGE = { x: 724, y: 109, w: 602, h: 896 };
/**
 * The page's paper corners inside PAGE, as fractions (top-left, top-right, bottom-right, bottom-left).
 * The camera sits a little left of the spine, so the outer edge reads taller and the top edge climbs.
 */
const PAPER: [number, number][] = [
  [0, 31 / PAGE.h],
  [1, 0],
  [1, 889 / PAGE.h],
  [0, 1],
];

/** CSS matrix3d mapping a w x h box (transform-origin 0 0) onto PAPER, so the text lies in the page's perspective. */
function paperMatrix(w: number, h: number): string {
  const [[x0, y0], [x1, y1], [x2, y2], [x3, y3]] = PAPER.map(([u, v]) => [u * w, v * h]);
  const dx1 = x1 - x2, dx2 = x3 - x2, dx3 = x0 - x1 + x2 - x3;
  const dy1 = y1 - y2, dy2 = y3 - y2, dy3 = y0 - y1 + y2 - y3;
  const den = dx1 * dy2 - dx2 * dy1;
  const g = (dx3 * dy2 - dx2 * dy3) / den;
  const k = (dx1 * dy3 - dx3 * dy1) / den;
  const a = x1 - x0 + g * x1, b = x3 - x0 + k * x3;
  const d = y1 - y0 + g * y1, e = y3 - y0 + k * y3;
  return `matrix3d(${a / w},${d / w},0,${g / w},${b / h},${e / h},0,${k / h},0,0,1,0,${x0},${y0},0,1)`;
}

/**
 * The open book: a photograph of an open volume in the book's leather colour, its left page left blank
 * and the introduction set on the right-hand page, warped to that page's perspective. 2.18x the closed
 * book's width, its right-hand page under the closed book. Drawn as two halves cut at the gutter, so the
 * left half can wait for the board to turn over.
 */
function BookPlate({ book, titleId, more }: { book: OpenableBook; titleId?: string; more?: More }) {
  const pageRef = useRef<HTMLElement>(null);

  useLayoutEffect(() => {
    const el = pageRef.current;
    if (!el) return;
    const fit = () => {
      if (el.offsetWidth > 0) el.style.transform = paperMatrix(el.offsetWidth, el.offsetHeight);
    };
    fit();
    const ro = new ResizeObserver(fit);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  return (
    <div className="bookplate">
      {(["left", "right"] as const).map((side) => (
        <div key={side} className={`bookplate__side bookplate__side--${side}`}>
          <Image
            src={openBookPhoto(book)}
            alt=""
            aria-hidden="true"
            width={PHOTO.w}
            height={PHOTO.h}
            sizes="(min-width: 1024px) 46vw, 100vw"
            loading="eager"
            fetchPriority="high"
            className="bookplate__photo"
          />
        </div>
      ))}
      <article ref={pageRef} className="bookplate__page">
        <Introduction book={book} titleId={titleId} more={more} />
      </article>
    </div>
  );
}
