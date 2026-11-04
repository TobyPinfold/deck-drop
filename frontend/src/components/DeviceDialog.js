import {useEffect} from 'preact/hooks';
import {Events} from '@wailsio/runtime';
import {ChooseFiles, SendFiles} from '../../bindings/github.com/TobyPinfold/deck-drop/service.js';
import {errorMessage} from '../format.js';
import {Modal} from './Modal.js';
import {PairedControls} from './Paired.js';
import {html} from '../html.js';

// Everything you can do with a device you picked: choose files to send, or
// change how it is trusted. Dropped files go straight to it.
export function DeviceDialog({device, peer, onClose, notify}) {
    async function sendFiles(paths) {
        if (!paths || paths.length === 0) {
            return;
        }
        try {
            await SendFiles(device.name, paths);
            notify(`Sending to ${device.name}`);
            onClose();
        } catch (error) {
            notify(`Could not send: ${errorMessage(error)}`);
        }
    }

    useEffect(() => Events.On('files-dropped', (event) => sendFiles(event.data)), [device.name]);

    return html`
        <${Modal} title=${device.name} onClose=${onClose}>
            <p class="hint">Choose files to send, or drop them on this window.</p>
            <div class="modal-actions">
                <button class="action primary" type="button" data-autofocus
                        onClick=${async () => sendFiles(await ChooseFiles())}>Send files</button>
            </div>
            <${PairedControls} peer=${peer} onUnpaired=${onClose} />
        <//>`;
}
