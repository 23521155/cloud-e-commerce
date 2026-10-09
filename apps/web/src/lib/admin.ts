// Admin data. Mock until the API and database exist. The shop sees every reader's orders: the
// signed-in reader's mock orders (lib/orders) plus other readers', each with who placed it.
// Names are unaccented: the display face does not render Vietnamese diacritics reliably.
import { getOrders, orderTotal, withBooks, type Order, type OrderStatus } from "@/lib/orders";
import { PAYMENT_METHODS, type PaymentMethodId } from "@/lib/payment";

export type Person = { name: string; email: string };

export type ShopOrder = Order & { customer: Person };

/** The reader the customer-side mocks belong to (same as the account page). */
const READER: Person = { name: "Nguyen Van An", email: "an.nguyen@example.com" };

const OTHER_ORDERS: { id: string; placed: string; status: OrderStatus; payment: PaymentMethodId; slugs: string[]; customer: Person }[] = [
  { id: "MA-0151", placed: "2026-10-06", status: "paid", payment: "momo", slugs: ["jamaica-inn-b001bjerdu"], customer: { name: "Tran Minh Thu", email: "thu.tran@example.com" } },
  { id: "MA-0149", placed: "2026-10-05", status: "packed", payment: "vietqr", slugs: ["jamaica-inn-b000k4z9km", "a-tree-grows-in-brooklyn-1943-b001kks4ny"], customer: { name: "Le Hoang Nam", email: "nam.le@example.com" } },
  { id: "MA-0148", placed: "2026-10-04", status: "placed", payment: "vietqr", slugs: ["pride-and-prejudice-b0006aqlwu"], customer: { name: "Pham Thu Ha", email: "ha.pham@example.com" } },
  { id: "MA-0144", placed: "2026-10-01", status: "placed", payment: "cod", slugs: ["officially-dead-b000jdt7l6"], customer: { name: "Do Quang Huy", email: "huy.do@example.com" } },
  { id: "MA-0130", placed: "2026-09-20", status: "delivered", payment: "momo", slugs: ["lost-horizon-b0006dlxji"], customer: { name: "Hoang Lan", email: "lan.hoang@example.com" } },
];

/** Every order in the shop, newest first. */
export async function getShopOrders(): Promise<ShopOrder[]> {
  const [mine, others] = await Promise.all([getOrders(), withBooks(OTHER_ORDERS)]);
  return [...mine.map((o) => ({ ...o, customer: READER })), ...others].sort((a, b) => b.placed.localeCompare(a.placed));
}

export const STATUS_LABEL: Record<OrderStatus, string> = {
  placed: "Placed",
  paid: "Paid",
  packed: "Packed",
  shipped: "Shipped",
  delivered: "Delivered",
  cancelled: "Cancelled",
};

export const ORDER_STATUSES = Object.keys(STATUS_LABEL) as OrderStatus[];

/** Search by order number, customer name or email, title or author; optionally one status. */
export async function queryShopOrders({ q = "", status }: { q?: string; status?: OrderStatus }): Promise<ShopOrder[]> {
  const needle = q.trim().toLowerCase();
  return (await getShopOrders()).filter(
    (o) =>
      (!status || o.status === status) &&
      (!needle ||
        [o.id, o.customer.name, o.customer.email, ...o.books.flatMap((b) => [b.title, b.author])].some((s) => s.toLowerCase().includes(needle))),
  );
}

export { orderTotal };

export type Offer = {
  id: string;
  /** ISO date the seller sent it */
  received: string;
  seller: Person;
  title: string;
  author: string;
  binding: string;
  grade: string;
  faults: string;
  /** VND, when the seller named a price */
  asking?: number;
  photos: number;
  answered: boolean;
};

const OFFERS: Offer[] = [
  { id: "OF-0059", received: "2026-10-07", seller: { name: "Dang Khoa", email: "khoa.dang@example.com" }, title: "The Old Man and the Sea", author: "Ernest Hemingway", binding: "Paperback", grade: "Good", faults: "Cover creased, spine faded by the sun.", photos: 2, answered: false },
  { id: "OF-0058", received: "2026-10-06", seller: { name: "Bui Thanh Tam", email: "tam.bui@example.com" }, title: "Rebecca", author: "Daphne du Maurier", binding: "Hardcover", grade: "Good", faults: "Jacket torn at the spine head, light foxing on the endpapers.", asking: 900000, photos: 3, answered: false },
  { id: "OF-0057", received: "2026-10-05", seller: { name: "Vo Ngoc Mai", email: "mai.vo@example.com" }, title: "The Good Earth", author: "Pearl S. Buck", binding: "Hardcover", grade: "Fair", faults: "No jacket, a previous owner's name in ink on the flyleaf.", photos: 1, answered: false },
  { id: "OF-0055", received: "2026-10-02", seller: { name: "Hoang Lan", email: "lan.hoang@example.com" }, title: "Anne of Green Gables", author: "L. M. Montgomery", binding: "Paperback", grade: "Very good", faults: "None I can see.", asking: 150000, photos: 0, answered: true },
];

