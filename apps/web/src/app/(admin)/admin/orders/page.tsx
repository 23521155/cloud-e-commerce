import type { Metadata } from "next";
import { FilterBar, StatusBadge } from "@/components/admin/AdminUI";
import { getShopOrders, ORDER_STATUSES, orderTotal, queryShopOrders, STATUS_LABEL } from "@/lib/admin";
import type { OrderStatus } from "@/lib/orders";
import { getPaymentMethod } from "@/lib/payment";

export const metadata: Metadata = {
  title: "Orders — Shop office",
};

const price = new Intl.NumberFormat("en-US", { style: "currency", currency: "VND", maximumFractionDigits: 0 });
const day = new Intl.DateTimeFormat("en-GB", { day: "numeric", month: "short", year: "numeric" });

const first = (v: string | string[] | undefined) => (Array.isArray(v) ? v[0] : v);

/** Every order in the shop with who placed it, searchable and filtered by status. */
export default async function AdminOrdersPage({ searchParams }: PageProps<"/admin/orders">) {
  const sp = await searchParams;
  const status = ORDER_STATUSES.find((s) => s === first(sp.status)) as OrderStatus | undefined;
  const q = (first(sp.q) ?? "").trim().slice(0, 80);

  const total = getShopOrders().length;
  const orders = queryShopOrders({ q, status });

  const href = (next: { status?: OrderStatus | null; q?: string }) => {
    const params = new URLSearchParams();
    const text = next.q === undefined ? q : next.q;
    const s = next.status === undefined ? status : next.status;
    if (text) params.set("q", text);
    if (s) params.set("status", s);
    const qs = params.toString();
    return qs ? `/admin/orders?${qs}` : "/admin/orders";
  };

  return (
    <div className="admin__page">
      <header className="admin__head">
        <h1 className="admin__title">Orders</h1>
        <p className="admin__date">{total} in all</p>
      </header>

      <FilterBar
        action="/admin/orders"
        q={q}
        label="Search the order book"
        placeholder="Order no., customer, title…"
        keep={status ? { status } : undefined}
        clearHref={href({ q: "" })}
        chips={[
          { label: "All", href: href({ status: null }), active: !status },
          ...ORDER_STATUSES.map((s) => ({ label: STATUS_LABEL[s], href: href({ status: s }), active: status === s })),
        ]}
      />

      {orders.length === 0 ? (
        <p className="admin__empty">No orders match. Try another order number, name or title.</p>
      ) : (
        <div className="admin__scroll">
          <table className="admin__table admin__table--wide">
            <caption className="sr-only">Orders, newest first</caption>
            <thead>
              <tr>
                <th scope="col">Order</th>
                <th scope="col">Customer</th>
                <th scope="col">Copies</th>
                <th scope="col">Payment</th>
                <th scope="col" className="is-num">
                  Total
                </th>
                <th scope="col">Status</th>
              </tr>
            </thead>
            <tbody>
              {orders.map((o) => (
                <tr key={o.id}>
                  <th scope="row">
                    {o.id}
                    <span className="admin__sub">
                      <time dateTime={o.placed}>{day.format(new Date(o.placed))}</time>
                    </span>
                  </th>
                  <td>
                    {o.customer.name}
                    <span className="admin__sub">{o.customer.email}</span>
                  </td>
                  <td>
                    {o.books[0]?.title}
                    <span className="admin__sub">{o.books.length > 1 ? `and ${o.books.length - 1} more` : o.books[0]?.author}</span>
                  </td>
                  <td>{getPaymentMethod(o.payment)?.name ?? "—"}</td>
                  <td className="is-num">{price.format(orderTotal(o))}</td>
                  <td>
                    <StatusBadge status={o.status} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
