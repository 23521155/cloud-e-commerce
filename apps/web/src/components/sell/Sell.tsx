"use client";

import { useState } from "react";
import { SubmitButton } from "@/components/auth/AuthClient";
import { DotButton } from "@/components/ui/DotButton";

const BINDINGS = ["Hardcover", "Paperback", "Other"];
/** Usual bookseller grades, best first */
const GRADES = ["Fine", "Very good", "Good", "Fair", "Poor", "Not sure"];

function Field({ label, name, hint, className = "", ...input }: { label: string; name: string; hint?: string; className?: string } & React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <label className={`co__field content-start ${className}`}>
      <span className="co__label small-upper">{label}</span>
      <input name={name} className="co__input" {...input} />
      {hint && <span className="co__hint">{hint}</span>}
    </label>
  );
}

function Select({ label, name, options, placeholder }: { label: string; name: string; options: string[]; placeholder: string }) {
  return (
    <label className="co__field content-start">
      <span className="co__label small-upper">{label}</span>
      <select name={name} className="co__input co__select" defaultValue="" required>
        <option value="" disabled>
          {placeholder}
        </option>
        {options.map((o) => (
          <option key={o}>{o}</option>
        ))}
      </select>
    </label>
  );
}

function Notes({ label, name, hint, placeholder, required = false }: { label: string; name: string; hint?: string; placeholder: string; required?: boolean }) {
  return (
    <label className="co__field co__full">
      <span className="co__label small-upper">{label}</span>
      <textarea name={name} className="co__input sell__notes" rows={3} placeholder={placeholder} required={required} />
      {hint && <span className="co__hint">{hint}</span>}
    </label>
  );
}

/**
 * Photo picker with its own English button and status. The native file input labels itself in the
 * browser's language ("Chọn tệp" on a Vietnamese system), so it stays hidden behind this one.
 */
function PhotoField() {
  const [count, setCount] = useState(0);
  return (
    <div className="co__field co__full">
      <span className="co__label small-upper" id="photos-label">
        Photos
      </span>
      <label className="sell__file">
        <input name="photos" type="file" accept="image/*" multiple className="sr-only" aria-labelledby="photos-label" onChange={(e) => setCount(e.currentTarget.files?.length ?? 0)} />
        <span className="sell__file-btn small-upper">Choose photos</span>
        <span className="sell__file-status" aria-live="polite">
          {count === 0 ? "No photos chosen" : `${count} ${count === 1 ? "photo" : "photos"} chosen`}
        </span>
      </label>
      <span className="co__hint">Optional. The cover, the spine and any damage help most.</span>
    </div>
  );
}

/** A section of the form drawn as a card-index card, like checkout. */
function Card({ title, note, children }: { title: string; note?: string; children: React.ReactNode }) {
  return (
    <fieldset className="co__card">
      <legend className="sr-only">{title}</legend>
      <div className="catalogue__card-head small-upper" aria-hidden="true">
        <span>{title}</span>
        {note && <span>{note}</span>}
      </div>
      <div className="co__grid">{children}</div>
    </fieldset>
  );
}

const STEPS = [
  { title: "Describe the copy", text: "Tell us what you have, faults first. Photos help but are not required." },
  { title: "We reply with an offer", text: "Someone at the shop reads every description and answers by email." },
  { title: "You decide", text: "If the offer suits you, we arrange the rest together. If not, nothing more happens." },
];

function Steps() {
  return (
    <ol className="sell__steps">
      {STEPS.map((s) => (
        <li key={s.title}>
          <span className="sell__step-title">{s.title}</span>
          <span className="sell__step-text">{s.text}</span>
        </li>
      ))}
    </ol>
  );
}

/** Sell-to-us: the seller describes one copy and leaves contact details; the shop replies with an offer. */
export function Sell({ action, sent }: { action: (formData: FormData) => Promise<void>; sent: boolean }) {
  return (
    <div className="sell__layout">
      <div className="sell__aside">
        <Steps />
      </div>

      {sent ? (
        <section className="co__card sell__sent" aria-labelledby="sell-sent-title">
          <div className="catalogue__card-head small-upper" aria-hidden="true">
            <span>Received</span>
            <span>Marginalleya</span>
          </div>
          <h2 id="sell-sent-title" className="sell__sent-title">
            Your copy is on our desk
          </h2>
          <p className="m-0 auth__sent" role="status">
            Thank you. We read every description and will reply by email with an offer, or to ask about anything we could not tell from it.
          </p>
          <DotButton href="/sell" className="mt-8">
            Describe another book
          </DotButton>
        </section>
      ) : (
        <form action={action} className="co">
          <Card title="The book" note="One copy per form">
            <Field label="Title" name="title" required className="co__full" />
            <Field label="Author" name="author" required />
            <Field label="Year" name="year" inputMode="numeric" placeholder="e.g. 1943" hint="As printed on the title page, if you can find it." />
            <Field label="Publisher or edition" name="edition" className="co__full" placeholder="e.g. Harper & Brothers, first edition" />
            <Select label="Binding" name="binding" options={BINDINGS} placeholder="Choose one" />
            <Select label="Condition" name="grade" options={GRADES} placeholder="Your best guess" />
          </Card>

          <Card title="Its condition" note="Faults first">
            <Notes label="Faults" name="faults" required placeholder="Torn jacket, foxing, a loose page, writing inside…" hint="Write “none I can see” if it really has none." />
            <Notes label="What is good about it" name="virtues" placeholder="Signed, original jacket, a story behind it…" />
            <PhotoField />
            <Field label="Price you have in mind" name="asking" inputMode="numeric" className="co__full" placeholder="Optional, in VND" hint="Leave empty if you would rather hear our offer first." />
          </Card>

          <Card title="About you" note="So we can reply">
            <Field label="Name" name="name" autoComplete="name" required className="co__full" />
            <Field label="Email" name="email" type="email" autoComplete="email" required placeholder="you@example.com" />
            <Field label="Phone" name="phone" type="tel" autoComplete="tel" hint="Optional." />
          </Card>

          <div>
            <SubmitButton pending="Sending…">Send for an offer</SubmitButton>
          </div>
        </form>
      )}
    </div>
  );
}
