import {PauseTransfer, ResumeTransfer} from '../../bindings/github.com/TobyPinfold/deck-drop/service.js';
import {errorMessage, transferDetailText, transferPercent, transferStatusText} from '../format.js';
import {html} from '../html.js';

const actions = {
    queued: ['Pause', PauseTransfer],
    sending: ['Pause', PauseTransfer],
    paused: ['Resume', ResumeTransfer],
    failed: ['Retry', ResumeTransfer],
};

// Rows are keyed by id, so progress updates patch them in place and a focused
// pause button keeps its focus.
function TransferRow({transfer, setHint}) {
    const [label, action] = actions[transfer.status] ?? [];
    return html`
        <li class="transfer" data-status=${transfer.status}>
            <span class="transfer-name">${transfer.name} ${transfer.direction === 'receive' ? '←' : '→'} ${transfer.device}</span>
            <span class="transfer-status">${transferStatusText(transfer)}</span>
            <div class="bar"><div class="bar-fill" style=${{width: `${transferPercent(transfer)}%`}}></div></div>
            <span class="transfer-detail">${transferDetailText(transfer)}</span>
            <button class="action small" type="button" hidden=${action === undefined}
                    onClick=${() => action?.(transfer.id).catch((error) => {
                        setHint(`Could not change the transfer: ${errorMessage(error)}`);
                    })}>${label}</button>
        </li>`;
}

export function Transfers({transfers, setHint}) {
    if (transfers.length === 0) {
        return null;
    }
    return html`
        <section>
            <h2>Transfers</h2>
            <ul class="transfers">
                ${transfers.map((transfer) => html`<${TransferRow} key=${transfer.id} transfer=${transfer} setHint=${setHint} />`)}
            </ul>
        </section>`;
}
