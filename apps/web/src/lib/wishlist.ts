// Placeholder wishlist until a real one exists (session/DB). Every listing is a single copy.
import { getBooks, type Book } from "@/lib/catalogue";

export type WishlistItem = {
  book: Book;
  /** Another reader bought this copy after it was saved; it can no longer be bought. */
  sold: boolean;
};

const MOCK: { slug: string; sold?: boolean }[] = [
  { slug: "murder-at-the-vicarage-b075k9f5hv" },
  { slug: "a-tree-grows-in-brooklyn-b0006aq1se" },
  { slug: "officially-dead-b000jdt7l6", sold: true },
  { slug: "the-complete-sherlock-holmes-b00005vo0t" },
  { slug: "the-razors-edge-b0007izgh2" },
];

export async function getWishlist(): Promise<WishlistItem[]> {
  const books = new Map((await getBooks(MOCK.map((m) => m.slug))).map((b) => [b.slug, b]));
  return MOCK.flatMap(({ slug, sold = false }) => {
    const book = books.get(slug);
    return book ? [{ book, sold }] : [];
  });
}
