// Overview charts, rendered on the server as plain HTML and CSS (no chart library, crisp text at any width).
// Hover or keyboard focus on a mark shows its value; every chart also opens as a table.
import type { CSSProperties } from "react";
import type { PaymentMethodId } from "@/lib/payment";

const price = new Intl.NumberFormat("en-US", { style: "currency", currency: "VND", maximumFractionDigits: 0 });
const compact = new Intl.NumberFormat("en-US", { notation: "compact", maximumFractionDigits: 1 });
const day = new Intl.DateTimeFormat("en-GB", { day: "numeric", month: "short", timeZone: "UTC" });
const share = new Intl.NumberFormat("en-US", { style: "percent", maximumFractionDigits: 1 });

/**
 * Categorical colours for payment methods, validated against the parchment surface (lightness band,
 * chroma, CVD and normal-vision separation in this adjacency). Colour follows the method, never its rank.
 */
const PAYMENT_COLOR: Partial<Record<PaymentMethodId, string>> = {
  vietqr: "#2a8256",
  momo: "#bf8530",
  cod: "#b0402f",
};
const OTHER_COLOR = "#8c7a68";

function TableView({ caption, head, rows }: { caption: string; head: [string, string]; rows: [string, string][] }) {
  return (
    <details className="chart__table">
      <summary className="small-upper">Show as table</summary>
      <table className="admin__table">
        <caption className="sr-only">{caption}</caption>
        <thead>
          <tr>
            <th scope="col">{head[0]}</th>
            <th scope="col" className="is-num">
              {head[1]}
            </th>
          </tr>
        </thead>
        <tbody>
          {rows.map(([a, b]) => (
            <tr key={a}>
              <th scope="row">{a}</th>
              <td className="is-num">{b}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </details>
  );
}

/** Orders per status as horizontal bars, value at each tip. One series, so no legend. */
export function StatusChart({ data }: { data: { status: string; label: string; count: number }[] }) {
  const max = Math.max(1, ...data.map((d) => d.count));
  return (
    <figure className="chart">
      <figcaption className="chart__title">Orders by status</figcaption>
      <ul className="chart__bars" role="list">
        {data.map((d) => (
          <li key={d.status} className="chart__bar-row">
            <span className="chart__bar-label">{d.label}</span>
            <span className="chart__bar-track">
              {d.count > 0 && (
                <span
                  className="chart__bar"
                  style={{ "--f": d.count / max } as CSSProperties}
                  tabIndex={0}
                  data-tip={`${d.label}: ${d.count} ${d.count === 1 ? "order" : "orders"}`}
                  aria-label={`${d.label}: ${d.count}`}
                />
              )}
              <span className="chart__bar-value">{d.count}</span>
            </span>
          </li>
        ))}
      </ul>
      <TableView caption="Orders by status" head={["Status", "Orders"]} rows={data.map((d) => [d.label, String(d.count)])} />
    </figure>
  );
}

/** Round the top of the value axis up to a clean step: 1, 2 or 5 times a power of ten. */
function niceScale(max: number, ticks = 5) {
  if (max <= 0) return { top: 1, step: 1 };
  const raw = max / ticks;
  const pow = 10 ** Math.floor(Math.log10(raw));
  const step = [1, 2, 5, 10].map((m) => m * pow).find((s) => s >= raw) ?? 10 * pow;
  return { top: Math.ceil(max / step) * step, step };
}

/** Revenue per day as columns on one axis; only the peak day is labelled, the rest live in the tooltip. */
export function RevenueChart({ data }: { data: { date: string; value: number }[] }) {
  const { top, step } = niceScale(Math.max(...data.map((d) => d.value)));
  const ticks = Array.from({ length: Math.round(top / step) + 1 }, (_, i) => i * step);
  const peak = data.reduce((best, d) => (d.value > best.value ? d : best), data[0]);
  const total = data.reduce((sum, d) => sum + d.value, 0);
  const first = data[0]?.date;
  const last = data[data.length - 1]?.date;

  return (
    <figure className="chart">
      <figcaption className="chart__title">
        Revenue by day
        <span className="chart__sub">
          {first && last ? `${day.format(new Date(first))} to ${day.format(new Date(last))}` : ""} · {price.format(total)} in all, cancelled orders left out
        </span>
      </figcaption>
      <div className="chart__plot">
        <div className="chart__grid" aria-hidden="true">
          {ticks.map((t) => (
            <span key={t} className="chart__tick" style={{ "--y": `${(t / top) * 100}%` } as CSSProperties}>
              <span>{t === 0 ? "0" : compact.format(t)}</span>
            </span>
          ))}
        </div>
        <ol className="chart__cols" role="list">
          {data.map((d) => {
            const isPeak = d === peak && d.value > 0;
            return (
              <li key={d.date} className="chart__col-slot">
                {d.value > 0 && (
                  <span
                    className="chart__col"
                    style={{ "--h": `${(d.value / top) * 100}%` } as CSSProperties}
                    tabIndex={0}
                    data-tip={`${day.format(new Date(d.date))}: ${price.format(d.value)}`}
                    aria-label={`${day.format(new Date(d.date))}: ${price.format(d.value)}`}
                  >
                    {isPeak && <span className="chart__peak">{compact.format(d.value)}</span>}
                  </span>
                )}
                <span className="chart__x">{new Date(d.date).getUTCDate()}</span>
              </li>
            );
          })}
        </ol>
      </div>
      <TableView
        caption="Revenue by day"
        head={["Day", "Revenue"]}
        rows={data.map((d) => [day.format(new Date(d.date)), d.value ? price.format(d.value) : "—"])}
      />
    </figure>
  );
}

/** Share of orders by payment method: one 100% bar, 2px surface gaps, legend with counts and shares. */
export function PaymentChart({ data }: { data: { payment: PaymentMethodId; label: string; count: number }[] }) {
  const total = data.reduce((sum, d) => sum + d.count, 0) || 1;
  // One decimal: equal counts always show equal shares, and the shares add up (37.5 + 37.5 + 25)
  const pct = (d: { count: number }) => `${share.format(d.count / total)}`;
  return (
    <figure className="chart">
      <figcaption className="chart__title">
        How orders were paid
        <span className="chart__sub">{total} orders, cancelled left out</span>
      </figcaption>
      <div className="chart__stack">
        {data.map((d) => (
          <span
            key={d.payment}
            className="chart__seg"
            style={{ flexGrow: d.count, "--c": PAYMENT_COLOR[d.payment] ?? OTHER_COLOR } as CSSProperties}
            tabIndex={0}
            data-tip={`${d.label}: ${d.count} (${pct(d)})`}
            aria-label={`${d.label}: ${d.count} orders, ${pct(d)}`}
          />
        ))}
      </div>
      <ul className="chart__legend" role="list">
        {data.map((d) => (
          <li key={d.payment}>
            <span className="chart__swatch" style={{ "--c": PAYMENT_COLOR[d.payment] ?? OTHER_COLOR } as CSSProperties} aria-hidden="true" />
            <span>{d.label}</span>
            <span className="chart__legend-value">
              {d.count} · {pct(d)}
            </span>
          </li>
        ))}
      </ul>
    </figure>
  );
}
