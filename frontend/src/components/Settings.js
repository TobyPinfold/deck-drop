import {useEffect, useState} from 'preact/hooks';
import {ChooseFolder, SaveSettings, Settings as LoadSettings} from '../../bindings/github.com/TobyPinfold/deck-drop/service.js';
import {errorMessage} from '../format.js';
import {Toggle} from './Toggle.js';
import {html} from '../html.js';

export function Settings({onSaved}) {
    const [deviceName, setDeviceName] = useState('');
    const [port, setPort] = useState('');
    const [receiveDir, setReceiveDir] = useState('');
    const [hidden, setHidden] = useState(false);
    const [message, setMessage] = useState('');

    useEffect(() => {
        LoadSettings().then((settings) => {
            setDeviceName(settings.deviceName);
            setPort(settings.port);
            setReceiveDir(settings.receiveDir);
            setHidden(settings.hidden);
        });
    }, []);

    async function chooseFolder() {
        const folder = await ChooseFolder();
        if (folder) {
            setReceiveDir(folder);
        }
    }

    async function save(event) {
        event.preventDefault();
        setMessage('Saving…');
        try {
            await SaveSettings({deviceName, receiveDir, port: Number(port), hidden});
            setMessage('Saved');
            onSaved();
        } catch (error) {
            setMessage(`Could not save: ${errorMessage(error)}`);
        }
    }

    return html`
        <h2>Settings</h2>
        <form class="panel settings" onSubmit=${save}>
            <label class="field">
                <span>Device name</span>
                <input type="text" autocomplete="off" spellcheck=${false} required
                       value=${deviceName} onInput=${(event) => setDeviceName(event.target.value)} />
            </label>
            <label class="field">
                <span>Port</span>
                <input type="number" min="1" max="65535" required
                       value=${port} onInput=${(event) => setPort(event.target.value)} />
            </label>
            <${Toggle} label="Hide this device from the network" checked=${hidden} onChange=${setHidden} />
            <p class="hint">Paired devices that know its address can still send.</p>
            <div class="field">
                <span>Receive folder</span>
                <div class="folder">
                    <code>${receiveDir}</code>
                    <button class="action" type="button" onClick=${chooseFolder}>Change folder</button>
                </div>
            </div>
            <div class="form-actions">
                <button class="action primary" type="submit">Save</button>
                <p class="hint" role="status">${message}</p>
            </div>
        </form>`;
}
