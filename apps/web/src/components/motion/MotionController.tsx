"use client";

import Lenis from "lenis";
import "lenis/dist/lenis.css";
import { useEffect } from "react";

/**
 * One controller for the page's scroll motion:
 * - Lenis smooth scroll
 * - [data-reveal]  -> adds .is-in when it enters the viewport
 * - [data-speed]   -> vertical parallax (--py), relative to the parent's position
 * - [data-mouse]   -> pointer parallax (--mx/--my)
 * - [data-progress="sticky" | "pass"] -> writes --progress (0..1) for scroll-linked CSS
 * - Scroll is locked until every visible hero book has finished its drop-in (see `hero-drop` in globals.css)
 * Does nothing beyond revealing content when the user prefers reduced motion.
 */
export function MotionController() {
  useEffect(() => {
    const root = document.documentElement;
    const reveal = document.querySelectorAll("[data-reveal]");

    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      reveal.forEach((el) => el.classList.add("is-in"));
      return;
    }

    root.dataset.motion = "";

    const io = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (!entry.isIntersecting) continue;
          entry.target.classList.add("is-in");
          io.unobserve(entry.target);
        }
      },
      { rootMargin: "0px 0px -12% 0px" },
    );
    reveal.forEach((el) => io.observe(el));

    // stopInertiaOnNavigate: clicking a link to another page stops the glide at once. Without it the old
    // instance keeps easing toward its target and drags the new page down to where the old one was.
    const lenis = new Lenis({ autoRaf: true, anchors: true, stopInertiaOnNavigate: true });

    // Hold scroll until the hero books have landed (skipped when the page is restored mid-scroll).
    const heroItems = Array.from(document.querySelectorAll<HTMLElement>(".hero__item")).filter((el) => el.offsetParent !== null);
    const landed = new Set<HTMLElement>();
    let introDone = heroItems.length === 0 || window.scrollY > 50;
    let introTimer = 0;
    const finishIntro = () => {
      if (introDone) return;
      introDone = true;
      window.clearTimeout(introTimer);
      lenis.start();
    };
    const onLanded = (e: AnimationEvent) => {
      if (e.animationName !== "hero-drop") return;
      landed.add(e.currentTarget as HTMLElement);
      if (landed.size >= heroItems.length) finishIntro();
    };
    if (!introDone) {
      lenis.stop();
      heroItems.forEach((el) => el.addEventListener("animationend", onLanded));
      introTimer = window.setTimeout(finishIntro, 9000); // safety net if an animation never reports back
    }

    // The menu panel asks for the page to stop scrolling while it is open.
    const onScrollLock = (e: Event) => {
      if ((e as CustomEvent<boolean>).detail) lenis.stop();
      else if (introDone) lenis.start();
    };
    window.addEventListener("of:scroll-lock", onScrollLock);

    // A page asks for a glide to one of its elements (detail: a selector). Cancelling the event tells
    // the sender it was handled; it is not when the intro still holds the scroll.
    const onScrollTo = (e: Event) => {
      if (!introDone) return;
      e.preventDefault();
      lenis.scrollTo((e as CustomEvent<string>).detail);
    };
    window.addEventListener("of:scroll-to", onScrollTo);

    const speedEls = Array.from(document.querySelectorAll<HTMLElement>("[data-speed]"));
    const mouseEls = Array.from(document.querySelectorAll<HTMLElement>("[data-mouse]"));
    const progressEls = Array.from(document.querySelectorAll<HTMLElement>("[data-progress]"));

    let targetX = 0;
    let targetY = 0;
    let mouseX = 0;
    let mouseY = 0;
    const onMove = (e: PointerEvent) => {
      targetX = e.clientX / window.innerWidth - 0.5;
      targetY = e.clientY / window.innerHeight - 0.5;
    };
    window.addEventListener("pointermove", onMove, { passive: true });

    let frame = 0;
    const tick = () => {
      const vh = window.innerHeight;

      for (const el of speedEls) {
        const box = (el.parentElement ?? el).getBoundingClientRect();
        if (box.bottom < -vh || box.top > vh * 2) continue;
        const offset = box.top + box.height / 2 - vh / 2;
        el.style.setProperty("--py", `${(-offset * Number(el.dataset.speed) * 0.25).toFixed(1)}px`);
      }

      mouseX += (targetX - mouseX) * 0.07;
      mouseY += (targetY - mouseY) * 0.07;
      for (const el of mouseEls) {
        const f = Number(el.dataset.mouse) * 60;
        el.style.setProperty("--mx", `${(mouseX * f).toFixed(1)}px`);
        el.style.setProperty("--my", `${(mouseY * f).toFixed(1)}px`);
      }

      for (const el of progressEls) {
        const box = el.getBoundingClientRect();
        const raw = el.dataset.progress === "sticky" ? -box.top / (box.height - vh) : (vh - box.top) / (box.height + vh);
        el.style.setProperty("--progress", Math.min(1, Math.max(0, raw)).toFixed(4));
      }

      frame = requestAnimationFrame(tick);
    };
    frame = requestAnimationFrame(tick);

    return () => {
      cancelAnimationFrame(frame);
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("of:scroll-lock", onScrollLock);
      window.removeEventListener("of:scroll-to", onScrollTo);
      window.clearTimeout(introTimer);
      heroItems.forEach((el) => el.removeEventListener("animationend", onLanded));
      io.disconnect();
      lenis.destroy();
      delete root.dataset.motion;
    };
  }, []);

  return null;
}
