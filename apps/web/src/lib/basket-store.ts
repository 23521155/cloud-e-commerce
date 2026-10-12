"use client";

// Guest basket, kept in this browser's localStorage as a list of book slugs. Every listing is a
// single copy, so a slug is either in the basket or not. Book details and prices are fetched
// fresh (/api/books) wherever the basket is shown; the order service reads prices again anyway.
import { useEffect, useMemo, useState, useSyncExternalStore } from "react";
import { MAX_BASKET, type BasketLine } from "@/lib/basket";
import type { Book } from "@/lib/catalogue";

const KEY = "basket";

/** Fired in this tab on every change; other tabs hear the browser's "storage" event. */
const CHANGE = "basket-change";

function readRaw(): string {
  try {
    return localStorage.getItem(KEY) ?? "[]";
  } catch {
    // Storage blocked (private mode, disabled cookies): an empty basket
    return "[]";
  }
}

function parse(raw: string): string[] {
  try {
    const value: unknown = JSON.parse(raw);
    return Array.isArray(value) ? value.filter((s): s is string => typeof s === "string") : [];
  } catch {
    return [];
  }
}

function write(slugs: string[]) {
  try {
    localStorage.setItem(KEY, JSON.stringify(slugs));
  } catch {
    // Storage full or blocked: the basket simply does not change
  }
  window.dispatchEvent(new Event(CHANGE));
}

function subscribe(onChange: () => void) {
  const onStorage = (e: StorageEvent) => {
    if (e.key === KEY || e.key === null) onChange();
  };
  window.addEventListener(CHANGE, onChange);
  window.addEventListener("storage", onStorage);
  return () => {
    window.removeEventListener(CHANGE, onChange);
    window.removeEventListener("storage", onStorage);
  };
}

/**
 * The slugs in the basket, oldest first. null until the browser has read localStorage: the server
 * render and the first client render both see null, so they match and show the skeleton.
 */
export function useBasket(): string[] | null {
  // The raw string is the snapshot: it compares equal while the basket is unchanged
  const raw = useSyncExternalStore(subscribe, readRaw, () => null);
  return useMemo(() => (raw === null ? null : parse(raw)), [raw]);
}

/**
 * The basket with its books, fetched from /api/books. lines is null while the basket or its
 * books are still loading; failed is true when the books could not be fetched. Books already
 * fetched are kept, so removing one shows at once instead of loading the basket again.
 */
export function useBasketLines(): { lines: BasketLine[] | null; failed: boolean } {
  const slugs = useBasket();
  // Book by slug; null for a slug the catalogue does not know
  const [known, setKnown] = useState<Record<string, Book | null>>({});
  const [failed, setFailed] = useState(false);

  const missing = (slugs ?? []).filter((s) => !(s in known)).join(",");

  useEffect(() => {
    if (!missing) return;
    let live = true;
    fetch(`/api/books?slugs=${encodeURIComponent(missing)}`)
      .then((res) => (res.ok ? (res.json() as Promise<Book[]>) : Promise.reject(new Error(String(res.status)))))
      .then(
        (books) => {
          if (!live) return;
          setKnown((prev) => {
            const next = { ...prev };
            for (const slug of missing.split(",")) next[slug] = null;
            for (const book of books) next[book.slug] = book;
            return next;
          });
          setFailed(false);
        },
        () => live && setFailed(true),
      );
    return () => {
      live = false;
    };
  }, [missing]);

  if (slugs === null || missing) return { lines: null, failed };
  return {
    lines: slugs.flatMap((slug) => {
      const book = known[slug];
      return book ? [{ book, sold: !book.available }] : [];
    }),
    failed: false,
  };
}

export type AddResult = "added" | "already" | "full";

export function addToBasket(slug: string): AddResult {
  const slugs = parse(readRaw());
  if (slugs.includes(slug)) return "already";
  if (slugs.length >= MAX_BASKET) return "full";
  write([...slugs, slug]);
  return "added";
}

export function removeFromBasket(slug: string) {
  write(parse(readRaw()).filter((s) => s !== slug));
}

export function clearBasket() {
  write([]);
}
