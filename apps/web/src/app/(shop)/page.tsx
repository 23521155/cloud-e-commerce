import Image from "next/image";
import Link from "next/link";
import type { CSSProperties } from "react";
import { MotionController } from "@/components/motion/MotionController";
import { Preloader } from "@/components/motion/Preloader";
import { BookCover, BookPhoto, OpenBook, SpineStack } from "@/components/ui/BookArt";
import { DotButton } from "@/components/ui/DotButton";

// Placeholder catalogue until the Prisma-backed products exist.
type Book = {
  slug: string;
  title: string;
  author: string;
  year: string;
  /** VND, integer */
  price: number;
  edition: string;
  condition: string;
  color: string;
};

// Sample rows from ref/antique_books.jsonl (Amazon Books metadata); USD prices × 25,000 → VND.
const NEW_BOOKS: Book[] = [
  { slug: "the-little-prince", title: "The Little Prince", author: "Antoine de Saint-Exupéry", year: "1943", price: 2760000, edition: "Reynal & Hitchcock", condition: "Hardcover", color: "#5c1a16" },
  { slug: "murder-at-the-vicarage", title: "Murder at the Vicarage", author: "Agatha Christie", year: "1948", price: 5000000, edition: "First edition", condition: "Paperback", color: "#24382e" },
  { slug: "the-razors-edge", title: "The Razor's Edge", author: "W. Somerset Maugham", year: "1944", price: 290000, edition: "First edition", condition: "Hardcover", color: "#3c1d0e" },
  { slug: "a-tree-grows-in-brooklyn", title: "A Tree Grows in Brooklyn", author: "Betty Smith", year: "1943", price: 800000, edition: "Harper & Brothers", condition: "Hardcover", color: "#3f110e" },
  { slug: "jamaica-inn", title: "Jamaica Inn", author: "Daphne du Maurier", year: "1936", price: 220000, edition: "First edition", condition: "Hardcover", color: "#64331e" },
  { slug: "pavilion-of-women", title: "Pavilion of Women", author: "Pearl S. Buck", year: "1946", price: 200000, edition: "The John Day Company", condition: "Hardcover", color: "#1a2a22" },
];

const RARE = [
  { title: "Pride and Prejudice", author: "Jane Austen", color: "#24382e" },
  { title: "The Complete Sherlock Holmes", author: "Arthur Conan Doyle", color: "#5c1a16" },
  { title: "Heidi", author: "Johanna Spyri", color: "#3c1d0e" },
  { title: "Lost Horizon", author: "James Hilton", color: "#7a2620" },
  { title: "Mythology", author: "Edith Hamilton", color: "#1a2a22" },
  { title: "Hiroshima", author: "John Hersey", color: "#64331e" },
];

const RARE_KINDS = [
  { name: "First editions", note: "The work's very first printing" },
  { name: "Signed", note: "By the author or translator" },
  { name: "Indochina", note: "Imprints from before 1954" },
  { name: "Engravings", note: "Plate books, maps, hand-coloured" },
];

const SUBJECT_ROWS = [
  { word: "Literature", slug: "literature", before: true, after: true },
  { word: "History", slug: "history", before: false, after: true },
  { word: "Philosophy", slug: "philosophy", before: true, after: false },
  { word: "Children's", slug: "children", before: true, after: true },
];

const BUBBLES = ["Tang poetry", "French novels", "Chronicles", "Buddhism", "Old maps", "Folk tales", "Vintage magazines", "Old schoolbooks"];

