"use client";

// Guest basket, kept in this browser's localStorage as a list of book slugs. Every listing is a
// single copy, so a slug is either in the basket or not. Book details and prices are fetched
// fresh (/api/books) wherever the basket is shown; the order service reads prices again anyway.
import { useMemo, useSyncExternalStore } from "react";
import { MAX_BASKET } from "@/lib/basket";

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
