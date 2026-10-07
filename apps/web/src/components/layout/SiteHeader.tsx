"use client";

import { BasketIcon } from "@/components/ui/BasketIcon";
import { Logo } from "@/components/ui/Logo";
import Link from "next/link";
import { useEffect, useRef, useState } from "react";

const MENU = [
  { href: "/catalogue", label: "Catalogue" },
  { href: "/catalogue?rare=1", label: "Rare books" },
  { href: "/sell", label: "Sell books" },
  { href: "/search", label: "Search" },
];

const FOOT = [
  { href: "/cart", label: "Basket" },
  { href: "/wishlist", label: "Wishlist" },
  { href: "/orders", label: "Orders" },
];

/**
 * Bar icons in the BasketIcon manner: 1.4 ink stroke, rough bled edge, plus worn speckle where the
 * ink did not take, like an old engraving. Inherit color from the link.
 */
function InkIcon({ id, children }: { id: string; children: React.ReactNode }) {
  return (
    <svg aria-hidden="true" viewBox="0 0 24 24" width={22} height={22} fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round">
      <defs>
        <filter id={id} x="-10%" y="-10%" width="120%" height="120%">
          <feTurbulence type="fractalNoise" baseFrequency=".9" numOctaves="2" seed="6" result="n" />
          <feDisplacementMap in="SourceGraphic" in2="n" scale="0.9" result="rough" />
          {/* worn ink: fine speckle knocked out of the strokes */}
          <feTurbulence type="fractalNoise" baseFrequency="2.2" numOctaves="1" seed="11" result="s" />
          <feColorMatrix in="s" type="matrix" values="0 0 0 0 0  0 0 0 0 0  0 0 0 0 0  7 0 0 0 -5.2" result="gaps" />
          <feComposite in="rough" in2="gaps" operator="out" />
        </filter>
      </defs>
      <g filter={`url(#${id})`}>{children}</g>
    </svg>
  );
}

/** Engraved heart: heavy outline, a hairline inside it, hatched shade on the left and a curl in the cleft. */
const HeartIcon = () => (
  <InkIcon id="heart-ink">
    <path d="M12 20.4C6.8 17 3 13.6 3 9.2 3 6.4 5.1 4.4 7.6 4.4c1.9 0 3.4 1 4.4 2.6 1-1.6 2.5-2.6 4.4-2.6 2.5 0 4.6 2 4.6 4.8 0 4.4-3.8 7.8-9 11.2Z" />
    <path d="M12 18.2c-4-2.7-6.8-5.4-6.8-8.8 0-1.7 1.2-2.9 2.6-2.9 1.6 0 2.8 1 3.5 2.6.3.6 1.1.6 1.4 0 .7-1.6 1.9-2.6 3.5-2.6 1.4 0 2.6 1.2 2.6 2.9 0 3.4-2.8 6.1-6.8 8.8Z" strokeWidth=".7" />
    <path d="M6.4 11.2 8.6 9M7 13.2l3.6-3.6M8.5 14.9l3-3" strokeWidth=".7" />
    <path d="M12 7c-.5-1.5.2-2.9 1.4-3.1.8-.1 1.2.6.8 1.1" strokeWidth=".9" />
  </InkIcon>
);

/** Victorian locket: hanging ring, double oval frame, a bust inside. */
const UserIcon = () => (
  <InkIcon id="user-ink">
    <circle cx="12" cy="2.4" r="1" strokeWidth="1" />
    <ellipse cx="12" cy="12.6" rx="7.2" ry="8.8" />
    <ellipse cx="12" cy="12.6" rx="5.6" ry="7.2" strokeWidth=".7" />
    <circle cx="12" cy="10.4" r="2.3" />
    <path d="M8.1 17.6c.6-2.3 2-3.6 3.9-3.6s3.3 1.3 3.9 3.6" />
  </InkIcon>
);

/**
 * rogue.studio-style navigation: a thin vertical bar on the right (a top bar on phones)
 * with a dots toggle that slides a menu panel out of the bar.
 */
export function SiteHeader() {
  const [open, setOpen] = useState(false);
  const toggleRef = useRef<HTMLButtonElement>(null);
  const firstLinkRef = useRef<HTMLAnchorElement>(null);

  useEffect(() => {
    window.dispatchEvent(new CustomEvent("of:scroll-lock", { detail: open }));
    if (!open) return;
    firstLinkRef.current?.focus();
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== "Escape") return;
      setOpen(false);
      toggleRef.current?.focus();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open]);

  const close = () => setOpen(false);

  return (
    <header data-open={open || undefined}>
      <div className="side-nav__bar">
        <Link href="/" className="side-nav__logo" aria-label="Marginalleya — home" onClick={close}>
          <Logo className="h-7 w-auto text-gold-400" />
          <span className="side-nav__wordmark" aria-hidden="true">
            Marginalleya
          </span>
        </Link>
        <Link href="/cart" className="side-nav__icon" aria-label="Basket" onClick={close}>
          <BasketIcon size={22} />
        </Link>
        <Link href="/wishlist" className="side-nav__icon" aria-label="Wishlist" onClick={close}>
          <HeartIcon />
        </Link>
        <Link href="/account" className="side-nav__icon" aria-label="Your account" onClick={close}>
          <UserIcon />
        </Link>
        <button
          ref={toggleRef}
          type="button"
          className="side-nav__toggle"
          aria-expanded={open}
          aria-controls="site-menu"
          aria-label={open ? "Close menu" : "Open menu"}
          onClick={() => setOpen((o) => !o)}
        >
          <span className="side-nav__dots" aria-hidden="true">
            <span />
            <span />
            <span />
          </span>
        </button>
        <Link href="/sell" className="side-nav__cta small-upper" onClick={close}>
          Sell your books
        </Link>
      </div>

      <div className="side-nav__overlay" aria-hidden="true" onClick={close} />

      <nav id="site-menu" className="menu" aria-label="Main" inert={!open}>
        <ul className="menu__list">
          {MENU.map((item, i) => (
            <li key={item.href}>
              <Link ref={i === 0 ? firstLinkRef : undefined} href={item.href} className="menu__link" onClick={close}>
                <span className="sr-only">{item.label}</span>
                <span className="menu__track" aria-hidden="true">
                  {[0, 1].map((half) => (
                    <span key={half}>
                      {item.label}
                      <i>✶</i>
                      <em>{item.label}</em>
                      <i>✶</i>
                    </span>
                  ))}
                </span>
              </Link>
            </li>
          ))}
        </ul>

        <div>
          <span className="small-upper text-ink-600">Account</span>
          <Link href="/sign-in" className="under-hover mt-1 block w-fit text-lg" onClick={close}>
            Sign in
          </Link>
        </div>

        <div className="small-upper flex flex-wrap items-center justify-between gap-4 border-t border-walnut-900/15 pt-5 text-ink-600">
          <div className="flex flex-wrap gap-x-7 gap-y-2">
            {FOOT.map((item) => (
              <Link key={item.href} href={item.href} className="under-hover" onClick={close}>
                {item.label}
              </Link>
            ))}
          </div>
          <span>© 2026 Marginalleya</span>
        </div>
      </nav>
    </header>
  );
}