/** Scattered hero art: position, size, rotation, parallax strength (negative speed = floats up faster than the scroll); `desk` items are hidden on phones. */
const HERO_ART = [
  { kind: "cover", book: 0, top: "22%", left: "30%", w: "clamp(150px,17vw,320px)", rot: -5, mouse: 0.3, speed: -0.1, desk: true },
  { kind: "stack", seed: 3, top: "14%", left: "82%", w: "clamp(90px,10vw,190px)", rot: 0, mouse: 1.4, speed: -0.35, desk: false },
  { kind: "cover", book: 3, top: "46%", left: "5%", w: "clamp(80px,10vw,190px)", rot: 6, mouse: -1.2, speed: -0.45, desk: true },
  { kind: "cover", book: 1, top: "40%", left: "80%", w: "clamp(100px,13vw,250px)", rot: 7, mouse: 1, speed: -0.25, desk: true },
  { kind: "stack", seed: 9, top: "87%", left: "40%", w: "clamp(70px,7vw,140px)", rot: -4, mouse: -1.6, speed: -0.5, desk: true },
  { kind: "open", top: "10%", left: "4%", w: "clamp(110px,14vw,260px)", rot: -6, mouse: 0.9, speed: -0.3, desk: false },
  { kind: "cover", book: 4, top: "80%", left: "74%", w: "clamp(70px,7vw,140px)", rot: -9, mouse: -0.8, speed: -0.6, desk: false },
] as const;

