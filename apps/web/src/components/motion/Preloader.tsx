"use client";

import { useEffect } from "react";
import { BookCover } from "@/components/ui/BookArt";

const COVERS = [
  { title: "The Screwtape Letters", author: "C. S. Lewis", color: "#5c1a16" },
  { title: "Huckleberry Finn", author: "Mark Twain", color: "#24382e" },
  { title: "Black Beauty", author: "Anna Sewell", color: "#3c1d0e" },
  { title: "I Capture the Castle", author: "Dodie Smith", color: "#7a2620" },
  { title: "The Nine Tailors", author: "Dorothy L. Sayers", color: "#1a2a22" },
  { title: "Anthem", author: "Ayn Rand", color: "#64331e" },
];

const DURATION_MS = 2200;

/**
 * Intro overlay, shown once per browser session. The inline script in the root layout sets
 * html[data-loaded="instant"] before paint for returning visitors and reduced-motion users.
 */
export function Preloader() {
  useEffect(() => {
    const root = document.documentElement;
    if (root.dataset.loaded) return;
    const timer = window.setTimeout(() => {
      root.dataset.loaded = "true";
      try {
        sessionStorage.setItem("of-preloaded", "1");
      } catch {
        // storage unavailable: the intro simply plays again next time
      }
    }, DURATION_MS);
    return () => window.clearTimeout(timer);
  }, []);

  const phrase = (
    <span>
      <span className="preload__sans">Marginalleya</span>
      <span className="preload__star">❦</span>
      <span className="preload__serif">Used &amp; rare books</span>
      <span className="preload__star">❦</span>
    </span>
  );

  return (
    <div className="preload" aria-hidden="true">
      <div className="preload__ticker">
        <div className="preload__track">
          {phrase}
          {phrase}
          {phrase}
          {phrase}
        </div>
      </div>
      <div className="preload__images">
        {COVERS.map((cover) => (
          <div key={cover.title} className="preload__image">
            <BookCover {...cover} className="w-[78%]" />
          </div>
        ))}
      </div>
    </div>
  );
}
