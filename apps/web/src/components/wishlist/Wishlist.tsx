import Link from "next/link";
import { BookPhoto } from "@/components/ui/BookArt";
import { DotButton } from "@/components/ui/DotButton";
import type { WishlistItem } from "@/lib/wishlist";

const price = new Intl.NumberFormat("en-US", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

/** Saved copies, as catalogue cards. A sold copy stays on the list, marked, so the reader knows what happened to it. */
export function Wishlist({ items }: { items: WishlistItem[] }) {
  if (items.length === 0) {
    return (
      <div className="catalogue__empty">
        <p className="m-0 font-serif text-[clamp(1.6rem,3vw,2.6rem)] uppercase">Nothing saved yet</p>
        <p className="m-0 mt-3 text-parchment-200">Save a copy from its page and it waits here until you decide.</p>
        <DotButton href="/catalogue" className="mt-8">
          Browse the shelves
        </DotButton>
      </div>
    );
  }

  return (
    <ul className="catalogue__grid m-0 list-none p-0">
      {items.map(({ book, sold }) => (
        <li key={book.slug} className={`wish${sold ? " wish--sold" : ""}`}>
          <Link href={`/books/${book.slug}`} className="work-card" data-reveal="fade">
            <div className="work-card__media">
              <BookPhoto title={book.title} author={book.author} color={book.color} />
              {sold && (
                <span className="wish__stamp" aria-hidden="true">
                  Sold
                </span>
              )}
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

          <div className="wish__actions">
            {sold ? (
              <span className="wish__sold-note">Sold to another reader.</span>
            ) : (
              <button type="button" className="wish__add small-upper under-hover" aria-label={`Add ${book.title} to basket`}>
                Add to basket
              </button>
            )}
            <button type="button" className="basket__remove small-upper under-hover" aria-label={`Remove ${book.title} from wishlist`}>
              Remove
            </button>
          </div>
        </li>
      ))}
    </ul>
  );
}