const price = new Intl.NumberFormat("en-US", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

function Letters({ word }: { word: string }) {
  return (
    <span aria-hidden="true">
      {Array.from(word).map((ch, i) =>
        ch === " " ? (
          " "
        ) : (
          <span key={i} className="svc__letter" style={{ "--i": i } as CSSProperties}>
            {ch}
          </span>
        ),
      )}
    </span>
  );
}

function SubjectArt({ index }: { index: number }) {
  const book = RARE[index % RARE.length];
  return (
    <div className="svc__img">
      <BookPhoto title={book.title} author={book.author} color={book.color} />
    </div>
  );
}

export default function Home() {
  return (
    <>
      <Preloader />
      <MotionController />

      {/* ---------- Hero ---------- */}
      <section className="hero" aria-labelledby="hero-title">
        <div className="hero__art" aria-hidden="true">
          {HERO_ART.map((item, i) => (
            <div
              key={i}
              className={`hero__item ${item.desk ? "max-md:hidden" : ""}`}
              data-mouse={item.mouse}
              data-speed={item.speed}
              style={{ top: item.top, left: item.left, width: item.w }}
            >
              <div style={{ rotate: `${item.rot}deg` }}>
                {item.kind === "cover" && (
                  <BookCover
                    plain={i === 0}
                    eager
                    title={NEW_BOOKS[item.book].title}
                    author={NEW_BOOKS[item.book].author}
                    color={NEW_BOOKS[item.book].color}
                  />
                )}
                {item.kind === "stack" && <SpineStack className="w-full" />}
                {item.kind === "open" && <OpenBook className="w-full" />}
              </div>
            </div>
          ))}
        </div>

        <div className="hero__center">
          <h1 id="hero-title" className="hero__title" aria-label="Every book has a story behind it">
            <span className="hero__line" aria-hidden="true">
              <span>
                Every book<em>*</em>
              </span>
            </span>
            <span className="hero__line" aria-hidden="true">
              <span>Has a</span>
            </span>
            <span className="hero__line hero__line--accent" aria-hidden="true">
              <span>story</span>
            </span>
            <span className="hero__line" aria-hidden="true">
              <span>Behind it</span>
            </span>
          </h1>
          <div className="hero__bottom">
            <div>
              <span className="small-upper text-gold-400">Bookshop</span>
              <span className="serif-line">Used · Rare · Antiquarian</span>
            </div>
            <div>
              <span className="small-upper text-gold-400">Buy &amp; sell</span>
              <span className="serif-line">From the people who keep them</span>
            </div>
          </div>
        </div>
      </section>

      {/* ---------- Reel: shelf wall scales up while sticky ---------- */}
      <section className="reel" data-progress="sticky" aria-label="Bookshelf">
        <div className="reel__sticky">
          <div className="reel__frame">
            <Image
              src="/images/bookshelf-wall.png"
              alt="A wooden bookcase filled with old leather-bound books"
              fill
              sizes="100vw"
              className="object-cover"
            />
            <div className="reel__label">
              <DotButton href="/catalogue" variant="solid">
                Browse the shelves
              </DotButton>
            </div>
          </div>
        </div>
      </section>

      {/* ---------- New arrivals: staggered tall cards ---------- */}
      <section className="work" aria-labelledby="new-title">
        <div className="work__intro">
          <div>
            <span className="small-upper text-gold-400" data-reveal="fade">
              ({String(NEW_BOOKS.length).padStart(2, "0")})
            </span>
            <h2 id="new-title" data-reveal className="big-sans m-0 mt-3 text-[clamp(2rem,4.2vw,4.6rem)]">
              <span className="line">
                <span>Newly</span>
              </span>
              <span className="line">
                <span className="em-italic text-gold-400">shelved</span>
              </span>
            </h2>
          </div>
          <DotButton href="/catalogue?sort=new" className="justify-self-start text-parchment-100">
            See all
          </DotButton>
        </div>

        <ul className="work__grid m-0 list-none p-0">
          {NEW_BOOKS.map((book, i) => (
            <li key={book.slug}>
              <div data-speed={[0.12, 0.06, 0][i % 3]}>
                <Link href={`/books/${book.slug}`} className="work-card" data-reveal="fade">
                  <div className="work-card__media">
                    <BookPhoto title={book.title} author={book.author} color={book.color} />
                  </div>
                  <div className="work-card__content">
                    <h3 className="work-card__title">
                      <span className="work-card__client">{book.author}</span>
                      {book.title}, {book.year}
                    </h3>
                    <div className="work-card__tags">
                      <span>+ {book.edition}</span>
                      <span>+ {book.condition}</span>
                    </div>
                    <p className="m-0 mt-3 font-serif text-xl tabular-nums text-parchment-100">{price.format(book.price)}</p>
                  </div>
                </Link>
              </div>
            </li>
          ))}
        </ul>
      </section>

      {/* ---------- Rare: cream culture block ---------- */}
      <section className="cream overflow-hidden" aria-labelledby="rare-title" data-progress="pass">
        <div className="px-[var(--gutter)] pt-[clamp(90px,12vw,180px)]">
          <div className="grid items-center gap-10 lg:grid-cols-[1.6fr_1fr]">
            <div>
              <span className="small-upper text-oxblood-700">Behind the counter</span>
              <h2 id="rare-title" data-reveal className="culture__heading big-sans mt-4">
                <span className="line">
                  <span>Keeping the past</span>
                </span>
                <span className="line">
                  <span className="em-serif">present on</span>
                </span>
                <span className="line">
                  <span>
                    every <span className="em-italic text-oxblood-700">page</span>
                  </span>
                </span>
              </h2>
            </div>
            <div data-speed="0.2">
              <OpenBook className="mx-auto w-full max-w-[460px]" />
            </div>
          </div>

          <div className="mt-[clamp(60px,8vw,130px)] grid items-center gap-12 lg:grid-cols-2">
            <div>
              <span aria-hidden="true" className="text-2xl text-oxblood-700">
                ❦
              </span>
              <p className="m-0 mt-4 max-w-[40ch] text-[clamp(18px,1.5vw,22px)] leading-relaxed">
                First editions, signed copies, hand-coloured plate books and Indochina imprints. Each one is collated leaf by leaf and photographed exactly as it is.
              </p>
              <dl className="m-0 mt-10 grid grid-cols-2 gap-x-8 gap-y-8 border-t border-walnut-900/20 pt-8">
                {RARE_KINDS.map((kind, i) => (
                  <div key={kind.name} data-reveal="fade" style={{ "--delay": `${i * 80}ms` } as CSSProperties}>
                    <dt className="font-serif text-[clamp(20px,2vw,28px)] uppercase leading-tight">{kind.name}</dt>
                    <dd className="m-0 mt-1 text-[15px] text-ink-600">{kind.note}</dd>
                  </div>
                ))}
              </dl>
              <DotButton href="/catalogue?rare=1" className="mt-10">
                View rare books
              </DotButton>
            </div>
            <div data-speed="0.15">
              <div className="aspect-[4/5] overflow-hidden rounded-[10px]">
                <BookPhoto title={RARE[0].title} author={RARE[0].author} color={RARE[0].color} />
              </div>
            </div>
          </div>
        </div>

        <div className="mt-[clamp(70px,9vw,140px)] overflow-hidden" aria-hidden="true">
          <div className="rail rail--loop px-[var(--gutter)]">
            {[...RARE, ...RARE].map((book, i) => (
              <div key={i} className="rail__item aspect-[4/5] overflow-hidden rounded-[10px]">
                <BookPhoto title={book.title} author={book.author} color={book.color} tone="light" />
              </div>
            ))}
          </div>
        </div>

        <div className="mx-[var(--gutter)] mt-[clamp(70px,9vw,140px)] grid gap-12 border-t border-walnut-900/20 py-[clamp(70px,9vw,140px)] lg:grid-cols-[1fr_1.5fr]">
          <div>
            <h3 data-reveal className="big-sans m-0 text-[clamp(1.75rem,3vw,3.25rem)]">
              <span className="line">
                <span className="em-italic">How we</span>
              </span>
              <span className="line">
                <span>describe books</span>
              </span>
            </h3>
            <p className="m-0 mt-5 text-lg">Faults first, virtues after.</p>
          </div>
          <div className="grid gap-10 md:grid-cols-[1fr_1.4fr]">
            <div className="flex flex-col gap-8">
              <div>
                <span className="small-upper">Condition first</span>
                <p className="m-0 mt-2 text-[15px] leading-relaxed text-ink-600">
                  Every fox mark, damp stain and rebinding is noted before we get to the beauty.
                </p>
              </div>
              <div>
                <span className="small-upper">A little patience</span>
                <p className="m-0 mt-2 text-[15px] leading-relaxed text-ink-600">
                  Old books take time to find the right reader. We are in no hurry.
                </p>
              </div>
            </div>
            <p className="m-0 text-[clamp(18px,1.55vw,23px)] leading-relaxed">
              Marginalleya is an online bookshop for used, new and rare books. Our sellers are the people who already keep them: second-hand shops, collectors, or a family shelf that has run out of room. Every book is described by the person holding it, so you know exactly what you are getting <em>before it arrives</em>.
            </p>
          </div>
        </div>
      </section>

      {/* ---------- Subjects: giant words with images ---------- */}
      <section className="cream px-[var(--gutter)] pb-[clamp(70px,9vw,140px)]" aria-labelledby="subjects-title">
        <h2 id="subjects-title" data-reveal className="big-sans m-0 mb-[clamp(40px,6vw,90px)] text-center text-[clamp(1.9rem,3.8vw,4.4rem)]">
          <span className="line">
            <span className="em-serif">You are looking</span>
          </span>
          <span className="line">
            <span>
              for books <span className="em-italic">on</span>
            </span>
          </span>
        </h2>

        <div className="flex flex-col items-center gap-[clamp(10px,2vw,28px)]">
          {SUBJECT_ROWS.map((row, r) => (
            <div key={row.slug} className="contents">
              <div data-reveal className="flex items-center justify-center gap-[clamp(12px,2.4vw,40px)]">
                {row.before && <SubjectArt index={r * 2} />}
                <Link
                  href={`/catalogue?subject=${row.slug}`}
                  aria-label={row.word}
                  className="svc__word"
                >
                  <Letters word={row.word} />
                </Link>
                {row.after && <SubjectArt index={r * 2 + 1} />}
              </div>
              {r === 0 && (
                <div data-reveal className="my-[clamp(12px,2vw,30px)] flex max-w-3xl flex-wrap justify-center gap-3">
                  {BUBBLES.map((bubble, i) => (
                    <Link
                      key={bubble}
                      href={`/catalogue?q=${encodeURIComponent(bubble)}`}
                      className="bubble"
                      style={{ "--i": i, "--r": `${((i * 37) % 11) - 5}deg` } as CSSProperties}
                    >
                      {bubble}
                    </Link>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>

        <div className="mt-[clamp(50px,7vw,110px)] flex justify-between gap-6">
          <div>
            <span className="small-upper text-oxblood-700">Made with</span>
            <span className="serif-line">Patience</span>
          </div>
          <div className="text-right">
            <span className="small-upper text-oxblood-700">Buy &amp; sell</span>
            <span className="serif-line">Used &amp; rare books</span>
          </div>
        </div>
      </section>
    </>
  );
}
