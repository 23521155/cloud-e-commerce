import Link from "next/link";
import { BookCover } from "@/components/ui/BookArt";
import { DotButton } from "@/components/ui/DotButton";
import { Logo } from "@/components/ui/Logo";
import type { BasketLine } from "@/lib/basket";

const price = new Intl.NumberFormat("en-US", { style: "currency", currency: "VND", maximumFractionDigits: 0 });
const day = new Intl.DateTimeFormat("en-GB", { day: "numeric", month: "short", year: "numeric" });

/** The fan never shows more than this many covers; the rest are counted in a tag. */
const FAN_MAX = 7;
/** The card always draws at least this many ruled lines, like a fresh library card. */
const CARD_ROWS = 6;

const copies = (n: number) => `${n} ${n === 1 ? "copy" : "copies"}`;

/**
 * Spread the covers so the fan keeps the same overall footprint whatever the count:
 * a few books open wide, many books close up.
 */
function fanPose(i: number, n: number) {
  if (n < 2) return { x: 0, r: 0 };
  const spreadDeg = Math.min(9 * (n - 1), 32);
  const spreadX = Math.min(34 * (n - 1), 100);
  const t = i / (n - 1) - 0.5;
  return { x: t * spreadX, r: t * spreadDeg };
}

function CoverFan({ lines }: { lines: BasketLine[] }) {
  const live = lines.filter((l) => !l.sold);
  const shown = live.slice(0, FAN_MAX);
  const hidden = live.length - shown.length;
  return (
    <div className="basket__fan" aria-hidden="true">
      {shown.map(({ book }, i) => {
        const { x, r } = fanPose(i, shown.length);
        return (
          <div key={book.slug} className="basket__fan-item" style={{ "--x": `${x}%`, "--r": `${r}deg` } as React.CSSProperties}>
            <BookCover title={book.title} author={book.author} color={book.color} />
          </div>
        );
      })}
      {hidden > 0 && <span className="basket__fan-more work-card__client">+ {hidden} more</span>}
    </div>
  );
}

/** Rubber seal on the card, lettered round the rim, the shop's mark in the middle in the seal's one ink. */
export function Seal() {
  return (
    <span className="lib-card__seal" aria-hidden="true">
      <svg viewBox="0 0 120 120">
        <defs>
          <path id="seal-rim" d="M60 60 m-44 0 a44 44 0 1 1 88 0 a44 44 0 1 1 -88 0" />
        </defs>
        <circle cx="60" cy="60" r="57" fill="none" stroke="currentColor" strokeWidth="2.5" />
        <circle cx="60" cy="60" r="33" fill="none" stroke="currentColor" strokeWidth="1.2" />
        <text fontSize="10" fill="currentColor">
          <textPath href="#seal-rim" textLength="272" lengthAdjust="spacing">MARGINALLEYA · USED &amp; RARE BOOKS ·</textPath>
        </text>
      </svg>
      <Logo className="lib-card__seal-logo" />
    </span>
  );
}

export function Basket({ lines, total }: { lines: BasketLine[]; total: number }) {
  const live = lines.filter((l) => !l.sold).length;
  const blanks = Math.max(0, CARD_ROWS - lines.length);

  return (
    <div className="basket__layout">
      <CoverFan lines={lines} />

      <div>
        <section className="lib-card" aria-labelledby="card-title">
          <div className="lib-card__frame">
            <h2 id="card-title" className="lib-card__head">
              Basket card
            </h2>

            <div className="lib-card__top">
              <dl className="lib-card__fields">
                <div>
                  <dt>Date</dt>
                  <dd>{day.format(new Date())}</dd>
                </div>
                <div>
                  <dt>Copies</dt>
                  <dd>{copies(live)} to buy</dd>
                </div>
                <div>
                  <dt>Shipping</dt>
                  <dd>Calculated at checkout</dd>
                </div>
              </dl>
              <Seal />
            </div>

            <table className="lib-card__table">
              <caption className="sr-only">Copies in your basket</caption>
              <thead>
                <tr>
                  <th scope="col">No.</th>
                  <th scope="col">Copy</th>
                  <th scope="col" className="lib-card__col-binding">
                    Binding
                  </th>
                  <th scope="col" className="text-right">
                    Price
                  </th>
                </tr>
              </thead>
              <tbody>
                {lines.map(({ book, sold }, i) => (
                  <tr key={book.slug} className={sold ? "is-sold" : ""}>
                    <td className="lib-card__no">{i + 1}.</td>
                    <td>
                      <Link href={`/books/${book.slug}`} className="lib-card__title under-hover">
                        {book.title}
                      </Link>
                      <span className="lib-card__by">
                        {book.author}, {book.year}
                        <span className="lib-card__binding-inline"> · {book.condition}</span>
                      </span>
                      {sold && <span className="basket__sold-note">Sold to another reader. It will not be charged.</span>}
                    </td>
                    <td className="lib-card__col-binding">{book.condition}</td>
                    <td className="lib-card__price">
                      <span className="lib-card__amount">{price.format(book.price)}</span>
                      <button type="button" className="basket__remove small-upper under-hover" aria-label={`Remove ${book.title} from basket`}>
                        Remove
                      </button>
                    </td>
                  </tr>
                ))}
                {Array.from({ length: blanks }, (_, i) => (
                  <tr key={`blank-${i}`} className="lib-card__blank" aria-hidden="true">
                    <td className="lib-card__no">{lines.length + i + 1}.</td>
                    <td />
                    <td className="lib-card__col-binding" />
                    <td />
                  </tr>
                ))}
              </tbody>
            </table>

            <div className="lib-card__foot">
              <span className="lib-card__marks" aria-hidden="true">
                ❦ ❦ ❦
              </span>
              <p className="lib-card__subtotal m-0">
                <span className="small-upper">Subtotal</span>
                <span className="lib-card__total">{price.format(total)}</span>
              </p>
            </div>
          </div>
        </section>

        <div className="basket__cta">
          <DotButton href="/checkout" variant="solid">
            Checkout
          </DotButton>
        </div>
      </div>
    </div>
  );
}
