import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { MotionController } from "@/components/motion/MotionController";
import { BookCover, BookPhoto } from "@/components/ui/BookArt";
import { DotButton } from "@/components/ui/DotButton";
import { BOOKS, SUBJECTS, getBook, relatedBooks } from "@/lib/catalogue";

const price = new Intl.NumberFormat("en-US", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

export function generateStaticParams() {
  return BOOKS.map((b) => ({ slug: b.slug }));
}

export async function generateMetadata({ params }: PageProps<"/books/[slug]">): Promise<Metadata> {
  const book = getBook((await params).slug);
  if (!book) return { title: "Book not found — Marginalleya" };
  return {
    title: `${book.title} — Marginalleya`,
    description: `${book.title} by ${book.author}, ${book.year}. ${book.edition}, ${book.condition}.`,
  };
}

export default async function BookPage({ params }: PageProps<"/books/[slug]">) {
  const book = getBook((await params).slug);
  if (!book) notFound();

  const subject = SUBJECTS.find((s) => s.slug === book.subject);
  const related = relatedBooks(book);
  const specs = [
    { term: "Author", value: book.author },
    { term: "Year", value: book.year },
    { term: "Edition", value: book.edition },
    { term: "Binding", value: book.condition },
    { term: "Subject", value: subject?.name ?? "" },
    { term: "Kind", value: book.rare ? "Rare" : "Used" },
  ];

  return (
    <>
      <MotionController />

      {/* ---------- Hero: cover + essentials ---------- */}
      <section className="work book" aria-labelledby="book-title">
        <nav aria-label="Breadcrumb" className="small-upper mb-[clamp(28px,4vw,56px)] flex flex-wrap items-center gap-x-3 gap-y-1 text-parchment-200">
          <Link href="/catalogue" className="under-hover">
            Catalogue
          </Link>
          <span aria-hidden="true" className="text-gold-400">
            ❧
          </span>
          <Link href={`/catalogue?subject=${book.subject}`} className="under-hover">
            {subject?.name}
          </Link>
          <span aria-hidden="true" className="text-gold-400">
            ❧
          </span>
          <span aria-current="page" className="text-gold-300">
            {book.title}
          </span>
        </nav>

        <div className="book__grid">
          <div className="book__cover" data-reveal="fade">
            <div className="book__cover-inner">
              <BookCover title={book.title} author={book.author} color={book.color} eager />
            </div>
          </div>

          <div className="book__info">
            <span className="work-card__client">{book.author}</span>
            <h1 id="book-title" data-reveal className="big-sans m-0 mt-6 text-[clamp(2rem,4.6vw,5rem)]">
              <span className="line">
                <span>{book.title}</span>
              </span>
              <span className="line">
                <span className="em-italic text-gold-400">{book.year}</span>
              </span>
            </h1>

            <div className="work-card__tags mt-6">
              <span>+ {book.edition}</span>
              <span>+ {book.condition}</span>
              {book.rare && <span>+ Rare</span>}
            </div>

            <p className="m-0 mt-8 font-serif text-[clamp(2rem,3.4vw,3.4rem)] tabular-nums text-parchment-100">{price.format(book.price)}</p>
            <p className="small-upper m-0 mt-1 text-gold-400">One copy · ships from the seller</p>

            <div className="mt-10 flex flex-wrap items-center gap-x-10 gap-y-6">
              <DotButton href="/cart" variant="solid">
                Add to basket
              </DotButton>
              <DotButton href="/wishlist">Save for later</DotButton>
            </div>

            <p className="m-0 mt-12 max-w-[44ch] text-[clamp(17px,1.4vw,21px)] leading-relaxed text-parchment-200">
              Described by the person who keeps it: faults first, virtues after. Every mark, stain and rebinding is noted before we get to the beauty.
            </p>
          </div>
        </div>
      </section>

      {/* ---------- Particulars: cream block ---------- */}
      <section className="cream px-[var(--gutter)] py-[clamp(70px,9vw,140px)]" aria-labelledby="particulars-title">
        <div className="grid gap-12 lg:grid-cols-[1fr_1.5fr]">
          <div>
            <h2 id="particulars-title" data-reveal className="big-sans m-0 text-[clamp(1.75rem,3vw,3.25rem)]">
              <span className="line">
                <span className="em-italic">The</span>
              </span>
              <span className="line">
                <span>particulars</span>
              </span>
            </h2>

            {book.image && (
              <figure data-reveal="fade" className="book__plate">
                {/* Remote scan, so a plain <img> until a domain is configured for next/image */}
                <div className="book__photo">
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img src={book.image} alt={`Photograph of the cover of ${book.title}`} loading="lazy" />
                </div>
                <figcaption>The copy, as photographed</figcaption>
              </figure>
            )}
          </div>

          <dl className="m-0 grid grid-cols-2 gap-x-8 gap-y-8 border-t border-walnut-900/20 pt-8 md:grid-cols-3">
            {specs.map((s, i) => (
              <div key={s.term} data-reveal="fade" style={{ "--delay": `${i * 70}ms` } as React.CSSProperties}>
                <dt className="small-upper text-oxblood-700">{s.term}</dt>
                <dd className="m-0 mt-1 font-serif text-[clamp(18px,1.7vw,24px)] leading-snug">{s.value}</dd>
              </div>
            ))}
          </dl>
        </div>
      </section>

      {/* ---------- More from the shelves ---------- */}
      <section className="work" aria-labelledby="more-title">
        <div className="work__intro">
          <div>
            <span className="small-upper text-gold-400" data-reveal="fade">
              ({String(related.length).padStart(2, "0")})
            </span>
            <h2 id="more-title" data-reveal className="big-sans m-0 mt-3 text-[clamp(2rem,4.2vw,4.6rem)]">
              <span className="line">
                <span>Also on</span>
              </span>
              <span className="line">
                <span className="em-italic text-gold-400">the shelves</span>
              </span>
            </h2>
          </div>
          <DotButton href={`/catalogue?subject=${book.subject}`} className="justify-self-start text-parchment-100">
            See all
          </DotButton>
        </div>

        <ul className="catalogue__grid book__related m-0 list-none p-0">
          {related.map((b) => (
            <li key={b.slug}>
              <Link href={`/books/${b.slug}`} className="work-card" data-reveal="fade">
                <div className="work-card__media">
                  <BookPhoto title={b.title} author={b.author} color={b.color} />
                </div>
                <div className="work-card__content">
                  <h3 className="work-card__title">
                    <span className="work-card__client">{b.author}</span>
                    {b.title}, {b.year}
                  </h3>
                  <p className="m-0 mt-3 font-serif text-xl tabular-nums text-parchment-100">{price.format(b.price)}</p>
                </div>
              </Link>
            </li>
          ))}
        </ul>
      </section>
    </>
  );
}
