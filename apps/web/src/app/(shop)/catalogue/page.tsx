import type { Metadata } from "next";
import Form from "next/form";
import Link from "next/link";
import { ScrollToShelf } from "@/components/catalogue/ScrollToShelf";
import { MotionController } from "@/components/motion/MotionController";
import { BookPhoto } from "@/components/ui/BookArt";
import { DotButton } from "@/components/ui/DotButton";
import { SORTS, getSubjects, queryBooks, type SortKey } from "@/lib/catalogue";

export const metadata: Metadata = {
  title: "Catalogue — Marginalleya",
  description: "Browse used, new and rare books by subject, each described by the person who keeps it.",
};

const price = new Intl.NumberFormat("en-US", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

const first = (v: string | string[] | undefined) => (Array.isArray(v) ? v[0] : v);

/** Subjects with fewer books than this are left out of the filter bar (a subject picked by link still shows). */
const MIN_SUBJECT_BOOKS = 20;

/** Page numbers to show: first, last and the neighbours of the current page, with null marking a gap. */
function pageWindow(current: number, total: number): (number | null)[] {
  const keep = new Set([1, total, current - 1, current, current + 1].filter((n) => n >= 1 && n <= total));
  const out: (number | null)[] = [];
  let prev = 0;
  for (const n of [...keep].sort((a, b) => a - b)) {
    if (n - prev > 1) out.push(null);
    out.push(n);
    prev = n;
  }
  return out;
}

export default async function CataloguePage({ searchParams }: PageProps<"/catalogue">) {
  const sp = await searchParams;
  // An unknown subject is not an error: the API just finds nothing on that shelf.
  const subjectParam = first(sp.subject) ?? "";
  const subject = /^[a-z0-9-]{1,100}$/.test(subjectParam) ? subjectParam : undefined;
  const rare = first(sp.rare) === "1";
  const sort = (SORTS.find((s) => s.key === first(sp.sort))?.key ?? "new") as SortKey;

  const q = (first(sp.q) ?? "").trim().slice(0, 80);
  const wantedPage = Math.max(1, Math.floor(Number(first(sp.page))) || 1);

  const [subjects, result] = await Promise.all([getSubjects(), queryBooks({ subject, rare, sort, q, page: wantedPage })]);
  const total = result.total;
  const totalPages = Math.max(1, Math.ceil(total / (result.pageSize || 1)));
  const page = Math.min(totalPages, wantedPage);
  // A page past the end shows the last one
  const books = page === wantedPage ? result.books : (await queryBooks({ subject, rare, sort, q, page })).books;
  const shownSubjects = subjects.filter((s) => s.slug === subject || (s.slug !== "uncategorized" && s.bookCount >= MIN_SUBJECT_BOOKS));

  /** Build a catalogue URL from the current filters with some of them overridden. Changing a filter resets to page 1. */
  const href = (next: { subject?: string | null; rare?: boolean; sort?: string; page?: number; q?: string }) => {
    const params = new URLSearchParams();
    const text = next.q === undefined ? q : next.q;
    if (text) params.set("q", text);
    const s = next.subject === undefined ? subject : next.subject;
    const r = next.rare === undefined ? rare : next.rare;
    const o = next.sort ?? sort;
    if (s) params.set("subject", s);
    if (r) params.set("rare", "1");
    if (o !== "new") params.set("sort", o);
    if (next.page && next.page > 1) params.set("page", String(next.page));
    const qs = params.toString();
    return qs ? `/catalogue?${qs}` : "/catalogue";
  };

  return (
    <>
      {/* New filters render new cards; remount so the reveal observer picks them up (it only scans on mount). */}
      <MotionController key={`${subject}-${rare}-${sort}-${page}-${q}`} />
      {/* After MotionController, so its scroll listener is already attached when this fires */}
      <ScrollToShelf filters={`${subject}-${rare}-${sort}-${q}`} page={page} />

      <section className="work catalogue" aria-labelledby="catalogue-title">
        <div className="catalogue__head">
          <div>
            <span className="small-upper text-gold-400" data-reveal="fade">
              ({String(total).padStart(2, "0")})
            </span>
            <h1 id="catalogue-title" data-reveal className="big-sans m-0 mt-3 text-[clamp(2.4rem,6vw,7rem)]">
              <span className="line">
                <span>The</span>
              </span>
              <span className="line">
                <span className="em-italic text-gold-400">shelves</span>
              </span>
            </h1>
          </div>
          <p className="m-0 max-w-[34ch] text-[clamp(17px,1.3vw,21px)] leading-relaxed text-parchment-200">
            Used, new and rare, each one described by the person who keeps it. Faults first, virtues after.
          </p>
        </div>

        <nav className="catalogue__filters" aria-label="Filter the catalogue">
          <ul className="catalogue__row" aria-label="Subject">
            <li>
              <Link href={href({ subject: null })} scroll={false} className={`catalogue__chip ${!subject ? "is-active" : ""}`} aria-current={!subject ? "true" : undefined}>
                All subjects
              </Link>
            </li>
            {shownSubjects.map((s) => (
              <li key={s.slug}>
                <Link href={href({ subject: s.slug })} scroll={false} className={`catalogue__chip ${subject === s.slug ? "is-active" : ""}`} aria-current={subject === s.slug ? "true" : undefined}>
                  {s.name}
                </Link>
              </li>
            ))}
            <li className="catalogue__sep" aria-hidden="true">
              ❦
            </li>
            <li>
              <Link href={href({ rare: !rare })} scroll={false} className={`catalogue__chip ${rare ? "is-active" : ""}`} aria-current={rare ? "true" : undefined}>
                Rare only
              </Link>
            </li>
          </ul>

          <ul className="catalogue__row catalogue__row--sort" aria-label="Sort">
            <li className="small-upper text-gold-400">Sort</li>
            {SORTS.map((o) => (
              <li key={o.key}>
                <Link
                  href={href({ sort: o.key })}
                  scroll={false}
                  className={`small-upper under-hover ${sort === o.key ? "text-gold-300" : "text-parchment-200"}`}
                  aria-current={sort === o.key ? "true" : undefined}
                >
                  {o.name}
                </Link>
              </li>
            ))}
          </ul>
        </nav>

        {/* Search and filters keep the scroll position: the reader stays by the shelf instead of jumping to the top */}
        <Form className="catalogue__search" action="/catalogue" scroll={false} role="search" aria-label="Search the catalogue">
          {subject && <input type="hidden" name="subject" value={subject} />}
          {rare && <input type="hidden" name="rare" value="1" />}
          {sort !== "new" && <input type="hidden" name="sort" value={sort} />}
          <label htmlFor="catalogue-q" className="catalogue__card-head small-upper">
            <span>Card search</span>
            <span>Title or author</span>
          </label>
          <div className="catalogue__card-body">
            <input
              id="catalogue-q"
              key={q}
              type="search"
              name="q"
              defaultValue={q}
              maxLength={80}
              placeholder="Jane Austen…"
              autoComplete="off"
              className="catalogue__input"
            />
            {q && (
              <Link href={href({ q: "" })} scroll={false} className="catalogue__clear small-upper under-hover">
                Clear
              </Link>
            )}
            <button type="submit" className="catalogue__stamp">
              Find
            </button>
          </div>
        </Form>

        {books.length > 0 ? (
          <>
          <ul id="shelf" className="catalogue__grid m-0 list-none scroll-mt-10 p-0">
            {books.map((book) => (
              <li key={book.slug}>
                <Link href={`/books/${book.slug}`} className="work-card" data-reveal="fade">
                  <div className="work-card__media">
                    <BookPhoto title={book.title} author={book.author} color={book.color} />
                  </div>
                  <div className="work-card__content">
                    <h2 className="work-card__title">
                      <span className="work-card__client">{book.author}</span>
                      {book.title}, {book.year}
                    </h2>
                    <div className="work-card__tags">
                      <span>+ {book.edition}</span>
                      <span>+ {book.condition}</span>
                      {book.rare && <span>+ Rare</span>}
                    </div>
                    <p className="m-0 mt-3 font-serif text-xl tabular-nums text-parchment-100">{price.format(book.price)}</p>
                  </div>
                </Link>
              </li>
            ))}
          </ul>

          {totalPages > 1 && (
            <nav className="catalogue__pager" aria-label="Pagination">
              {page > 1 ? (
                <Link href={href({ page: page - 1 })} scroll={false} className="catalogue__step small-upper under-hover" rel="prev">
                  <span aria-hidden="true" className="catalogue__step-mark catalogue__step-mark--prev">❧</span>
                  Previous
                </Link>
              ) : (
                <span className="catalogue__step small-upper is-disabled" aria-disabled="true">
                  <span aria-hidden="true" className="catalogue__step-mark catalogue__step-mark--prev">❧</span>
                  Previous
                </span>
              )}

              <ol className="catalogue__pages">
                {pageWindow(page, totalPages).map((n, i) =>
                  n === null ? (
                    <li key={`gap-${i}`} aria-hidden="true" className="catalogue__gap">
                      ···
                    </li>
                  ) : (
                    <li key={n}>
                      <Link
                        href={href({ page: n })}
                        scroll={false}
                        className={`catalogue__chip catalogue__chip--num ${n === page ? "is-active" : ""}`}
                        aria-label={`Page ${n}`}
                        aria-current={n === page ? "page" : undefined}
                      >
                        {n}
                      </Link>
                    </li>
                  ),
                )}
              </ol>

              {page < totalPages ? (
                <Link href={href({ page: page + 1 })} scroll={false} className="catalogue__step small-upper under-hover" rel="next">
                  Next
                  <span aria-hidden="true" className="catalogue__step-mark">❧</span>
                </Link>
              ) : (
                <span className="catalogue__step small-upper is-disabled" aria-disabled="true">
                  Next
                  <span aria-hidden="true" className="catalogue__step-mark">❧</span>
                </span>
              )}

              <p className="catalogue__count small-upper">
                Page {String(page).padStart(2, "0")} of {String(totalPages).padStart(2, "0")}
              </p>
            </nav>
          )}
          </>
        ) : (
          <div className="catalogue__empty">
            <span aria-hidden="true" className="text-3xl text-gold-400">
              ❦
            </span>
            <p className="m-0 mt-4 font-serif text-[clamp(1.6rem,3vw,2.6rem)] uppercase">Nothing on this shelf yet</p>
            <p className="m-0 mt-3 text-parchment-200">
              {q ? `Nothing matches “${q}”. Check the spelling or try fewer words.` : "Old books take time to find their way here. Try another subject."}
            </p>
            <DotButton href="/catalogue" className="mt-8">
              Clear filters
            </DotButton>
          </div>
        )}
      </section>
    </>
  );
}
