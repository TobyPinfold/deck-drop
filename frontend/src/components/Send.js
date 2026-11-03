import {useEffect} from 'preact/hooks';
import {Events} from '@wailsio/runtime';
import {ChooseFiles, SendFiles} from '../../bindings/github.com/TobyPinfold/deck-drop/service.js';
import {errorMessage} from '../format.js';
import {html} from '../html.js';

export function Send({selectedName, hint, setHint}) {
    async function sendFiles(paths) {
        if (!paths || paths.length === 0) {
            return;
        }
        if (selectedName === null) {
            setHint('Select a device first, then drop your files again.');
            return;
        }
        try {
            await SendFiles(selectedName, paths);
            setHint(`Sending to ${selectedName}`);
        } catch (error) {
            setHint(`Could not send: ${errorMessage(error)}`);
        }
    }

    // Resubscribe when the selection changes so a drop uses the current device.
    useEffect(() => Events.On('files-dropped', (event) => sendFiles(event.data)), [selectedName]);

    return html`
        <h2>Send</h2>
        <div class="panel" data-file-drop-target>
            <button class="action" type="button" disabled=${selectedName === null}
                    onClick=${async () => sendFiles(await ChooseFiles())}>Choose files</button>
            <p class="hint">${hint}</p>
        </div>`;
}
