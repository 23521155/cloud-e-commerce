"use client";

import { useEffect } from "react";

/** Survives client-side navigations, so opening the page afresh does not count as a page change. */
let last: { filters: string; page: number } | null = null;

/**
 * Glides to the first row of books when only the page number changed (the pager, not a new
 * search or filter, which keep the reader where they are). MotionController owns the smooth
 * scroll; without it (reduced motion) the page jumps there instead.
 */
export function ScrollToShelf({ filters, page }: { filters: string; page: number }) {
  useEffect(() => {
    const paged = last !== null && last.filters === filters && last.page !== page;
    last = { filters, page };
    if (!paged) return;

    // Wait a frame: MotionController is remounted on the same navigation (and, in development,
    // StrictMode destroys and recreates it once more), so ask the Lenis instance that stays.
    const frame = requestAnimationFrame(() => {
      const handled = !window.dispatchEvent(new CustomEvent("of:scroll-to", { detail: "#shelf", cancelable: true }));
      if (!handled) document.getElementById("shelf")?.scrollIntoView();
    });
    return () => cancelAnimationFrame(frame);
  }, [filters, page]);

  return null;
}
