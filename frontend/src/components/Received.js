import {useEffect, useState} from 'preact/hooks';
import {OpenReceiveFolder, ReceiveDir, ReceivedFiles} from '../../bindings/github.com/TobyPinfold/deck-drop/service.js';
import {errorMessage, formatBytes} from '../format.js';
import {html} from '../html.js';

const refreshMs = 2000;

// settingsVersion changes after settings are saved, since the receive folder
// and its contents may have changed.
export function Received({settingsVersion}) {
    const [files, setFiles] = useState([]);
    const [directory, setDirectory] = useState('');
    const [error, setError] = useState('');

    const refreshFiles = () => ReceivedFiles().then((received) => setFiles(received ?? []), () => {});

    useEffect(() => {
        ReceiveDir().then(setDirectory);
        refreshFiles();
        const timer = setInterval(refreshFiles, refreshMs);
        return () => clearInterval(timer);
    }, [settingsVersion]);

    return html`
        <div class="panel">
            <button class="action" type="button" onClick=${() => OpenReceiveFolder().then(
                () => setError(''),
                (failure) => setError(`Could not open the folder: ${errorMessage(failure)}`))}>Open folder</button>
            <p class="hint">${error || directory}</p>
        </div>
        <ul class="received">
            ${files.map((file) => html`
                <li class="received-file" key=${file.name}>
                    <span>${file.name}</span>
                    <span class="transfer-detail">${formatBytes(file.size)} · ${new Date(file.modified).toLocaleString()}</span>
                </li>`)}
        </ul>
        ${files.length === 0 && html`<p class="empty">Nothing received yet.</p>`}`;
}
