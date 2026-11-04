import {useEffect, useRef, useState} from 'preact/hooks';
import {Events} from '@wailsio/runtime';
import {Devices as ListDevices, Info, Offers as ListOffers, PairedDevices, Quit, Transfers as ListTransfers} from '../bindings/github.com/TobyPinfold/deck-drop/service.js';
import {DeviceDialog} from './components/DeviceDialog.js';
import {Devices} from './components/Devices.js';
import {Modal} from './components/Modal.js';
import {Offers} from './components/Offers.js';
import {PairedDialog} from './components/Paired.js';
import {PairDialog} from './components/Pairing.js';
import {Received} from './components/Received.js';
import {Settings} from './components/Settings.js';
import {Transfers} from './components/Transfers.js';
import {html} from './html.js';
import {activeDownloads} from './format.js';
import {startNavigation} from './input.js';

const NOTICE_MS = 6000;

// Subscribes to a backend event and loads the initial value.
function useBackendList(eventName, load) {
    const [items, setItems] = useState([]);
    useEffect(() => {
        load().then((initial) => setItems((current) => (current.length === 0 ? initial ?? [] : current)));
        return Events.On(eventName, (event) => setItems(event.data ?? []));
    }, []);
    return items;
}

function useStatus(settingsVersion) {
    const [status, setStatus] = useState('Starting…');
    useEffect(() => {
        Info().then((info) => setStatus(info.running
            ? `Receiving on port ${info.port}`
            : `Backend not running: ${info.error}`));
    }, [settingsVersion]);
    return status;
}

export function App() {
    const devices = useBackendList('devices', ListDevices);
    const transfers = useBackendList('transfers', ListTransfers);
    const paired = useBackendList('paired', PairedDevices);
    const offers = useBackendList('offers', ListOffers);
    // The open dialog: null, or {name: 'device' | 'pair' | 'paired' | 'received' | 'settings'}.
    const [dialog, setDialog] = useState(null);
    const [notice, setNotice] = useState('');
    const [settingsVersion, setSettingsVersion] = useState(0);
    const status = useStatus(settingsVersion);
    const noticeTimer = useRef(null);

    function notify(message) {
        setNotice(message);
        clearTimeout(noticeTimer.current);
        noticeTimer.current = setTimeout(() => setNotice(''), NOTICE_MS);
    }

    const downloads = activeDownloads(transfers);
    const close = () => setDialog(null);
    const pairedFingerprints = new Set(paired.map((peer) => peer.fingerprint));

    function openDevice(device) {
        if (pairedFingerprints.has(device.fingerprint)) {
            setDialog({name: 'device', deviceName: device.name});
        } else {
            notify(`Pair with ${device.name} first.`);
            setDialog({name: 'pair'});
        }
    }

    // B answers an incoming offer by declining it, otherwise it backs out of
    // the open dialog.
    useEffect(() => {
        startNavigation({
            onBack: () => {
                const decline = document.querySelector('[data-offer-decline]');
                const closeButtons = document.querySelectorAll('[data-modal-close]');
                (decline ?? closeButtons[closeButtons.length - 1])?.click();
            },
        });
    }, []);

    const dialogDevice = dialog?.name === 'device'
        ? devices.find((device) => device.name === dialog.deviceName)
        : undefined;
    const openPeer = dialogDevice && paired.find((peer) => peer.fingerprint === dialogDevice.fingerprint);
    // A device that goes away takes its dialog with it.
    useEffect(() => {
        if (dialog?.name === 'device' && !openPeer) {
            setDialog(null);
        }
    }, [dialog, openPeer]);

    const modalOpen = dialog !== null || offers.length > 0;

    return html`
        <div inert=${modalOpen}>
            <header>
                <button class="exit" type="button" aria-label="Exit" title="Exit" onClick=${() => Quit()}>✕</button>
                <h1>DeckDrop</h1>
                <p>${status}</p>
            </header>
            ${notice && html`<p class="notice" role="status">${notice}</p>`}
            <main>
                <${Devices} devices=${devices} pairedFingerprints=${pairedFingerprints} onOpen=${openDevice} />
                <nav class="actions" aria-label="Actions">
                    <button class="action primary" type="button" onClick=${() => setDialog({name: 'pair'})}>Pair a device</button>
                    <button class="action" type="button" onClick=${() => setDialog({name: 'paired'})}>Paired devices</button>
                    <button class="action" type="button" onClick=${() => setDialog({name: 'received'})}>
                        Received files
                        ${downloads > 0 && html`
                            <span class="busy" role="status" aria-label=${downloads === 1 ? 'Downloading a file' : `Downloading ${downloads} files`}>
                                <span class="spinner" aria-hidden="true"></span>${downloads > 1 && html`<span class="busy-count">${downloads}</span>`}
                            </span>`}
                    </button>
                    <button class="action" type="button" onClick=${() => setDialog({name: 'settings'})}>Settings</button>
                </nav>
                <${Transfers} transfers=${transfers} setHint=${notify} />
            </main>
        </div>
        ${dialog?.name === 'device' && dialogDevice && openPeer && html`
            <${DeviceDialog} device=${dialogDevice} peer=${openPeer} onClose=${close} notify=${notify} />`}
        ${dialog?.name === 'pair' && html`<${PairDialog} devices=${devices} paired=${paired} onClose=${close} notify=${notify} />`}
        ${dialog?.name === 'paired' && html`<${PairedDialog} paired=${paired} onClose=${close} />`}
        ${dialog?.name === 'received' && html`
            <${Modal} title="Received files" onClose=${close}><${Received} settingsVersion=${settingsVersion} /><//>`}
        ${dialog?.name === 'settings' && html`
            <${Modal} title="Settings" onClose=${close}>
                <${Settings} onSaved=${() => setSettingsVersion((version) => version + 1)} />
            <//>`}
        <${Offers} offers=${offers} />`;
}
