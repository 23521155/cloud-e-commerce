"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import type { OrderStatus } from "@/lib/orders";

const SECTIONS = [
  // No page lives at /admin itself: it answers 404, so the bare path gives nothing away
  { href: "/admin/overview", label: "Overview" },
  { href: "/admin/orders", label: "Orders" },
  { href: "/admin/offers", label: "Offers" },
];

/** Sidebar sections; the one for the current page is marked. */
export function AdminNav() {
  const path = usePathname();
  return (
    <nav aria-label="Admin">
      <ul className="admin__nav">
        {SECTIONS.map((s) => {
          const active = path === s.href;
          return (
            <li key={s.href}>
              <Link href={s.href} className={`admin__link${active ? " is-active" : ""}`} aria-current={active ? "page" : undefined}>
                {s.label}
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}

type Chip = { label: string; href: string; active: boolean };

/**
 * Search box and filter chips above an admin table. Plain GET form: the filter lives in the URL,
 * so a filtered list can be bookmarked or shared.
 */
export function FilterBar({
  action,
  q,
  label,
  placeholder,
  keep,
  chips,
  clearHref,
}: {
  action: string;
  q: string;
  label: string;
  placeholder: string;
  /** Other params the search should keep, e.g. the active status */
  keep?: Record<string, string>;
  chips: Chip[];
  clearHref: string;
}) {
  return (
    <div className="admin__filters">
      <form action={action} role="search" className="admin__search">
        {Object.entries(keep ?? {}).map(([k, v]) => (
          <input key={k} type="hidden" name={k} value={v} />
        ))}
        <label htmlFor="admin-q" className="admin__search-label small-upper">
          {label}
        </label>
        <span className="admin__search-line">
          <input id="admin-q" key={q} type="search" name="q" defaultValue={q} placeholder={placeholder} maxLength={80} autoComplete="off" className="admin__input" />
          {q && (
            <Link href={clearHref} className="admin__clear small-upper under-hover">
              Clear
            </Link>
          )}
          {/* The catalogue's rubber stamp, so searching feels the same on both sides of the shop */}
          <button type="submit" className="catalogue__stamp">
            Find
          </button>
        </span>
      </form>
      <ul className="admin__chips" aria-label="Filter">
        {chips.map((c) => (
          <li key={c.href}>
            <Link href={c.href} className={`admin__chip${c.active ? " is-active" : ""}`} aria-current={c.active ? "true" : undefined}>
              {c.label}
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}

const BADGE: Record<OrderStatus, string> = {
  placed: "Placed",
  paid: "Paid",
  packed: "Packed",
  shipped: "Shipped",
  delivered: "Delivered",
  cancelled: "Cancelled",
};

export function StatusBadge({ status }: { status: OrderStatus }) {
  return <span className={`admin__badge admin__badge--${status}`}>{BADGE[status]}</span>;
}
