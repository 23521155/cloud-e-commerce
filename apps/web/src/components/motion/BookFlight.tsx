"use client";

import { usePathname } from "next/navigation";
import { useEffect, useLayoutEffect, useRef, useState, useSyncExternalStore, type CSSProperties } from "react";
import { BookSpread, type OpenableBook } from "@/components/ui/BookSpread";

/**
 * Catalogue -> book page as one continuous movement (FLIP across a route change).
 *
 * The catalogue hands the clicked book to this module (`takeOff`) and starts the navigation. The host,
 * mounted in the shop layout (which stays mounted while the page under it changes), draws the book in a
 * fixed overlay at the shelf spot (tilt included) and flies it closed with a spring, swinging the front
 * board open once most of the way is behind it.
 * The book page renders a `BookStand` (the same open spread) hidden, which reports its element (`arrive`);
 * the spring re-aims at that element and, once the book sits exactly on it fully open, `land` removes the
 * overlay and shows the stand in the same render. Both draw `BookSpread` open at the same box, so the
 * hand-over shows no change: the book never closes again.
 */

/** An on-screen book: its untransformed box, plus a tilt in degrees round its centre (a hovered shelf copy). */
type Box = { x: number; y: number; w: number; h: number; rot?: number };

export type Flight = {
  id: number;
  /** Drawn in flight; swapped for the book page's own copy when it arrives, so the text matches the page. */
  book: OpenableBook;
  /** Where the book is on screen at take-off (closed: its shelf spot; open: the catalogue preview). */
  from: Box;
  /** The board is already open at take-off (taken from the catalogue preview). */
  opened: boolean;
  /** The preview's backdrop was showing: start the veil opaque so nothing flashes. */
  veiled: boolean;
  origin: string;
  /** The book page's stand, once it has rendered. */
  target: HTMLElement | null;
};

/** Book photo proportions (see Book.tsx). */
const RATIO = 433 / 315;
/** The board starts to swing once the flight is about three quarters done (the spring at OMEGA 9). */
const OPEN_DELAY_MS = 300;
/** Board swing (900ms) and its inside's dissolve into the left page (300ms after 750ms); `.bookopen__cover` in globals.css. */
const OPEN_MS = 1050;
/** Introduction fade once the board is open and the page has arrived; `.bookopen__scroll` (600ms + 500ms delay). */
const TEXT_MS = 1100;
/** Spring stiffness (critically damped): lands in about a second. */
const OMEGA = 9;
/** Give up when the book page has not rendered by then (navigation failed or was left). */
const ABANDON_MS = 10_000;
const FADE_MS = 300;

let flight: Flight | null = null;
let nextId = 1;
/** Where the last flight landed, per viewport size: the best first aim before the page has rendered. */
let lastLanding: { vw: number; vh: number; box: Box } | null = null;
const listeners = new Set<() => void>();

function publish(next: Flight | null) {
  flight = next;
  listeners.forEach((l) => l());
}
const subscribe = (l: () => void) => {
  listeners.add(l);
  return () => listeners.delete(l);
};

export const prefersReducedMotion = () => window.matchMedia("(prefers-reduced-motion: reduce)").matches;

/** Starts a flight from the catalogue. The caller navigates to `/books/{slug}` right after. */
export function takeOff(f: Pick<Flight, "book" | "from" | "opened" | "veiled">) {
  publish({ ...f, id: nextId++, origin: window.location.pathname, target: null });
}

function arrive(id: number, target: HTMLElement, book: OpenableBook) {
  if (flight?.id === id && flight.target !== target) publish({ ...flight, target, book });
}

function clear(id: number) {
  if (flight?.id === id) publish(null);
}

export function useBookFlight(): Flight | null {
  return useSyncExternalStore(subscribe, () => flight, () => null);
}

