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
