import type { Metadata } from "next";
import Link from "next/link";
import { PaymentChart, RevenueChart, StatusChart } from "@/components/admin/Charts";
import { getTasks, overviewCounts, paymentChart, revenueChart, statusChart } from "@/lib/admin";

export const metadata: Metadata = {
  title: "Overview — Shop office",
};

const day = new Intl.DateTimeFormat("en-GB", { day: "numeric", month: "short" });
const today = new Intl.DateTimeFormat("en-GB", { weekday: "long", day: "numeric", month: "long", year: "numeric" });

/** Admin landing: who is waiting on the shop and for what, then a short count of each section. */
export default function AdminOverviewPage() {
  const tasks = getTasks();
  const counts = overviewCounts();

  return (
    <div className="admin__page">
      <header className="admin__head">
        <h1 className="admin__title">Overview</h1>
        <p className="admin__date">{today.format(new Date())}</p>
      </header>

      <section aria-labelledby="todo-title">
        <h2 id="todo-title" className="admin__h2">
          To do <span className="admin__count">{tasks.length}</span>
        </h2>
        {tasks.length === 0 ? (
          <p className="admin__empty">Nothing waiting. Every order is moving and every offer has a reply.</p>
        ) : (
          <ol className="admin__tasks">
            {tasks.map((t) => (
              <li key={`${t.kind}-${t.ref}`}>
                <span className={`admin__tag admin__tag--${t.kind}`}>{t.kind === "order" ? "Order" : "Offer"}</span>
                <span className="admin__task">
                  <span className="admin__who">{t.who}</span>
                  <span className="admin__what">{t.what}</span>
                </span>
                <span className="admin__task-meta">
                  <span className="admin__ref">{t.ref}</span>
                  <time dateTime={t.since}>since {day.format(new Date(t.since))}</time>
                </span>
                <Link href={t.href} className="admin__open small-upper under-hover" aria-label={`Open ${t.ref}`}>
                  Open
                </Link>
              </li>
            ))}
          </ol>
        )}
      </section>

      <section className="admin__charts" aria-label="Charts">
        <div className="admin__chart-card admin__chart-card--wide">
          <RevenueChart data={revenueChart()} />
        </div>
        <div className="admin__chart-card">
          <StatusChart data={statusChart()} />
        </div>
        <div className="admin__chart-card">
          <PaymentChart data={paymentChart()} />
        </div>
      </section>

      <div className="admin__pair">
        <section className="admin__panel" aria-labelledby="sum-orders">
          <h2 id="sum-orders" className="admin__h2">
            Orders <span className="admin__count">{counts.totalOrders}</span>
          </h2>
          <dl className="admin__facts">
            {counts.orders.map((r) => (
              <div key={r.label}>
                <dt>{r.label}</dt>
                <dd>{r.count}</dd>
              </div>
            ))}
          </dl>
          <Link href="/admin/orders" className="admin__more small-upper under-hover">
            All orders
          </Link>
        </section>

        <section className="admin__panel" aria-labelledby="sum-offers">
          <h2 id="sum-offers" className="admin__h2">
            Offers from sellers
          </h2>
          <dl className="admin__facts">
            <div>
              <dt>Waiting for a reply</dt>
              <dd>{counts.newOffers}</dd>
            </div>
            <div>
              <dt>Answered</dt>
              <dd>{counts.answeredOffers}</dd>
            </div>
          </dl>
          <Link href="/admin/offers" className="admin__more small-upper under-hover">
            All offers
          </Link>
        </section>
      </div>
    </div>
  );
}
