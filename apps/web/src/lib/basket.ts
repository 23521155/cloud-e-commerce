// Placeholder basket until a real one exists (session/DB). Every listing is a single copy.
import { BOOKS, getBook, type Book } from "@/lib/catalogue";

export type BasketLine = {
  book: Book;
  /** Another reader bought this copy after it was added; it is not charged. */
  sold: boolean;
};

const MOCK: { slug: string; sold?: boolean }[] = [
  { slug: "pride-and-prejudice" },
  { slug: "jamaica-inn" },
  { slug: "lost-horizon", sold: true },
  { slug: "meditations" },
];

export function getBasket(): BasketLine[] {
  return MOCK.flatMap(({ slug, sold = false }) => {
    const book = getBook(slug);
    return book ? [{ book, sold }] : [];
  });
}

/** Sum of the copies that can still be bought, VND. */
export function subtotal(lines: BasketLine[]): number {
  return lines.reduce((sum, l) => (l.sold ? sum : sum + l.book.price), 0);
}

/** Stand-in for the recommender: books not already in the basket. */
export function suggestions(lines: BasketLine[], limit = 4): Book[] {
  const inBasket = new Set(lines.map((l) => l.book.slug));
  return BOOKS.filter((b) => !inBasket.has(b.slug)).slice(0, limit);
}
