// k6 swap test: a steady stream of home page views while traffic moves from one copy of the web app
// to the other (an App Service slot swap, or a blue/green switch behind a reverse proxy). It counts
// the requests the switch breaks and records every request, so latency can be plotted around it.
//
// Run from anywhere (k6 runs in Docker, nothing to install):
//   docker run --rm -v <repo>/scripts/loadtest:/scripts:ro -v <results dir>:/out grafana/k6:1.0.0 run \
//     -e URL=https://<web>/ --out csv=/out/swap.csv --summary-export=/out/swap.json /scripts/swap.js
//
// Options (-e NAME=value): URL of the page to request (required), RATE (10) requests per second,
// DURATION (4m). A constant rate keeps the load the same whether responses are fast or slow.
import http from "k6/http";
import { check } from "k6";
import { Counter } from "k6/metrics";

const URL = __ENV.URL;

const failed = new Counter("swap_failed_requests");

export const options = {
  scenarios: {
    steady: {
      executor: "constant-arrival-rate",
      rate: Number(__ENV.RATE || 10),
      timeUnit: "1s",
      duration: __ENV.DURATION || "4m",
      preAllocatedVUs: 20,
      maxVUs: 100,
    },
  },
  summaryTrendStats: ["avg", "med", "p(95)", "p(99)", "max"],
};

export function setup() {
  if (!URL) throw new Error("Pass -e URL=https://...");
}

export default function () {
  const res = http.get(URL, { timeout: "10s" });
  if (!check(res, { "status is 200": (r) => r.status === 200 })) failed.add(1, { status: String(res.status) });
}
