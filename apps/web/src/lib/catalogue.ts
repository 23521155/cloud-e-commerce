// Placeholder catalogue until the Prisma-backed products exist.
export type Book = {
  slug: string;
  title: string;
  author: string;
  year: string;
  /** VND, integer */
  price: number;
  edition: string;
  condition: string;
  color: string;
  subject: SubjectSlug;
  rare: boolean;
};

export const SUBJECTS = [
  { slug: "literature", name: "Literature" },
  { slug: "history", name: "History" },
  { slug: "philosophy", name: "Philosophy" },
  { slug: "children", name: "Children's" },
] as const;

export type SubjectSlug = (typeof SUBJECTS)[number]["slug"];

export const SORTS = [
  { key: "new", name: "Newly shelved" },
  { key: "price-asc", name: "Price, low to high" },
  { key: "price-desc", name: "Price, high to low" },
  { key: "year", name: "Oldest first" },
] as const;

export type SortKey = (typeof SORTS)[number]["key"];

// Listed newest first. Sample rows from ref/antique_books.jsonl; USD prices × 25,000 → VND.
export const BOOKS: Book[] = [
  { slug: "the-little-prince", title: "The Little Prince", author: "Antoine de Saint-Exupéry", year: "1943", price: 2760000, edition: "Reynal & Hitchcock", condition: "Hardcover", color: "#5c1a16", subject: "children", rare: true },
  { slug: "murder-at-the-vicarage", title: "Murder at the Vicarage", author: "Agatha Christie", year: "1948", price: 5000000, edition: "First edition", condition: "Paperback", color: "#24382e", subject: "literature", rare: true },
  { slug: "the-razors-edge", title: "The Razor's Edge", author: "W. Somerset Maugham", year: "1944", price: 290000, edition: "First edition", condition: "Hardcover", color: "#3c1d0e", subject: "literature", rare: false },
  { slug: "a-tree-grows-in-brooklyn", title: "A Tree Grows in Brooklyn", author: "Betty Smith", year: "1943", price: 800000, edition: "Harper & Brothers", condition: "Hardcover", color: "#3f110e", subject: "literature", rare: false },
  { slug: "jamaica-inn", title: "Jamaica Inn", author: "Daphne du Maurier", year: "1936", price: 220000, edition: "First edition", condition: "Hardcover", color: "#64331e", subject: "literature", rare: false },
  { slug: "pavilion-of-women", title: "Pavilion of Women", author: "Pearl S. Buck", year: "1946", price: 200000, edition: "The John Day Company", condition: "Hardcover", color: "#1a2a22", subject: "literature", rare: false },
  { slug: "marigold-garden", title: "Marigold Garden", author: "Kate Greenaway", year: "1885", price: 3250000, edition: "Routledge", condition: "Hardcover", color: "#7a2620", subject: "children", rare: true },
  { slug: "officially-dead", title: "Officially Dead", author: "Quentin Reynolds", year: "1945", price: 500000, edition: "Random House, 1st edition", condition: "Hardcover", color: "#2c4436", subject: "history", rare: false },
  { slug: "pride-and-prejudice", title: "Pride and Prejudice", author: "Jane Austen", year: "1894", price: 4200000, edition: "George Allen", condition: "Hardcover", color: "#24382e", subject: "literature", rare: true },
  { slug: "the-complete-sherlock-holmes", title: "The Complete Sherlock Holmes", author: "Arthur Conan Doyle", year: "1930", price: 1850000, edition: "Doubleday", condition: "Hardcover", color: "#5c1a16", subject: "literature", rare: false },
  { slug: "heidi", title: "Heidi", author: "Johanna Spyri", year: "1922", price: 650000, edition: "Grosset & Dunlap", condition: "Hardcover", color: "#3c1d0e", subject: "children", rare: false },
  { slug: "lost-horizon", title: "Lost Horizon", author: "James Hilton", year: "1933", price: 960000, edition: "First edition", condition: "Hardcover", color: "#93352b", subject: "history", rare: true },
  { slug: "mythology", title: "Mythology", author: "Edith Hamilton", year: "1942", price: 420000, edition: "Little, Brown", condition: "Hardcover", color: "#1a2a22", subject: "philosophy", rare: false },
  { slug: "hiroshima", title: "Hiroshima", author: "John Hersey", year: "1946", price: 380000, edition: "Alfred A. Knopf", condition: "Hardcover", color: "#64331e", subject: "history", rare: false },
  { slug: "meditations", title: "Meditations", author: "Marcus Aurelius", year: "1906", price: 1400000, edition: "J. M. Dent", condition: "Hardcover", color: "#2a140b", subject: "philosophy", rare: true },
  { slug: "the-story-of-philosophy", title: "The Story of Philosophy", author: "Will Durant", year: "1926", price: 540000, edition: "Simon & Schuster", condition: "Hardcover", color: "#3f110e", subject: "philosophy", rare: false },
];

export const PAGE_SIZE = 8;

export type CatalogueQuery = {
  subject?: SubjectSlug;
  rare: boolean;
  sort: SortKey;
  /** Free text matched against title and author, ignoring case and diacritics */
  q?: string;
};

const fold = (s: string) => s.normalize("NFD").replace(/\p{Diacritic}/gu, "").toLowerCase();

export function queryBooks({ subject, rare, sort, q }: CatalogueQuery): Book[] {
  const terms = q ? fold(q).split(/\s+/).filter(Boolean) : [];
  const books = BOOKS.filter((b) => {
    if (subject && b.subject !== subject) return false;
    if (rare && !b.rare) return false;
    const hay = fold(`${b.title} ${b.author}`);
    return terms.every((t) => hay.includes(t));
  });
  if (sort === "price-asc") return [...books].sort((a, b) => a.price - b.price);
  if (sort === "price-desc") return [...books].sort((a, b) => b.price - a.price);
  if (sort === "year") return [...books].sort((a, b) => Number(a.year) - Number(b.year));
  return books;
}
