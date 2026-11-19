import Link from "next/link";
import { Seal } from "@/components/cart/Basket";
import { DotButton } from "@/components/ui/DotButton";
import { orderSteps, orderTotal, type Order, type OrderStatus } from "@/lib/orders";
import { getPaymentMethod } from "@/lib/payment";

const price = new Intl.NumberFormat("en-US", { style: "currency", currency: "VND", maximumFractionDigits: 0 });
const day = new Intl.DateTimeFormat("en-GB", { day: "numeric", month: "short", year: "numeric" });

const STATUS_TEXT: Record<OrderStatus, string> = {
  placed: "Waiting for payment",
  paid: "Paid, waiting to be packed",
  packed: "Packed, waiting for the courier",
  shipped: "With the courier",
  delivered: "Delivered",
  cancelled: "Cancelled",
};

const statusText = (order: Order) =>
  order.payment === "cod" && order.status === "placed" ? "Placed, pay the courier on delivery" : STATUS_TEXT[order.status];

/** Where the order is: steps done, the current one marked, the rest still ahead. */
function Tracker({ order }: { order: Order }) {
  const steps = orderSteps(order);
  const current = steps.findIndex((s) => s.key === order.status);
  return (
    <ol className="order__track" aria-label="Order progress">
      {steps.map((s, i) => (
        <li key={s.key} className={i < current ? "is-done" : i === current ? "is-current" : ""} aria-current={i === current ? "step" : undefined}>
          <span className="order__dot" aria-hidden="true" />
          <span className="order__step small-upper">{s.label}</span>
        </li>
      ))}
    </ol>
  );
}

function OrderCard({ order }: { order: Order }) {
  const cancelled = order.status === "cancelled";
  const method = getPaymentMethod(order.payment);
  const titleId = `order-${order.id}`;

  return (
    <article className={`lib-card order${cancelled ? " order--cancelled" : ""}`} aria-labelledby={titleId}>
      <div className="lib-card__frame">
        <h2 id={titleId} className="lib-card__head">
          Order {order.id}
        </h2>

        <div className="lib-card__top">
          <dl className="lib-card__fields">
            <div>
              <dt>Placed</dt>
              <dd>
                <time dateTime={order.placed}>{day.format(new Date(order.placed))}</time>
              </dd>
            </div>
            <div>
              <dt>Payment</dt>
              <dd>{method?.name ?? "—"}</dd>
            </div>
            <div>
              <dt>Status</dt>
              <dd>{statusText(order)}</dd>
            </div>
          </dl>
          <Seal />
        </div>

        {!cancelled && <Tracker order={order} />}

        <table className="lib-card__table">
          <caption className="sr-only">Copies in order {order.id}</caption>
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
            {order.books.map((book, i) => (
              <tr key={book.slug}>
                <td className="lib-card__no">{i + 1}.</td>
                <td>
                  <Link href={`/books/${book.slug}`} className="lib-card__title under-hover">
                    {book.title}
                  </Link>
                  <span className="lib-card__by">
                    {book.author}, {book.year}
                    <span className="lib-card__binding-inline"> · {book.condition}</span>
                  </span>
                </td>
                <td className="lib-card__col-binding">{book.condition}</td>
                <td className="lib-card__price">
                  <span className="lib-card__amount">{price.format(book.price)}</span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        <div className="lib-card__foot">
          <span className="lib-card__marks" aria-hidden="true">
            ❦ ❦ ❦
          </span>
          <p className="lib-card__subtotal m-0">
            <span className="small-upper">Total</span>
            <span className="lib-card__total">{price.format(orderTotal(order))}</span>
          </p>
        </div>
      </div>

      {cancelled && (
        <span className="order__void" aria-hidden="true">
          Cancelled
        </span>
      )}
    </article>
  );
}

/** Order history: one library card per order, newest first. */
export function Orders({ orders, filtered = false }: { orders: Order[]; filtered?: boolean }) {
  if (orders.length === 0 && filtered) {
    return (
      <div className="catalogue__empty">
        <p className="m-0 font-serif text-[clamp(1.6rem,3vw,2.6rem)] uppercase">No orders match</p>
        <p className="m-0 mt-3 text-parchment-200">Try another order number, title or author, or a different status.</p>
        <DotButton href="/orders" className="mt-8">
          Show all orders
        </DotButton>
      </div>
    );
  }

  if (orders.length === 0) {
    return (
      <div className="catalogue__empty">
        <p className="m-0 font-serif text-[clamp(1.6rem,3vw,2.6rem)] uppercase">No orders yet</p>
        <p className="m-0 mt-3 text-parchment-200">When you buy a copy, its order card is kept here.</p>
        <DotButton href="/catalogue" className="mt-8">
          Browse the shelves
        </DotButton>
      </div>
    );
  }

  return (
    <div className="orders">
      {orders.map((o) => (
        <OrderCard key={o.id} order={o} />
      ))}
    </div>
  );
}
