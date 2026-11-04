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

    // Return to what had focus before the first offer appeared.
    const opener = useRef(null);
    useEffect(() => {
        if (current && !opener.current) {
            opener.current = document.activeElement;
        }
        if (!current && opener.current) {
            const previous = opener.current;
            opener.current = null;
            requestAnimationFrame(() => previous.isConnected && previous.focus());
        }
    }, [current === undefined]);

    if (!current) {
        return null;
    }
    const respond = (answer) => RespondToOffer(current.id, answer).catch(() => {});
    return html`
        <div class="modal-backdrop">
        <section class="modal offer" role="alertdialog" aria-modal="true" aria-label="Incoming files">
            <p><strong>${current.device}</strong> wants to send ${offerSummary(current)}</p>
            ${current.count > 1 && html`<p class="hint">${current.names.join(', ')}${current.count > current.names.length ? '…' : ''}</p>`}
            <div class="offer-actions">
                <button class="action primary" type="button" ref=${accept} onClick=${() => respond(true)}>Accept</button>
                <button class="action" type="button" data-offer-decline onClick=${() => respond(false)}>Decline</button>
            </div>
            ${offers.length > 1 && html`<p class="hint">${offers.length - 1} more waiting</p>`}
        </section>
        </div>`;
}
