# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

- Readers and collectors of used, new and rare books, both in Vietnam and abroad.
- Vietnamese visitors pay in VND; international visitors pay in USD.
- The interface is English-only for every visitor. Vietnamese UI copy is out of scope because the display typeface (IM Fell English) does not render Vietnamese diacritics reliably; Vietnamese book titles are the only Vietnamese text on the site.

## Product Purpose

Marginalleya is a single online bookshop for used, new and rare books. People who want to sell a book sell it to the shop; the shop describes it, lists it, and resells it to other readers. Success is a buyer trusting a described copy enough to buy it.

Also a university course project (đồ án): the shop must run fully on Azure (Web Apps, Azure SQL, Functions) with IaC, CI/CD and monitoring, and must let visitors browse, search and pay for products.

## Positioning

Every copy is a single, specific object, described honestly by the shop that holds it: faults first, virtues after. Listings are individual copies, not interchangeable SKUs.

## Operating Context

- Most listings are one copy only, so quantity is usually 1 and a copy can sell out.
- The shop buys books from users, then resells them; sellers never ship directly to buyers.
- Terminology in the existing UI: "basket" (not cart) for the customer-facing label, "catalogue", "copy", "edition", "binding/condition", "rare".

## Capabilities and Constraints

- Web: Next.js fullstack (`apps/web`); recommender service supplies "you may also like" suggestions.
- Currency: VND and USD. How the currency is chosen (locale, toggle, shipping country) is undecided.
- Payment methods offered at checkout (mock, no gateway wired yet; list in `apps/web/src/lib/payment.ts`): VietQR bank transfer, MoMo and cash on delivery for Vietnam (VND); card and PayPal from anywhere (USD). Which gateways back them is undecided.
- Undecided: payment gateways, shipping options and costs, taxes, returns policy, and the sell-to-us flow. Do not show shipping fees or delivery times until decided.
- Data is mocked (`apps/web/src/lib/catalogue.ts`) until the API and database are wired.

## Brand Commitments

- Name: Marginalleya. Tagline in use: "Used, new & rare books".
- Voice: plain, candid, a little literary; describes faults before beauty.

## Evidence on Hand

- Mock catalogue data in `apps/web/src/lib/catalogue.ts`; some books carry a remote cover scan (`image`).
- No real customers, reviews, testimonials or sales figures exist. Do not invent them.

## Product Principles

1. Honest description earns the sale: show condition and faults plainly.
2. Each copy is unique: design for single copies that can sell out.
3. Do not promise what is undecided (payment, shipping, delivery dates).
4. English UI, two currencies.
