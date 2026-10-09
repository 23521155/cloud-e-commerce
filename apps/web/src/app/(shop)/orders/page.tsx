import type { Metadata } from "next";
import Link from "next/link";
import { MotionController } from "@/components/motion/MotionController";
import { Orders } from "@/components/orders/Orders";
import { getOrders, ORDER_FILTERS, ORDER_SORTS, queryOrders, type OrderFilter, type OrderSort } from "@/lib/orders";

export const metadata: Metadata = {
  title: "Your orders — Marginalleya",
  description: "Every order you have placed, where it is now, and the copies in it.",
};

const first = (v: string | string[] | undefined) => (Array.isArray(v) ? v[0] : v);

export default async function OrdersPage({ searchParams }: PageProps<"/orders">) {
  const sp = await searchParams;
  const status = ORDER_FILTERS.find((f) => f.key === first(sp.status))?.key as OrderFilter | undefined;
  const sort = (ORDER_SORTS.find((s) => s.key === first(sp.sort))?.key ?? "newest") as OrderSort;
  const q = (first(sp.q) ?? "").trim().slice(0, 80);

  const [total, orders] = await Promise.all([getOrders().then((o) => o.length), queryOrders({ q, status, sort })]);
  const filtered = Boolean(q || status);

  /** Orders URL from the current search, filter and sort with some of them overridden. */
  const href = (next: { status?: OrderFilter | null; q?: string; sort?: OrderSort }) => {
    const params = new URLSearchParams();
    const text = next.q === undefined ? q : next.q;
    const s = next.status === undefined ? status : next.status;
    const o = next.sort ?? sort;
    if (text) params.set("q", text);
    if (s) params.set("status", s);
    if (o !== "newest") params.set("sort", o);
    const qs = params.toString();
    return qs ? `/orders?${qs}` : "/orders";
  };

  return (
    <>
      {/* A new search renders new cards; remount so the reveal observer picks them up */}
      <MotionController key={`${status}-${sort}-${q}`} />

      <section className="work basket" aria-labelledby="orders-title">
        <div className="catalogue__head">
          <h1 id="orders-title" data-reveal className="big-sans m-0 text-[clamp(2.4rem,6vw,7rem)]">
            <span className="line">
              <span>Your</span>
            </span>
            <span className="line">
              <span className="em-italic text-gold-400">orders</span>
            </span>
          </h1>
          <p className="m-0 max-w-[34ch] text-[clamp(17px,1.3vw,21px)] leading-relaxed text-parchment-200">
            {total === 0 ? "Nothing ordered yet." : `${total} ${total === 1 ? "order" : "orders"} so far. Each card shows where the parcel is now.`}
          </p>
        </div>

        {total > 0 && (
          <>
            <form className="catalogue__search" action="/orders" role="search" aria-label="Search your orders">
              {status && <input type="hidden" name="status" value={status} />}
              {sort !== "newest" && <input type="hidden" name="sort" value={sort} />}
              <label htmlFor="orders-q" className="catalogue__card-head small-upper">
                <span>Order search</span>
                <span>Order no., title or author</span>
              </label>
              <div className="catalogue__card-body">
                <input
                  id="orders-q"
                  key={q}
                  type="search"
                  name="q"
                  defaultValue={q}
                  maxLength={80}
                  placeholder="MA-0147, Heidi…"
                  autoComplete="off"
                  className="catalogue__input"
                />
                {q && (
                  <Link href={href({ q: "" })} className="catalogue__clear small-upper under-hover">
                    Clear
                  </Link>
                )}
                <button type="submit" className="catalogue__stamp">
                  Find
                </button>
              </div>
            </form>

            <nav className="catalogue__filters" aria-label="Filter your orders">
              <ul className="catalogue__row" aria-label="Status">
                <li>
                  <Link href={href({ status: null })} className={`catalogue__chip ${!status ? "is-active" : ""}`} aria-current={!status ? "true" : undefined}>
                    All orders
                  </Link>
                </li>
                {ORDER_FILTERS.map((f) => (
                  <li key={f.key}>
                    <Link href={href({ status: f.key })} className={`catalogue__chip ${status === f.key ? "is-active" : ""}`} aria-current={status === f.key ? "true" : undefined}>
                      {f.name}
                    </Link>
                  </li>
                ))}
              </ul>

              <ul className="catalogue__row catalogue__row--sort" aria-label="Sort">
                <li className="small-upper text-gold-400">Sort</li>
                {ORDER_SORTS.map((o) => (
                  <li key={o.key}>
                    <Link
                      href={href({ sort: o.key })}
                      className={`small-upper under-hover ${sort === o.key ? "text-gold-300" : "text-parchment-200"}`}
                      aria-current={sort === o.key ? "true" : undefined}
                    >
                      {o.name}
                    </Link>
                  </li>
                ))}
              </ul>
            </nav>
          </>
        )}

        <Orders orders={orders} filtered={filtered} />
      </section>
    </>
  );
}
