import {useState} from 'preact/hooks';
import {SetAutoAccept, Unpair} from '../../bindings/github.com/TobyPinfold/deck-drop/service.js';
import {errorMessage} from '../format.js';
import {Modal} from './Modal.js';
import {Toggle} from './Toggle.js';
import {html} from '../html.js';

// Manages one paired device: whether it may send without asking, and removing it.
export function PairedControls({peer, onUnpaired}) {
    const [message, setMessage] = useState('');

    async function unpair() {
        try {
            await Unpair(peer.fingerprint);
            onUnpaired?.();
        } catch (error) {
            setMessage(`Could not unpair: ${errorMessage(error)}`);
        }
    }

    async function setAutoAccept(enabled) {
        try {
            await SetAutoAccept(peer.fingerprint, enabled);
            setMessage('');
        } catch (error) {
            setMessage(`Could not change the setting: ${errorMessage(error)}`);
        }
    }

    return html`
        <div class="paired-actions">
            <${Toggle} label="Always accept" checked=${peer.autoAccept} onChange=${setAutoAccept} />
            <button class="action" type="button" onClick=${unpair}>Unpair</button>
        </div>
        ${message && html`<p class="hint">${message}</p>`}`;
}

export function PairedDialog({paired, onClose}) {
    return html`
        <${Modal} title="Paired devices" onClose=${onClose}>
            <ul class="paired">
                ${paired.map((peer) => html`
                    <li key=${peer.fingerprint}>
                        <span>${peer.name}</span>
                        <${PairedControls} peer=${peer} />
                    </li>`)}
            </ul>
            ${paired.length === 0 && html`<p class="empty">No paired devices yet. Only paired devices can send files here.</p>`}
        <//>`;
}
