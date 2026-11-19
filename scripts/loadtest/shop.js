// k6 load test: PEAK simulated shoppers browse the web app, and a few place a COD order.
//
// Run from the repo root (k6 runs in Docker, nothing to install):
//   wsl -e docker run --rm -v /mnt/d/cloud-e-commerce/scripts/loadtest:/scripts grafana/k6:1.0.0 run \
//     -e WEB=https://<web fqdn> -e GATEWAY=https://<gateway fqdn> /scripts/shop.js
//
// Options (-e NAME=value): PEAK (default 100) shoppers, RAMP (1m) to reach them, HOLD (3m) at the peak,
// ORDER_RATE (0.05) share of visits that end in an order. Every order sells a real book in that database.
import http from "k6/http";
import { check, group, sleep } from "k6";
import { Counter } from "k6/metrics";

const WEB = __ENV.WEB;
const GATEWAY = __ENV.GATEWAY;
const PEAK = Number(__ENV.PEAK || 100);
const ORDER_RATE = Number(__ENV.ORDER_RATE || 0.05);

const rateLimited = new Counter("rate_limited_429");
const soldOut = new Counter("order_book_unavailable_409");

export const options = {
  scenarios: {
    shoppers: {
      executor: "ramping-vus",
      stages: [
        { duration: __ENV.RAMP || "1m", target: PEAK },
        { duration: __ENV.HOLD || "3m", target: PEAK },
        { duration: "30s", target: 0 },
      ],
      gracefulRampDown: "30s",
    },
  },
  thresholds: {
    "http_req_duration{page:home}": ["p(95)<2000"],
    "http_req_duration{page:catalogue}": ["p(95)<2000"],
    "http_req_duration{page:book}": ["p(95)<2000"],
    http_req_failed: ["rate<0.05"],
  },
  summaryTrendStats: ["avg", "med", "p(90)", "p(95)", "max"],
};

const SEARCH_WORDS = ["mystery", "history", "poems", "war", "love", "children", "music", "travel"];

// Runs once before the test: collect slugs of books that can still be bought.
export function setup() {
  if (!WEB || !GATEWAY) throw new Error("Pass -e WEB=https://... and -e GATEWAY=https://...");
  const slugs = [];
  for (let page = 1; page <= 5; page++) {
    const res = http.get(`${GATEWAY}/api/catalog/books?page=${page}`);
    if (res.status !== 200) continue;
    for (const b of res.json("items")) if (b.status === "AVAILABLE") slugs.push(b.slug);
  }
  if (slugs.length === 0) throw new Error("No available books found through the gateway");
  return { slugs };
}

function pick(list) {
  return list[Math.floor(Math.random() * list.length)];
}

function visit(url, page) {
  const res = http.get(url, { tags: { page } });
  if (res.status === 429) rateLimited.add(1, { page });
  check(res, { [`${page} is 200`]: (r) => r.status === 200 });
  return res;
}

// One iteration = one shopper's visit, with think time between pages.
export default function ({ slugs }) {
  group("browse", () => {
    visit(`${WEB}/`, "home");
    sleep(1 + Math.random() * 2);

    visit(`${WEB}/catalogue?page=${1 + Math.floor(Math.random() * 5)}`, "catalogue");
    sleep(1 + Math.random() * 2);

    if (Math.random() < 0.3) {
      visit(`${WEB}/catalogue?q=${pick(SEARCH_WORDS)}`, "search");
      sleep(1 + Math.random() * 2);
    }

    visit(`${WEB}/books/${pick(slugs)}`, "book");
    sleep(1 + Math.random() * 2);

    visit(`${WEB}/cart`, "cart");
  });

  // The web server places orders through the gateway, so the test does the same.
  if (Math.random() < ORDER_RATE) {
    group("checkout", () => {
      const res = http.post(
        `${GATEWAY}/api/orders`,
        JSON.stringify({
          items: [{ slug: pick(slugs) }],
          paymentMethod: "cod",
          email: `loadtest+${__VU}-${__ITER}@example.com`,
          phone: "0900000000",
          fullName: `Load Test ${__VU}`,
          address: "1 Test Street",
          city: "Ho Chi Minh City",
          country: "Vietnam",
        }),
        {
          headers: { "Content-Type": "application/json", "Idempotency-Key": crypto.randomUUID() },
          tags: { page: "order" },
        },
      );
      if (res.status === 429) rateLimited.add(1, { page: "order" });
      // 409: another shopper got that single copy first, which is expected under load.
      if (res.status === 409) soldOut.add(1);
      check(res, { "order is 201 or 409": (r) => r.status === 201 || r.status === 409 });
    });
  }

  sleep(2 + Math.random() * 3);
}
