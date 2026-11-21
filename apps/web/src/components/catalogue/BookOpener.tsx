"use client";

import { useRouter } from "next/navigation";
import { useCallback, useEffect, useId, useRef, useState, type CSSProperties, type MouseEvent, type ReactNode } from "react";
import { createPortal, flushSync } from "react-dom";
import { prefersReducedMotion, takeOff, useBookFlight } from "@/components/motion/BookFlight";
import { BookSpread, type OpenableBook } from "@/components/ui/BookSpread";

export type { OpenableBook };

/**
 * idle -> from (overlay mounted, book drawn over its shelf spot) -> center (flies to the middle)
 * -> open (front cover swings round the spine) -> closing (cover shuts) -> return (flies back) -> idle
 */
type Phase = "idle" | "from" | "center" | "open" | "closing" | "return";
/** An on-screen book: its untransformed box, plus a tilt in degrees round its centre (a hovered shelf copy). */
type Box = { x: number; y: number; w: number; h: number; rot?: number };

/** Flight to and from the shelf (the transition is set inline, not in CSS). */
const FLY_MS = 650;
/** The cover starts to open while the book is still settling. */
const OPEN_AT_MS = 450;
/** Cover closing time: the board swings shut (900ms after 200ms), then the photo fades out; `.bookopen__cover` in globals.css. */
const CLOSE_MS = 1200;
/** Cross-fade used instead of every movement when the reader prefers reduced motion. */
const FADE_MS = 300;

/** Book photo proportions (see Book.tsx). */
const RATIO = 433 / 315;
/** The open-book photo (`.bookplate`) is 2.18x the closed book's width and 1.69x its width tall. */
const OPEN_WIDTH = 2.18;
const OPEN_HEIGHT = 1.69;
/** The closed book lies on the photo's right-hand page: its left edge is this far (x its width) right of the photo's centre. */
const PAGE_OFFSET = 0.017;

const lockScroll = (on: boolean) => window.dispatchEvent(new CustomEvent("of:scroll-lock", { detail: on }));

/** The closed book's place: on the right-hand page of the open book, which is centred in the viewport and fits it. */
function stage(): { to: Box } {
  const vw = document.documentElement.clientWidth;
  const vh = window.innerHeight;
  const w = Math.min(400, (vw - 32) / OPEN_WIDTH, (vh - 112) / OPEN_HEIGHT);
  const h = w * RATIO;
  return { to: { x: vw / 2 + PAGE_OFFSET * w, y: Math.max(56, (vh - h) / 2), w, h } };
}

/**
 * Where the visible book inside a card (BookPhoto's centred wrapper) is on screen. Hovered, the shelf
 * copy is scaled and tilted: the box is its untilted size round the same centre, plus the tilt.
 */
function measure(card: HTMLElement | null): Box | null {
  const el = card?.querySelector<HTMLElement>(".work-card__media > * > *");
  if (!el || el.offsetWidth === 0) return null;
  const r = el.getBoundingClientRect();
  const t = el.parentElement ? getComputedStyle(el.parentElement).transform : "none";
  const m = new DOMMatrix(t === "none" ? undefined : t);
  const rad = Math.atan2(m.b, m.a);
  const aspect = el.offsetHeight / el.offsetWidth;
  // The bounding box of a w x h box turned by rad is (w cos + h sin) wide.
  const w = r.width / (Math.abs(Math.cos(rad)) + aspect * Math.abs(Math.sin(rad)));
  const h = w * aspect;
  return { x: r.left + (r.width - w) / 2, y: r.top + (r.height - h) / 2, w, h, rot: (rad * 180) / Math.PI };
}

/**
 * Wraps a catalogue card. On a touch screen, tapping the book lifts it off the shelf to the middle of the
 * screen and opens its front cover on the introduction; the shelf layout never changes because the open
 * book lives in a fixed overlay. Hovering with a mouse does nothing.
 *
 * Clicking the card (or the open book, or its "View this copy" link) hands the book to BookFlight, which
 * flies it onto the book page, opening on the way, while the page loads underneath. With reduced motion
 * the link simply navigates.
 */
