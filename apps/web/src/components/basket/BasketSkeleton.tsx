import { CARD_ROWS, Seal } from "@/components/cart/Basket";

/**
 * Blank basket card shown until the browser has read localStorage and fetched the books. It has
 * the same parts as the real card (fields, seal, column heads, ruled lines), so the card does not
 * change size or grow new parts when the books arrive.
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

            <div className="lib-card__top" aria-hidden="true">
              <dl className="lib-card__fields">
                <div>
                  <dt>Date</dt>
                  <dd>—</dd>
                </div>
                <div>
                  <dt>Copies</dt>
                  <dd>—</dd>
                </div>
                <div>
                  <dt>Shipping</dt>
                  <dd>Calculated at checkout</dd>
                </div>
              </dl>
              <Seal />
            </div>

            <table className="lib-card__table" aria-hidden="true">
              <thead>
                <tr>
                  <th scope="col">No.</th>
                  <th scope="col">Copy</th>
                  <th scope="col" className="lib-card__col-binding">
                    Binding
                  </th>
                  <th scope="col" className="text-right">
                    Price
                  </th>
                </tr>
              </thead>
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
