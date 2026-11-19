"use client";

import Link from "next/link";
import { useState, type ReactNode } from "react";

export type SubjectChip = { slug: string; name: string; href: string; active: boolean };

/** Subjects shown before the reader asks for more: on a phone the full list runs a screen long. */
const SHOWN = 6;

/**
 * The subject row of the catalogue filters. Only the first few subjects (and the one picked) show
 * until the reader opens the rest. `lead` and `trail` are the chips around them ("All subjects",
 * "Rare only").
 */
export function SubjectFilter({ chips, lead, trail }: { chips: SubjectChip[]; lead: ReactNode; trail: ReactNode }) {
  const [open, setOpen] = useState(false);
  const hidden = chips.length - SHOWN;

  return (
    <ul id="subject-filter" className="catalogue__row" aria-label="Subject">
      {lead}
      {chips.map((s, i) =>
        open || i < SHOWN || s.active ? (
          <li key={s.slug}>
            <Link href={s.href} scroll={false} className={`catalogue__chip ${s.active ? "is-active" : ""}`} aria-current={s.active ? "true" : undefined}>
              {s.name}
            </Link>
          </li>
        ) : null,
      )}
      {hidden > 0 && (
        <li>
          <button type="button" className="catalogue__chip catalogue__more" aria-expanded={open} aria-controls="subject-filter" onClick={() => setOpen(!open)}>
            {open ? "Fewer" : `+ ${hidden} more`}
          </button>
        </li>
      )}
      {trail}
    </ul>
  );
}
