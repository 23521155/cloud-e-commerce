import { CARD_ROWS } from "@/components/cart/Basket";

/**
 * Blank basket card shown while the basket is read: by loading.tsx while the server renders, then
 * by the views until the browser has read localStorage and fetched the books.
 */
export function BasketSkeleton() {
  return (
    <div className="basket__layout" aria-busy="true">
      <div className="basket__fan" aria-hidden="true" />

      <div>
        <section className="lib-card" aria-label="Basket card, loading">
          <div className="lib-card__frame">
            <h2 className="lib-card__head">Basket card</h2>
            <p className="sr-only" role="status">
              Reading your basket…
            </p>
            <table className="lib-card__table" aria-hidden="true">
              <tbody>
                {Array.from({ length: CARD_ROWS }, (_, i) => (
                  <tr key={i} className="lib-card__blank">
                    <td className="lib-card__no">{i + 1}.</td>
                    <td />
                    <td className="lib-card__col-binding" />
                    <td />
                  </tr>
                ))}
              </tbody>
            </table>
            <div className="lib-card__foot" aria-hidden="true">
              <span className="lib-card__marks">❦ ❦ ❦</span>
              <p className="lib-card__subtotal m-0">
                <span className="small-upper">Subtotal</span>
                <span className="lib-card__total">—</span>
              </p>
            </div>
          </div>
        </section>
      </div>
    </div>
  );
}
