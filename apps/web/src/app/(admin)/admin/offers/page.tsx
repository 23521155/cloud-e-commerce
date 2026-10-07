import type { Metadata } from "next";
import { FilterBar } from "@/components/admin/AdminUI";
import { getOffers, type OfferFilter } from "@/lib/admin";

export const metadata: Metadata = {
  title: "Offers — Shop office",
};

const price = new Intl.NumberFormat("en-US", { style: "currency", currency: "VND", maximumFractionDigits: 0 });
const day = new Intl.DateTimeFormat("en-GB", { day: "numeric", month: "short" });

const first = (v: string | string[] | undefined) => (Array.isArray(v) ? v[0] : v);

const STATES: { key: OfferFilter; label: string }[] = [
  { key: "new", label: "Waiting for a reply" },
  { key: "answered", label: "Answered" },
];

/** Books readers have offered to sell to the shop, with who sent each one. */
export default async function AdminOffersPage({ searchParams }: PageProps<"/admin/offers">) {
  const sp = await searchParams;
  const state = STATES.find((s) => s.key === first(sp.state))?.key;
  const q = (first(sp.q) ?? "").trim().slice(0, 80);

  const total = getOffers().length;
  const offers = getOffers({ q, state });

  const href = (next: { state?: OfferFilter | null; q?: string }) => {
    const params = new URLSearchParams();
    const text = next.q === undefined ? q : next.q;
    const s = next.state === undefined ? state : next.state;
    if (text) params.set("q", text);
    if (s) params.set("state", s);
    const qs = params.toString();
    return qs ? `/admin/offers?${qs}` : "/admin/offers";
  };

  return (
    <div className="admin__page">
      <header className="admin__head">
        <h1 className="admin__title">Offers from sellers</h1>
        <p className="admin__date">{total} in all</p>
      </header>

      <FilterBar
        action="/admin/offers"
        q={q}
        label="Search the offers"
        placeholder="Seller, title, author, OF-…"
        keep={state ? { state } : undefined}
        clearHref={href({ q: "" })}
        chips={[{ label: "All", href: href({ state: null }), active: !state }, ...STATES.map((s) => ({ label: s.label, href: href({ state: s.key }), active: state === s.key }))]}
      />

      {offers.length === 0 ? (
        <p className="admin__empty">No offers match. Try another name, title or reference.</p>
      ) : (
        <div className="admin__scroll">
          <table className="admin__table admin__table--wide">
            <caption className="sr-only">Offers, newest first</caption>
            <thead>
              <tr>
                <th scope="col">Received</th>
                <th scope="col">Seller</th>
                <th scope="col">Book</th>
                <th scope="col">Condition</th>
                <th scope="col">Faults</th>
                <th scope="col" className="is-num">
                  Asking
                </th>
                <th scope="col">State</th>
              </tr>
            </thead>
            <tbody>
              {offers.map((o) => (
                <tr key={o.id}>
                  <td>
                    <time dateTime={o.received}>{day.format(new Date(o.received))}</time>
                    <span className="admin__sub">{o.id}</span>
                  </td>
                  <th scope="row">
                    {o.seller.name}
                    <span className="admin__sub">{o.seller.email}</span>
                  </th>
                  <td>
                    {o.title}
                    <span className="admin__sub">
                      {o.author} · {o.binding}
                    </span>
                  </td>
                  <td>
                    {o.grade}
                    <span className="admin__sub">{o.photos ? `${o.photos} ${o.photos === 1 ? "photo" : "photos"}` : "No photos"}</span>
                  </td>
                  <td className="admin__faults">{o.faults}</td>
                  <td className="is-num">{o.asking ? price.format(o.asking) : "—"}</td>
                  <td>
                    <span className={`admin__badge ${o.answered ? "admin__badge--delivered" : "admin__badge--new"}`}>{o.answered ? "Answered" : "New"}</span>
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
