import {useEffect, useState} from 'preact/hooks';
import {ChooseFolder, Info, SaveSettings, Settings as LoadSettings} from '../../bindings/github.com/TobyPinfold/deck-drop/service.js';
import {errorMessage} from '../format.js';
import {Toggle} from './Toggle.js';
import {html} from '../html.js';

// Every change applies as soon as it is made. Text fields apply when they
// lose focus or Enter is pressed. If a change is rejected, the field goes back
// to the value in effect.
export function Settings({onSaved}) {
    const [current, setCurrent] = useState(null);
    const [deviceName, setDeviceName] = useState('');
    const [port, setPort] = useState('');
    const [message, setMessage] = useState('');
    const [version, setVersion] = useState('');

    function show(settings) {
        setCurrent(settings);
        setDeviceName(settings.deviceName);
        setPort(String(settings.port));
    }

    useEffect(() => {
        LoadSettings().then(show);
        Info().then((info) => setVersion(info.version));
    }, []);

    async function apply(changes) {
        const next = {...current, ...changes};
        if (JSON.stringify(next) === JSON.stringify(current)) {
            return;
        }
        setMessage('Applying…');
        try {
            await SaveSettings(next);
            show(next);
            setMessage('');
            onSaved();
        } catch (error) {
            show(current);
            setMessage(`Could not apply: ${errorMessage(error)}`);
        }
    }

    async function chooseFolder() {
        const folder = await ChooseFolder();
        if (folder) {
            apply({receiveDir: folder});
        }
    }

    if (!current) {
        return null;
    }

    return html`
        <div class="settings">
            <label class="field">
                <span>Device name</span>
                <input type="text" autocomplete="off" spellcheck=${false}
                       value=${deviceName} onInput=${(event) => setDeviceName(event.target.value)}
                       onChange=${(event) => apply({deviceName: event.target.value.trim()})} />
            </label>
            <label class="field">
                <span>Port</span>
                <input type="number" min="1" max="65535"
                       value=${port} onInput=${(event) => setPort(event.target.value)}
                       onChange=${(event) => apply({port: Number(event.target.value)})} />
            </label>
            <${Toggle} label="Hide this device from the network" checked=${current.hidden}
                       onChange=${(enabled) => apply({hidden: enabled})} />
            <p class="hint">Paired devices that know its address can still send.</p>
            <div class="field">
                <span>Receive folder</span>
                <div class="folder">
                    <code>${current.receiveDir}</code>
                    <button class="action" type="button" onClick=${chooseFolder}>Change folder</button>
                </div>
            </div>
            <p class="hint" role="status">${message}</p>
            ${version && html`<p class="hint version">DeckDrop ${version}</p>`}
        </div>`;
}
