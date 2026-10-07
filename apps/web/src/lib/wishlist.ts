// Placeholder wishlist until a real one exists (session/DB). Every listing is a single copy.
import { getBook, type Book } from "@/lib/catalogue";

export type WishlistItem = {
  book: Book;
  /** Another reader bought this copy after it was saved; it can no longer be bought. */
  sold: boolean;
};

const MOCK: { slug: string; sold?: boolean }[] = [
  { slug: "murder-at-the-vicarage" },
  { slug: "a-tree-grows-in-brooklyn" },
  { slug: "officially-dead", sold: true },
  { slug: "the-complete-sherlock-holmes" },
  { slug: "the-story-of-philosophy" },
];

export function getWishlist(): WishlistItem[] {
  return MOCK.flatMap(({ slug, sold = false }) => {
    const book = getBook(slug);
    return book ? [{ book, sold }] : [];
  });
}
