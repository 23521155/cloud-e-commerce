// In-memory stand-in for the catalogue API, used when MOCK_CATALOGUE=true (a UI-only deploy with no
// backend). Answers the same paths with the same JSON shapes as catalog-service.
import type { ApiBook } from "@/lib/catalogue";

const PAGE_SIZE = 12;

const SUBJECTS = [
  { slug: "literature", name: "Literature" },
  { slug: "history", name: "History" },
  { slug: "philosophy", name: "Philosophy" },
  { slug: "science", name: "Science" },
  { slug: "poetry", name: "Poetry" },
];

type Seed = [slug: string, title: string, author: string, year: number, priceVnd: number, subject: string, rare?: boolean, status?: ApiBook["status"]];

// Newest shelved first, which is the "new" sort order.
const SEEDS: Seed[] = [
  ["pride-and-prejudice-1894", "Pride and Prejudice", "Jane Austen", 1894, 1_250_000, "literature", true],
  ["moby-dick-1930", "Moby-Dick", "Herman Melville", 1930, 890_000, "literature"],
  ["the-origin-of-species-1872", "On the Origin of Species", "Charles Darwin", 1872, 4_800_000, "science", true],
  ["meditations-1945", "Meditations", "Marcus Aurelius", 1945, 320_000, "philosophy"],
  ["leaves-of-grass-1900", "Leaves of Grass", "Walt Whitman", 1900, 2_100_000, "poetry", true, "RESERVED"],
  ["the-histories-1920", "The Histories", "Herodotus", 1920, 560_000, "history"],
  ["great-expectations-1907", "Great Expectations", "Charles Dickens", 1907, 740_000, "literature"],
  ["a-brief-history-of-time-1988", "A Brief History of Time", "Stephen Hawking", 1988, 280_000, "science"],
  ["beyond-good-and-evil-1923", "Beyond Good and Evil", "Friedrich Nietzsche", 1923, 450_000, "philosophy", false, "SOLD"],
  ["the-waste-land-1922", "The Waste Land", "T. S. Eliot", 1922, 6_500_000, "poetry", true],
  ["the-decline-and-fall-1910", "The Decline and Fall of the Roman Empire", "Edward Gibbon", 1910, 1_900_000, "history", true],
  ["jane-eyre-1897", "Jane Eyre", "Charlotte Brontë", 1897, 980_000, "literature"],
  ["principia-mathematica-1934", "Principia Mathematica", "Isaac Newton", 1934, 2_600_000, "science", true],
  ["the-republic-1955", "The Republic", "Plato", 1955, 260_000, "philosophy"],
  ["truyen-kieu-1925", "Truyện Kiều", "Nguyễn Du", 1925, 1_450_000, "poetry", true],
  ["the-art-of-war-1963", "The Art of War", "Sun Tzu", 1963, 210_000, "history"],
  ["war-and-peace-1931", "War and Peace", "Leo Tolstoy", 1931, 1_150_000, "literature"],
  ["the-double-helix-1968", "The Double Helix", "James D. Watson", 1968, 390_000, "science"],
  ["walden-1910", "Walden", "Henry David Thoreau", 1910, 670_000, "philosophy"],
  ["the-raven-and-other-poems-1884", "The Raven and Other Poems", "Edgar Allan Poe", 1884, 3_200_000, "poetry", true],
  ["vietnam-a-history-1983", "Vietnam: A History", "Stanley Karnow", 1983, 430_000, "history"],
  ["crime-and-punishment-1951", "Crime and Punishment", "Fyodor Dostoevsky", 1951, 360_000, "literature"],
  ["cosmos-1980", "Cosmos", "Carl Sagan", 1980, 310_000, "science"],
  ["thus-spoke-zarathustra-1933", "Thus Spoke Zarathustra", "Friedrich Nietzsche", 1933, 520_000, "philosophy"],
];

const BOOKS: ApiBook[] = SEEDS.map(([slug, title, author, year, priceVnd, subject, rare = false, status = "AVAILABLE"]) => ({
  slug,
  title,
  author,
  year,
  publisher: null,
  edition: "First edition",
  binding: "Hardcover",
  priceVnd,
  coverUrl: null,
  rare,
  status,
  subjectSlug: subject,
  subjectName: SUBJECTS.find((s) => s.slug === subject)!.name,
}));

function fold(text: string): string {
  return text.normalize("NFD").replace(/\p{Diacritic}/gu, "").replace(/đ/gi, "d").toLowerCase();
}

function listBooks(params: URLSearchParams) {
  const slugs = params.get("slugs");
  if (slugs) {
    const wanted = slugs.split(",");
    const items = wanted.map((s) => BOOKS.find((b) => b.slug === s)).filter((b): b is ApiBook => !!b);
    return { items, total: items.length, page: 1, pageSize: items.length };
  }

  const subject = params.get("subject");
  const q = params.get("q");
  let items = BOOKS.filter(
    (b) =>
      (!subject || b.subjectSlug === subject) &&
      (params.get("rare") !== "true" || b.rare) &&
      (!q || fold(`${b.title} ${b.author}`).includes(fold(q))),
  );

  const sort = params.get("sort");
  if (sort === "price-asc") items = [...items].sort((a, b) => a.priceVnd - b.priceVnd);
  if (sort === "price-desc") items = [...items].sort((a, b) => b.priceVnd - a.priceVnd);
  if (sort === "year") items = [...items].sort((a, b) => a.year - b.year);

  const page = Math.max(1, Number(params.get("page")) || 1);
  return { items: items.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE), total: items.length, page, pageSize: PAGE_SIZE };
}

/** The mock answer for a catalogue API path, or undefined where the API would answer 404. */
export function mockCatalogue(path: string, params = new URLSearchParams()): unknown {
  if (path === "/subjects") {
    return { items: SUBJECTS.map((s) => ({ ...s, bookCount: BOOKS.filter((b) => b.subjectSlug === s.slug).length })) };
  }
  if (path === "/books") return listBooks(params);

  const [, , rawSlug, rest] = path.split("/");
  const book = BOOKS.find((b) => b.slug === decodeURIComponent(rawSlug ?? ""));
  if (!book) return undefined;
  if (!rest) return book;
  if (rest === "related") {
    const limit = Number(params.get("limit")) || 4;
    const others = BOOKS.filter((b) => b.slug !== book.slug);
    const items = [...others.filter((b) => b.subjectSlug === book.subjectSlug), ...others.filter((b) => b.subjectSlug !== book.subjectSlug)];
    return { items: items.slice(0, limit) };
  }
  return undefined;
}
