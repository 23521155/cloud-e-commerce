import Link from "next/link";
import { BookPhoto } from "@/components/ui/BookArt";
import { DotButton } from "@/components/ui/DotButton";
import { Logo } from "@/components/ui/Logo";

const LINKS = [
  { href: "/catalogue", label: "Catalogue" },
  { href: "/catalogue?rare=1", label: "Rare books" },
  { href: "/catalogue?sort=new", label: "New arrivals" },
  { href: "/sell", label: "Sell books" },
];

const PHOTOS = [
  { title: "The Good Earth", author: "Pearl S. Buck", color: "#3c1d0e", speed: "0.12" },
  { title: "Autobiography of Benjamin Franklin", author: "Benjamin Franklin", color: "#5c1a16", speed: "0.28" },
  { title: "Earth Abides", author: "George R. Stewart", color: "#24382e", speed: "0.18" },
];

export function SiteFooter() {
  return (
    <footer className="site-footer relative overflow-hidden px-[var(--gutter)] pb-8 pt-[clamp(100px,14vw,220px)]">
      <div className="grid items-end gap-14 lg:grid-cols-[1.5fr_1fr]">
        <div>
          <span className="small-upper text-gold-400">❦ Let&rsquo;s make something</span>
          <Link href="/sell" data-reveal className="footer__heading big-sans mt-6 block">
            <span className="line">
              <span>Got books</span>
            </span>
            <span className="line">
              <span className="em-italic text-gold-400">to sell?</span>
            </span>
          </Link>
        </div>
        <div className="flex flex-col items-start gap-6">
          <span className="small-upper text-parchment-300">Get started</span>
          <p className="m-0 max-w-[34ch] text-lg leading-relaxed text-parchment-300">
            List a single book or a whole library. You describe it, you set the price, readers find you.
          </p>
          <DotButton href="/sell" variant="solid">
            Start selling
          </DotButton>
        </div>
      </div>

      <div className="mt-[clamp(60px,8vw,120px)] grid grid-cols-3 gap-[clamp(10px,2vw,32px)]">
        {PHOTOS.map((photo) => (
          <div key={photo.title} data-speed={photo.speed}>
            <div className="aspect-[4/3] overflow-hidden rounded-[10px]">
              <BookPhoto title={photo.title} author={photo.author} color={photo.color} />
            </div>
          </div>
        ))}
      </div>

      <div className="mt-[clamp(48px,6vw,96px)] flex flex-wrap items-end justify-between gap-6 border-t border-parchment-100/15 pt-6">
        <nav aria-label="Footer" className="small-upper flex flex-wrap gap-x-8 gap-y-3">
          {LINKS.map((link) => (
            <Link key={link.href} href={link.href} className="under-hover text-parchment-200">
              {link.label}
            </Link>
          ))}
        </nav>
        <div className="flex items-center gap-4">
          <Logo className="h-9 w-auto text-gold-400" />
          <p className="small-upper m-0 text-parchment-300">© 2026 Marginalleya</p>
        </div>
      </div>
    </footer>
  );
}
