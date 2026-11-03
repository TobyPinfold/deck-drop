import {useEffect, useRef} from 'preact/hooks';
import {RespondToOffer} from '../../bindings/github.com/TobyPinfold/deck-drop/service.js';
import {offerSummary} from '../format.js';
import {html} from '../html.js';

// Shown at the top whenever a paired device wants to send files. Only the
// oldest offer is shown, and Accept is focused so A answers it.
export function Offers({offers}) {
    const accept = useRef(null);
    const current = offers[0];

    useEffect(() => {
        accept.current?.focus();
    }, [current?.id]);

    if (!current) {
        return null;
    }
    const respond = (answer) => RespondToOffer(current.id, answer).catch(() => {});
    return html`
        <section class="offer" role="alertdialog" aria-label="Incoming files">
            <p><strong>${current.device}</strong> wants to send ${offerSummary(current)}</p>
            ${current.count > 1 && html`<p class="hint">${current.names.join(', ')}${current.count > current.names.length ? '…' : ''}</p>`}
            <div class="offer-actions">
                <button class="action" type="button" ref=${accept} onClick=${() => respond(true)}>Accept</button>
                <button class="action" type="button" data-offer-decline onClick=${() => respond(false)}>Decline</button>
            </div>
            ${offers.length > 1 && html`<p class="hint">${offers.length - 1} more waiting</p>`}
        </section>`;
}
