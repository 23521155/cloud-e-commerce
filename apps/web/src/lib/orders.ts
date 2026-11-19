// A reader's order history, from order-service. Every line is a single copy.
import { getBooks, type Book } from "@/lib/catalogue";
import { listMyOrders, lookupOrder, type PlacedOrder } from "@/lib/order-api";
import type { PaymentMethodId } from "@/lib/payment";

export type OrderStatus = "placed" | "paid" | "packed" | "shipped" | "delivered" | "cancelled";

export type Order = {
  id: string;
  /** ISO date the order was placed */
  placed: string;
  status: OrderStatus;
  payment: PaymentMethodId;
  books: Book[];
};

/** Orders of the mock reader the admin pages still show (lib/admin); not anyone's real orders. */
export const MOCK_READER_ORDERS: { id: string; placed: string; status: OrderStatus; payment: PaymentMethodId; slugs: string[] }[] = [
  { id: "MA-0147", placed: "2026-10-03", status: "shipped", payment: "vietqr", slugs: ["heidi-b00087e21s", "hiroshima-john-hersey-b00b0k8hzy"] },
  { id: "MA-0139", placed: "2026-09-28", status: "placed", payment: "cod", slugs: ["mythology-b000eehz9g"] },
  { id: "MA-0121", placed: "2026-09-12", status: "delivered", payment: "momo", slugs: ["the-razors-edge-b0007izgh2", "pavilion-of-women-b0006aqxl4", "hiroshima-b00005w19o"] },
  { id: "MA-0102", placed: "2026-08-30", status: "cancelled", payment: "vietqr", slugs: ["marigold-garden-pictures-and-rhymes-b00085szfy"] },
];

/** Attach the catalogue books to mock orders, fetching every slug with one request. */
export async function withBooks<T extends { slugs: string[] }>(orders: T[]): Promise<(Omit<T, "slugs"> & { books: Book[] })[]> {
  const books = new Map((await getBooks(orders.flatMap((o) => o.slugs))).map((b) => [b.slug, b]));
  return orders.map(({ slugs, ...order }) => ({
    ...order,
    books: slugs.flatMap((slug) => {
      const book = books.get(slug);
      return book ? [book] : [];
    }),
  }));
}

/**
 * Where an order is, in the steps the reader sees. order-service records payment, not shipping
 * yet, so a confirmed order is "placed" until paid.
 */
function toStatus(o: PlacedOrder): OrderStatus {
  if (o.status === "CANCELLED") return "cancelled";
  return o.paymentStatus === "PAID" ? "paid" : "placed";
}

/**
 * The signed-in reader's orders, newest first, or null when the token was refused. Books come from
 * the catalogue for their details, with the price the reader paid.
 */
export async function getMyOrders(accessToken: string): Promise<Order[] | null> {
  const placed = await listMyOrders(accessToken);
  if (!placed.ok) {
    if (placed.reason === "signed-out") return null;
    throw new Error("Your orders could not be loaded right now.");
  }
  return toOrders(placed.orders);
}

/**
 * A guest's order by its code and the email it was placed with, or null when none matches both.
 * Throws when the order service cannot be reached.
 */
export async function getGuestOrder(code: string, email: string): Promise<Order | null> {
  const found = await lookupOrder(code, email);
  if (!found.ok) {
    if (found.reason === "not-found") return null;
    throw new Error("Your order could not be loaded right now.");
  }
  return (await toOrders([found.order]))[0];
}

/** Orders as the reader sees them: books from the catalogue for their details, with the price paid. */
async function toOrders(placed: PlacedOrder[]): Promise<Order[]> {
  const books = new Map((await getBooks(placed.flatMap((o) => o.items.map((i) => i.slug)))).map((b) => [b.slug, b]));

  return placed.map((o) => ({
    id: o.code,
    placed: o.createdAt,
    status: toStatus(o),
    payment: o.paymentMethod as PaymentMethodId,
    books: o.items.flatMap((item) => {
      const book = books.get(item.slug);
      return book ? [{ ...book, price: item.priceVnd }] : [];
    }),
  }));
}

/** Status filters shown above the list; "open" is everything not yet delivered or cancelled. */
export const ORDER_FILTERS = [
  { key: "open", name: "In progress" },
  { key: "delivered", name: "Delivered" },
  { key: "cancelled", name: "Cancelled" },
] as const;

export type OrderFilter = (typeof ORDER_FILTERS)[number]["key"];

const matchesFilter = (order: Order, filter?: OrderFilter) =>
  !filter || (filter === "open" ? order.status !== "delivered" && order.status !== "cancelled" : order.status === filter);

export const ORDER_SORTS = [
  { key: "newest", name: "Newest" },
  { key: "oldest", name: "Oldest" },
] as const;

export type OrderSort = (typeof ORDER_SORTS)[number]["key"];

/** Search by order number, title or author, narrow by status, then order by date placed. */
export function queryOrders(orders: Order[], { q = "", status, sort = "newest" }: { q?: string; status?: OrderFilter; sort?: OrderSort }): Order[] {
  const needle = q.trim().toLowerCase();
  const dir = sort === "oldest" ? 1 : -1;
  return orders
    .filter(
      (o) =>
        matchesFilter(o, status) &&
        (!needle || o.id.toLowerCase().includes(needle) || o.books.some((b) => b.title.toLowerCase().includes(needle) || b.author.toLowerCase().includes(needle))),
    )
    .sort((a, b) => dir * a.placed.localeCompare(b.placed));
}

export function orderTotal(order: Order): number {
  return order.books.reduce((sum, b) => sum + b.price, 0);
}

/**
 * The steps an order passes through. Cash on delivery has no separate payment step:
 * the courier is paid when the parcel arrives.
 */
export function orderSteps(order: Order): { key: OrderStatus; label: string }[] {
  const steps: { key: OrderStatus; label: string }[] = [
    { key: "placed", label: "Placed" },
    { key: "paid", label: "Paid" },
    { key: "packed", label: "Packed" },
    { key: "shipped", label: "Shipped" },
    { key: "delivered", label: "Delivered" },
  ];
  return order.payment === "cod" ? steps.filter((s) => s.key !== "paid") : steps;
}