export function BookOpener({ book, children }: { book: OpenableBook; children: ReactNode }) {
  const wrap = useRef<HTMLDivElement>(null);
  const closeBtn = useRef<HTMLButtonElement>(null);
  const bookEl = useRef<HTMLDivElement>(null);
  const [phase, setPhase] = useState<Phase>("idle");
  const phaseRef = useRef<Phase>("idle");
  /** Set on open: where the book flies from and to, and whether the reader prefers reduced motion. */
  const [geo, setGeo] = useState<{ from: Box; to: Box; reduced: boolean } | null>(null);
  const pointerType = useRef("mouse");
  const titleId = useId();
  const router = useRouter();
  const flight = useBookFlight();
  /** Its double is flying to the book page: the shelf copy stays hidden until the catalogue is gone. */
  const flying = flight !== null && flight.book.slug === book.slug;

  const go = (p: Phase) => {
    phaseRef.current = p;
    setPhase(p);
  };

  const open = useCallback(() => {
    if (phaseRef.current !== "idle") return;
    const from = measure(wrap.current);
    if (!from) return;
    lockScroll(true);
    // Mount the overlay with the book drawn over its shelf spot, read layout so that spot becomes the
    // transition's starting point, then let it fly.
    flushSync(() => {
      setGeo({ from, ...stage(), reduced: window.matchMedia("(prefers-reduced-motion: reduce)").matches });
      go("from");
    });
    bookEl.current?.getBoundingClientRect();
    go("center");
  }, []);

  const close = useCallback(() => {
    const p = phaseRef.current;
    if (p === "idle" || p === "closing" || p === "return") return;
    go("closing");
  }, []);

  /** Flies the book to its page: from the shelf, or from the preview when it is out. */
  const fly = (e: MouseEvent) => {
    if (prefersReducedMotion()) return; // the link navigates as usual
    const p = phaseRef.current;
    const href = `/books/${book.slug}`;
    if (p === "idle") {
      const from = measure(wrap.current);
      if (!from) return;
      e.preventDefault();
      takeOff({ book, from, opened: false, veiled: false });
    } else if ((p === "center" || p === "open") && geo && bookEl.current) {
      e.preventDefault();
      const r = bookEl.current.getBoundingClientRect();
      takeOff({
        book,
        from: { x: r.left, y: r.top, w: r.width, h: r.height },
        opened: p === "open",
        veiled: true,
      });
      // The flying copy is drawn in the same render, exactly over the preview: drop the preview at once.
      go("idle");
      setGeo(null);
      lockScroll(false);
    } else {
      e.preventDefault(); // closing or flying back: ignore
      return;
    }
    router.push(href);
  };

  /** A click on the open book itself (not a text selection, not a modified click on its link). */
  const onBookClick = (e: MouseEvent<HTMLDivElement>) => {
    if (e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
    const link = (e.target as Element).closest("a");
    if (!link && window.getSelection()?.isCollapsed === false) return;
    fly(e);
  };

  // Drive the timeline: each phase schedules the next.
  useEffect(() => {
    const fast = geo?.reduced ?? false;
    let timer = 0;
    if (phase === "center") {
      timer = window.setTimeout(() => go("open"), fast ? 0 : OPEN_AT_MS);
    } else if (phase === "open") {
      closeBtn.current?.focus({ preventScroll: true });
    } else if (phase === "closing") {
      timer = window.setTimeout(() => {
        // The shelf may have moved (resize); fly back to where the book is now.
        const from = measure(wrap.current);
        if (from) setGeo((g) => (g ? { ...g, from } : g));
        go("return");
      }, fast ? FADE_MS : CLOSE_MS);
    } else if (phase === "return") {
      timer = window.setTimeout(() => {
        const hadFocus = document.activeElement?.closest(".bookopen");
        go("idle");
        setGeo(null);
        lockScroll(false);
        if (hadFocus) wrap.current?.querySelector<HTMLElement>("a")?.focus({ preventScroll: true });
      }, fast ? FADE_MS : FLY_MS);
    }
    return () => window.clearTimeout(timer);
    // Only a phase change moves the timeline on; `geo` is read for the reduced-motion flag, set before it starts.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [phase]);

  // Escape closes; the page scroll comes back if the card unmounts while open.
  useEffect(() => {
    if (phase === "idle") return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && close();
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [phase, close]);
  useEffect(
    () => () => {
      if (phaseRef.current !== "idle") lockScroll(false);
    },
    [],
  );

  // Clicking the card flies the book to its page. On touch screens tapping the book opens the preview
  // instead, tapping the title flies. Modified clicks (new tab) keep the link's own behaviour.
  const onClickCapture = (e: MouseEvent<HTMLDivElement>) => {
    if (e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
    if (phaseRef.current !== "idle" || !(e.target as Element).closest("a")) return;
    // detail 0: a click from the keyboard (Enter)
    const tap = pointerType.current !== "mouse" && e.detail !== 0;
    if (tap && (e.target as Element).closest(".work-card__media")) {
      e.preventDefault();
      open();
      return;
    }
    fly(e);
  };

  const away = phase === "from" || phase === "return";
  let bookStyle: CSSProperties | undefined;
  if (geo) {
    const { from, to, reduced } = geo;
    // Round the book's centre (the element's origin is its top-left corner), tilt included.
    const k = from.w / to.w;
    const offShelf =
      `translate(${from.x + from.w / 2 - to.x}px, ${from.y + from.h / 2 - to.y}px) rotate(${from.rot ?? 0}deg) ` +
      `scale(${k}) translate(${-to.w / 2}px, ${-to.h / 2}px)`;
    bookStyle = {
      left: to.x,
      top: to.y,
      width: to.w,
      height: to.h,
      transform: away && !reduced ? offShelf : "none",
      opacity: away && reduced ? 0 : 1,
      transition:
        phase === "from" ? "none" : `transform ${FLY_MS}ms cubic-bezier(0.22, 1, 0.36, 1), opacity ${FADE_MS}ms ease`,
    };
  }

  const overlay = geo && (
    <div className={`bookopen is-${phase}`}>
      <div className="bookopen__backdrop" onClick={close} aria-hidden="true" />
      <button ref={closeBtn} type="button" className="bookopen__close small-upper" onClick={close}>
        Close <span aria-hidden="true">✕</span>
      </button>

      <div ref={bookEl} role="dialog" aria-modal="true" aria-labelledby={titleId} className="bookopen__book" style={bookStyle} onClick={onBookClick}>
        <BookSpread
          book={book}
          open={phase === "open"}
          closing={phase === "closing"}
          titleId={titleId}
          more={{ href: `/books/${book.slug}` }}
        />
      </div>
    </div>
  );

  return (
    <div
      ref={wrap}
      className={`bookopener ${phase === "idle" && !flying ? "" : "is-away"}`}
      onPointerDown={(e) => (pointerType.current = e.pointerType)}
      onClickCapture={onClickCapture}
    >
      {children}
      {overlay && createPortal(overlay, document.body)}
    </div>
  );
}
