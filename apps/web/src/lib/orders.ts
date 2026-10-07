// Placeholder order history until orders are stored (DB). Every line is a single copy.
import { getBook, type Book } from "@/lib/catalogue";
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

const MOCK: { id: string; placed: string; status: OrderStatus; payment: PaymentMethodId; slugs: string[] }[] = [
  { id: "MA-0147", placed: "2026-10-03", status: "shipped", payment: "vietqr", slugs: ["the-little-prince", "heidi"] },
  { id: "MA-0139", placed: "2026-09-28", status: "placed", payment: "cod", slugs: ["mythology"] },
  { id: "MA-0121", placed: "2026-09-12", status: "delivered", payment: "momo", slugs: ["the-razors-edge", "pavilion-of-women", "hiroshima"] },
  { id: "MA-0102", placed: "2026-08-30", status: "cancelled", payment: "vietqr", slugs: ["marigold-garden"] },
];

/** Newest first */
export function getOrders(): Order[] {
  return MOCK.map(({ slugs, ...order }) => ({
    ...order,
    books: slugs.flatMap((slug) => {
      const book = getBook(slug);
      return book ? [book] : [];
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
export function queryOrders({ q = "", status, sort = "newest" }: { q?: string; status?: OrderFilter; sort?: OrderSort }): Order[] {
  const needle = q.trim().toLowerCase();
  const dir = sort === "oldest" ? 1 : -1;
  return getOrders()
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