/** The first aim: last landing spot, else roughly where the book page puts its book (`.bookstand`: left column, upper half). */
function predict(): Box {
  const vw = document.documentElement.clientWidth;
  const vh = window.innerHeight;
  if (lastLanding && lastLanding.vw === vw && lastLanding.vh === vh) return lastLanding.box;
  const wide = vw >= 1024;
  const col = wide ? vw * 0.44 : vw - 40;
  const w = Math.max(135, Math.min(327, col / 2.18, (vh - 240) * 0.7));
  const h = w * RATIO;
  return { x: (wide ? 40 : 20) + (col - w) / 2 + 0.517 * w, y: Math.min(Math.max(140, (vh - h) / 2), vh - h - 20), w, h };
}

function safeDecode(s: string) {
  try {
    return decodeURIComponent(s);
  } catch {
    return s;
  }
}

function rectOf(el: HTMLElement): Box {
  const r = el.getBoundingClientRect();
  return { x: r.left, y: r.top, w: r.width, h: r.height };
}

/** Lives in the shop layout: draws the book in flight, if any. */
export function BookFlightHost() {
  const f = useBookFlight();
  return f ? <FlightOverlay key={f.id} f={f} /> : null;
}

function FlightOverlay({ f }: { f: Flight }) {
  const { id } = f;
  const pathname = usePathname();
  const rootRef = useRef<HTMLDivElement>(null);
  const bookRef = useRef<HTMLDivElement>(null);
  /** The book's layout size; the flight is drawn by translate + scale relative to it. */
  const [size, setSize] = useState(() => {
    const p = predict();
    return { w: p.w, h: p.h };
  });
  const sizeRef = useRef(size);
  const pos = useRef({ x: f.from.x, y: f.from.y, w: f.from.w, r: f.from.rot ?? 0, vx: 0, vy: 0, vw: 0, vr: 0 });
  const [open, setOpen] = useState(f.opened);
  const openAt = useRef(f.opened ? -Infinity : Infinity);
  const textAt = useRef(Infinity);
  const [veil, setVeil] = useState(f.veiled ? 1 : 0);
  const [timedOut, setTimedOut] = useState(false);
  // Abandon the flight (fade out) when the navigation went somewhere else, or the page never came.
  const dest = `/books/${f.book.slug}`;
  const same = (a: string, b: string) => a === b || safeDecode(a) === safeDecode(b);
  const leaving = timedOut || (!same(pathname, f.origin) && !same(pathname, dest));

  const apply = () => {
    const el = bookRef.current;
    if (!el) return;
    const p = pos.current;
    const { w, h } = sizeRef.current;
    const k = p.w / w;
    // Tilted round the book's centre (the element's origin is its top-left corner).
    el.style.transform = `translate(${p.x + (w * k) / 2}px, ${p.y + (h * k) / 2}px) rotate(${p.r}deg) scale(${k}) translate(${-w / 2}px, ${-h / 2}px)`;
  };

  useEffect(() => {
    if (!leaving) return;
    const t = window.setTimeout(() => clear(id), FADE_MS);
    return () => window.clearTimeout(t);
  }, [leaving, id]);

  // A new layout size (the page's real stand): re-draw at once with the same on-screen box.
  useLayoutEffect(() => {
    sizeRef.current = size;
    apply();
  }, [size]);

  // Take-off: the veil darkens the catalogue; the board swings open as the book nears the page.
  useEffect(() => {
    const raf = requestAnimationFrame(() => setVeil((v) => Math.max(v, 0.9)));
    const t = f.opened
      ? 0
      : window.setTimeout(() => {
          openAt.current = performance.now();
          setOpen(true);
        }, OPEN_DELAY_MS);
    return () => {
      cancelAnimationFrame(raf);
      window.clearTimeout(t);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // The page cannot scroll while the book is in the air (Lenis skips [data-lenis-prevent]).
  useEffect(() => {
    const root = rootRef.current;
    if (!root) return;
    const stop = (e: Event) => e.preventDefault();
    root.addEventListener("wheel", stop, { passive: false });
    root.addEventListener("touchmove", stop, { passive: false });
    return () => {
      root.removeEventListener("wheel", stop);
      root.removeEventListener("touchmove", stop);
    };
  }, []);

  // The flight: a critically damped spring towards the stand (or the predicted spot until it exists).
  useLayoutEffect(() => {
    const started = performance.now();
    const guess = predict();
    let last = started;
    let raf = 0;
    apply();

    const tick = (now: number) => {
      const cur = flight;
      if (!cur || cur.id !== id) return;
      const target = cur.target?.isConnected ? rectOf(cur.target) : guess;

      if (cur.target && target.w > 0 && Math.abs(target.w - sizeRef.current.w) > 0.5) {
        setSize({ w: target.w, h: target.h });
      }
      // The introduction starts to fade in once the board is open and the page's copy has arrived.
      if (cur.target && openAt.current !== Infinity && textAt.current === Infinity) textAt.current = now;

      const p = pos.current;
      // Sub-stepped, so a slow frame (or a throttled tab) still advances in real time, up to 100ms at once.
      let dt = Math.min(0.1, (now - last) / 1000);
      last = now;
      while (dt > 0) {
        const h = Math.min(dt, 1 / 120);
        dt -= h;
        p.vx += (OMEGA * OMEGA * (target.x - p.x) - 2 * OMEGA * p.vx) * h;
        p.vy += (OMEGA * OMEGA * (target.y - p.y) - 2 * OMEGA * p.vy) * h;
        p.vw += (OMEGA * OMEGA * (target.w - p.w) - 2 * OMEGA * p.vw) * h;
        p.vr += (OMEGA * OMEGA * -p.r - 2 * OMEGA * p.vr) * h;
        p.x += p.vx * h;
        p.y += p.vy * h;
        p.w += p.vw * h;
        p.r += p.vr * h;
      }
      const settled =
        Math.abs(target.x - p.x) < 0.75 &&
        Math.abs(target.y - p.y) < 0.75 &&
        Math.abs(target.w - p.w) < 0.75 &&
        Math.abs(p.r) < 0.05 &&
        Math.hypot(p.vx, p.vy, p.vw) < 20;

      if (cur.target && settled && now - openAt.current >= OPEN_MS && now - textAt.current >= TEXT_MS) {
        // On the stand, fully open, text shown: snap the last fraction of a pixel and hand over.
        Object.assign(p, { x: target.x, y: target.y, w: target.w, r: 0, vx: 0, vy: 0, vw: 0, vr: 0 });
        apply();
        lastLanding = { vw: document.documentElement.clientWidth, vh: window.innerHeight, box: target };
        clear(id);
        return;
      }
      apply();
      if (!cur.target && now - started > ABANDON_MS) return setTimedOut(true);
      raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const bookStyle: CSSProperties = { left: 0, top: 0, width: size.w, height: size.h };

  return (
    // Covers the screen until the hand-over, so nothing can be clicked mid-flight.
    <div ref={rootRef} className={`bookflight ${leaving ? "is-leaving" : ""}`} data-lenis-prevent aria-hidden="true">
      {/* Once the page has rendered under it, the veil lifts and the book glides onto the page */}
      <div className="bookflight__veil" style={{ opacity: f.target ? 0 : veil }} />
      <div ref={bookRef} className="bookopen__book" style={bookStyle}>
        <BookSpread book={f.book} open={open} hush={!f.target} />
      </div>
    </div>
  );
}

/**
 * The open book on the book page. Arriving from the catalogue, it stays hidden while the flying copy
 * lands on it, then shows in the same render that removes the overlay. Opened directly, it is simply open.
 */
export function BookStand({ book }: { book: OpenableBook }) {
  const ref = useRef<HTMLDivElement>(null);
  const f = useBookFlight();
  const incoming = f !== null && f.book.slug === book.slug;
  // Decided once: a stand that arrived by air must not start the scroll-reveal fade after landing.
  const [byAir] = useState(() => flight !== null && flight.book.slug === book.slug);
  const flightId = incoming ? f.id : 0;

  useEffect(() => {
    if (flightId && ref.current) arrive(flightId, ref.current, book);
  }, [flightId, book]);

  return (
    <div ref={ref} className="bookstand" data-reveal={byAir ? undefined : "fade"} style={incoming ? { visibility: "hidden" } : undefined}>
      <BookSpread book={book} open />
    </div>
  );
}