export type OfferFilter = "new" | "answered";

/** Newest first. Search by reference, seller, title or author; optionally only new or answered. */
export function getOffers({ q = "", state }: { q?: string; state?: OfferFilter } = {}): Offer[] {
  const needle = q.trim().toLowerCase();
  return OFFERS.filter(
    (f) =>
      (!state || (state === "new") !== f.answered) &&
      (!needle || [f.id, f.seller.name, f.seller.email, f.title, f.author].some((s) => s.toLowerCase().includes(needle))),
  );
}

export type Task = { ref: string; kind: "order" | "offer"; who: string; what: string; since: string; href: string };

const copies = (n: number) => `${n} ${n === 1 ? "copy" : "copies"}`;

/** What needs a person at the shop next, oldest first so nothing waits too long. */
export async function getTasks(): Promise<Task[]> {
  const tasks: Task[] = [];
  for (const o of await getShopOrders()) {
    const base = { ref: o.id, kind: "order" as const, who: o.customer.name, since: o.placed, href: `/admin/orders?q=${o.id}` };
    if (o.status === "paid" || (o.status === "placed" && o.payment === "cod")) tasks.push({ ...base, what: `Pack ${copies(o.books.length)}` });
    if (o.status === "packed") tasks.push({ ...base, what: `Hand ${copies(o.books.length)} to the courier` });
  }
  for (const f of OFFERS) {
    if (!f.answered) tasks.push({ ref: f.id, kind: "offer", who: f.seller.name, what: `Reply with an offer for “${f.title}”`, since: f.received, href: `/admin/offers?q=${f.id}` });
  }
  return tasks.sort((a, b) => a.since.localeCompare(b.since));
}

const DAY_MS = 86_400_000;
const isoDay = (t: number) => new Date(t).toISOString().slice(0, 10);

/** Order count per status, in the order an order moves. */
export async function statusChart() {
  const orders = await getShopOrders();
  return ORDER_STATUSES.map((status) => ({ status, label: STATUS_LABEL[status], count: orders.filter((o) => o.status === status).length }));
}

/**
 * Revenue per day (cancelled orders excluded) over the 14 days up to the newest order.
 * Ends at the newest order rather than today so the mock data always fills the window.
 */
export async function revenueChart(days = 14) {
  const orders = (await getShopOrders()).filter((o) => o.status !== "cancelled");
  const end = Date.parse(orders.reduce((max, o) => (o.placed > max ? o.placed : max), "0000-00-00"));
  return Array.from({ length: days }, (_, i) => {
    const date = isoDay(end - (days - 1 - i) * DAY_MS);
    return { date, value: orders.filter((o) => o.placed === date).reduce((sum, o) => sum + orderTotal(o), 0) };
  });
}

/**
 * How orders were paid (cancelled excluded). Fixed method order, never sorted by size, so each
 * method keeps its place and colour in the stack.
 */
export async function paymentChart() {
  const orders = (await getShopOrders()).filter((o) => o.status !== "cancelled");
  return PAYMENT_METHODS.map((m) => ({ payment: m.id, label: m.name, count: orders.filter((o) => o.payment === m.id).length })).filter((r) => r.count > 0);
}

/** Overview counts: where orders stand and how many offers wait. */
export async function overviewCounts() {
  const orders = await getShopOrders();
  const count = (fn: (o: ShopOrder) => boolean) => orders.filter(fn).length;
  return {
    orders: [
      { label: "Waiting for payment", count: count((o) => o.status === "placed" && o.payment !== "cod") },
      { label: "To pack", count: count((o) => o.status === "paid" || (o.status === "placed" && o.payment === "cod")) },
      { label: "To hand to the courier", count: count((o) => o.status === "packed") },
      { label: "With the courier", count: count((o) => o.status === "shipped") },
    ],
    totalOrders: orders.length,
    newOffers: OFFERS.filter((f) => !f.answered).length,
    answeredOffers: OFFERS.filter((f) => f.answered).length,
  };
}
