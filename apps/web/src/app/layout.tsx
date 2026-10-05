import type { Metadata } from "next";
import { IM_Fell_English } from "next/font/google";
import { SiteFooter } from "@/components/layout/SiteFooter";
import { SiteHeader } from "@/components/layout/SiteHeader";
import "./globals.css";

// Only a "latin" subset exists; it also covers the Vietnamese book titles via combining marks.
const fell = IM_Fell_English({
  variable: "--font-fell",
  subsets: ["latin"],
  weight: "400",
  style: ["normal", "italic"],
  display: "swap",
});

// Runs before first paint: skip the intro for returning visitors and reduced-motion users.
const PRELOAD_GATE = `try{var d=document.documentElement;if(sessionStorage.getItem("of-preloaded")||matchMedia("(prefers-reduced-motion: reduce)").matches){d.dataset.loaded="instant"}}catch(e){}`;

export const metadata: Metadata = {
  title: "Marginalleya — Used, new & rare books",
  description:
    "An online bookshop for used, new and rare books, each one described by the person who keeps it.",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="en"
      suppressHydrationWarning
      className={`${fell.variable} antialiased`}
    >
      <head>
        <script dangerouslySetInnerHTML={{ __html: PRELOAD_GATE }} />
        <noscript>
          <style>{`.preload{display:none}.hero__line>span{transform:none!important}.hero__item{opacity:1}`}</style>
        </noscript>
      </head>
      <body className="flex min-h-svh flex-col">
        <a
          href="#main"
          className="small-upper sr-only z-[200] bg-parchment-100 px-4 py-3 text-walnut-900 focus:not-sr-only focus:fixed focus:left-4 focus:top-4"
        >
          Skip to content
        </a>
        <SiteHeader />
        <main id="main" className="flex-1">
          {children}
        </main>
        <SiteFooter />
      </body>
    </html>
  );
}
