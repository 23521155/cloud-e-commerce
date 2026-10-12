// Shared basket types and helpers, for server and browser alike. The guest basket itself lives in
// the browser's localStorage (see basket-store.ts). Every listing is a single copy.
import type { Book } from "@/lib/catalogue";

/** Most copies a basket holds: the same limit as one order in the order service. */
export const MAX_BASKET = 20;

export type BasketLine = {
  book: Book;
  /** Another reader bought this copy after it was added; it is not charged. */
  sold: boolean;
};

/** Sum of the copies that can still be bought, VND. */
export function subtotal(lines: BasketLine[]): number {
  return lines.reduce((sum, l) => (l.sold ? sum : sum + l.book.price), 0);
}
