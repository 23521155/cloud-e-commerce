"use client";

import Link from "next/link";
import { BasketSkeleton } from "@/components/basket/BasketSkeleton";
import { Basket } from "@/components/cart/Basket";
import { MotionController } from "@/components/motion/MotionController";
import { BookPhoto } from "@/components/ui/BookArt";
import { DotButton } from "@/components/ui/DotButton";
import { subtotal } from "@/lib/basket";
import { removeFromBasket, useBasket, useBasketLines } from "@/lib/basket-store";
import type { Book } from "@/lib/catalogue";

const price = new Intl.NumberFormat("en-US", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

/** How many suggestions are shown under the basket. */
const SUGGESTIONS = 4;

/**
 * The basket page. The basket lives in this browser, so it is read here; the page only fetches
 * the suggestions. `shelf` holds a few more than are shown, so books already in the basket can be
 * left out.
 */
export function CartView({ shelf }: { shelf: Book[] }) {
  const slugs = useBasket();
  const { lines, failed } = useBasketLines();

  const inBasket = new Set(slugs ?? []);
  const more = shelf.filter((b) => !inBasket.has(b.slug)).slice(0, SUGGESTIONS);

  return (
    <>
      <MotionController />

      <section className="work basket" aria-labelledby="basket-title">
        <div className="catalogue__head">
          <div>
            <span className="small-upper text-gold-400" data-reveal="fade">
              ({slugs ? String(slugs.length).padStart(2, "0") : "··"})
            </span>
            <h1 id="basket-title" data-reveal className="big-sans m-0 mt-3 text-[clamp(2.4rem,6vw,7rem)]">
              <span className="line">
                <span>Your</span>
              </span>
              <span className="line">
                <span className="em-italic text-gold-400">basket</span>
              </span>
            </h1>
          </div>
          <p className="m-0 max-w-[34ch] text-[clamp(17px,1.3vw,21px)] leading-relaxed text-parchment-200">
            Every copy here is the only one we have. It stays on the shelf for others until you check out.
          </p>
        </div>

        {failed ? (
          <div className="catalogue__empty" role="alert">
            <p className="m-0 font-serif text-[clamp(1.6rem,3vw,2.6rem)] uppercase">The shelves did not answer</p>
            <p className="m-0 mt-3 text-parchment-200">Your basket is safe in this browser. Reload the page in a moment.</p>
          </div>
        ) : lines === null ? (
          <BasketSkeleton />
        ) : lines.length === 0 ? (
          <div className="catalogue__empty">
            <span aria-hidden="true" className="text-3xl text-gold-400">
              ❦
            </span>
            <p className="m-0 mt-4 font-serif text-[clamp(1.6rem,3vw,2.6rem)] uppercase">Your basket is empty</p>
            <p className="m-0 mt-3 text-parchment-200">Nothing set aside yet. The shelves are a good place to start.</p>
            <DotButton href="/catalogue" className="mt-8">
              Browse the shelves
            </DotButton>
          </div>
        ) : (
          <Basket lines={lines} total={subtotal(lines)} onRemove={removeFromBasket} />
        )}
      </section>

      {/* ---------- Suggestions (from the recommender later) ---------- */}
      <section className="work pt-0" aria-labelledby="more-title">
        <div className="work__intro">
          <h2 id="more-title" data-reveal className="big-sans m-0 text-[clamp(2rem,4.2vw,4.6rem)]">
            <span className="line">
              <span>You may</span>
            </span>
            <span className="line">
              <span className="em-italic text-gold-400">also like</span>
            </span>
          </h2>
          <DotButton href="/catalogue" className="justify-self-start text-parchment-100">
            See all
          </DotButton>
        </div>

        <ul className="catalogue__grid book__related m-0 list-none p-0">
          {more.map((b) => (
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
