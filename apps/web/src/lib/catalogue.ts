// Catalogue client for catalog-service, reached through the API gateway.
// Server-side only: CATALOG_API_URL is read on the server and never sent to the browser.

export type Book = {
  slug: string;
  title: string;
  author: string;
  year: string;
  /** VND, integer */
  price: number;
  edition: string;
  /** Binding, e.g. Hardcover */
  condition: string;
  /** Cover tint, chosen from the slug: the API stores no colour */
  color: string;
  subject: string;
  subjectName: string;
  rare: boolean;
  /** Real cover scan, when we have one */
  image?: string;
};

export type Subject = { slug: string; name: string; bookCount: number };

export const SORTS = [
  { key: "new", name: "Newly shelved" },
  { key: "price-asc", name: "Price, low to high" },
  { key: "price-desc", name: "Price, high to low" },
  { key: "year", name: "Oldest first" },
] as const;

export type SortKey = (typeof SORTS)[number]["key"];

const BASE = (process.env.CATALOG_API_URL ?? "http://localhost:8080/api/catalog").replace(/\/$/, "");

/** The slugs the API accepts in one /books?slugs= request. */
const MAX_BATCH = 50;

/** Tints the book photo variants know about (see BookArt). */
const COVER_COLORS = ["#7a2620", "#5c1a16", "#3f110e", "#93352b", "#2c4436", "#24382e", "#1a2a22", "#3c1d0e", "#64331e", "#8e613c"];

type ApiBook = {
  slug: string;
  title: string;
  author: string | null;
  year: number;
  publisher: string | null;
  edition: string | null;
  binding: string | null;
  priceVnd: number;
  coverUrl: string | null;
  rare: boolean;
  subjectSlug: string;
  subjectName: string;
};

type ApiPage = { items: ApiBook[]; total: number; page: number; pageSize: number };

function colorFor(slug: string): string {
  let hash = 0;
  for (const ch of slug) hash = (hash * 31 + ch.charCodeAt(0)) >>> 0;
  return COVER_COLORS[hash % COVER_COLORS.length];
}

function toBook(b: ApiBook): Book {
  return {
    slug: b.slug,
    title: b.title,
    author: b.author ?? "Unknown author",
    year: String(b.year),
    price: b.priceVnd,
    edition: b.edition ?? b.publisher ?? "Edition not recorded",
    condition: b.binding ?? "Binding not recorded",
    color: colorFor(b.slug),
    subject: b.subjectSlug,
    subjectName: b.subjectName,
    rare: b.rare,
    image: b.coverUrl ?? undefined,
  };
}

/** GET a catalogue API path. A 404 is "not found" (undefined); any other failure throws. */
async function get<T>(path: string, params?: URLSearchParams): Promise<T | undefined> {
  const query = params?.toString();
  const res = await fetch(`${BASE}${path}${query ? `?${query}` : ""}`);
  if (res.status === 404) return undefined;
  if (!res.ok) throw new Error(`Catalogue API answered ${res.status} for ${path}`);
  return (await res.json()) as T;
}

export type CatalogueQuery = {
  subject?: string;
  rare?: boolean;
  sort?: SortKey;
  /** Free text matched against title and author, ignoring case and diacritics */
  q?: string;
  page?: number;
};

export type CataloguePage = { books: Book[]; total: number; page: number; pageSize: number };

export async function queryBooks({ subject, rare, sort, q, page }: CatalogueQuery): Promise<CataloguePage> {
  const params = new URLSearchParams();
  if (subject) params.set("subject", subject);
  if (rare) params.set("rare", "true");
  if (sort) params.set("sort", sort);
  if (q) params.set("q", q);
  if (page && page > 1) params.set("page", String(page));

  const data = await get<ApiPage>("/books", params);
  if (!data) throw new Error("Catalogue API has no /books route");
  return { books: data.items.map(toBook), total: data.total, page: data.page, pageSize: data.pageSize };
}

export async function getBook(slug: string): Promise<Book | undefined> {
  const data = await get<ApiBook>(`/books/${encodeURIComponent(slug)}`);
  return data && toBook(data);
}

/** Several books with one request, in the order asked. Unknown slugs are left out. */
export async function getBooks(slugs: string[]): Promise<Book[]> {
  const unique = [...new Set(slugs)].slice(0, MAX_BATCH);
  if (unique.length === 0) return [];

  const data = await get<ApiPage>("/books", new URLSearchParams({ slugs: unique.join(",") }));
  return (data?.items ?? []).map(toBook);
}

/** Up to `limit` other books: same subject first, then the rest of the shelf. */
export async function relatedBooks(slug: string, limit = 4): Promise<Book[]> {
  const data = await get<{ items: ApiBook[] }>(`/books/${encodeURIComponent(slug)}/related`, new URLSearchParams({ limit: String(limit) }));
  return (data?.items ?? []).map(toBook);
}

export async function getSubjects(): Promise<Subject[]> {
  const data = await get<{ items: Subject[] }>("/subjects");
  return data?.items ?? [];
}
