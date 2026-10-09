// Placeholder basket until a real one exists (session/DB). Every listing is a single copy.
import { getBooks, queryBooks, type Book } from "@/lib/catalogue";

export type BasketLine = {
  book: Book;
  /** Another reader bought this copy after it was added; it is not charged. */
  sold: boolean;
};

const MOCK: { slug: string; sold?: boolean }[] = [
  { slug: "pride-and-prejudice-b0006aqlwu" },
  { slug: "jamaica-inn-b0000ef83i" },
  { slug: "lost-horizon-b0006dlxji", sold: true },
  { slug: "heidi-b00087e21s" },
];

export async function getBasket(): Promise<BasketLine[]> {
  const books = new Map((await getBooks(MOCK.map((m) => m.slug))).map((b) => [b.slug, b]));
  return MOCK.flatMap(({ slug, sold = false }) => {
    const book = books.get(slug);
    return book ? [{ book, sold }] : [];
  });
}

/** Sum of the copies that can still be bought, VND. */
export function subtotal(lines: BasketLine[]): number {
  return lines.reduce((sum, l) => (l.sold ? sum : sum + l.book.price), 0);
}

/** Stand-in for the recommender: the newest books not already in the basket. */
export async function suggestions(lines: BasketLine[], limit = 4): Promise<Book[]> {
  const inBasket = new Set(lines.map((l) => l.book.slug));
  const { books } = await queryBooks({ sort: "new" });
  return books.filter((b) => !inBasket.has(b.slug)).slice(0, limit);
}
