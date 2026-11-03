import {useEffect, useState} from 'preact/hooks';
import {Events} from '@wailsio/runtime';
import {Devices as ListDevices, Info, Offers as ListOffers, PairedDevices, Transfers as ListTransfers} from '../bindings/github.com/TobyPinfold/deck-drop/service.js';
import {Devices} from './components/Devices.js';
import {Offers} from './components/Offers.js';
import {Pairing} from './components/Pairing.js';
import {Received} from './components/Received.js';
import {Send} from './components/Send.js';
import {Settings} from './components/Settings.js';
import {Transfers} from './components/Transfers.js';
import {html} from './html.js';
import {startNavigation} from './input.js';

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
    const [requestedName, setRequestedName] = useState(null);
    const [hint, setHint] = useState('Select a device, then choose or drop files to send.');
    const [settingsVersion, setSettingsVersion] = useState(0);
    const status = useStatus(settingsVersion);

    // A device that disappears is no longer selected.
    const selectedDevice = devices.find((device) => device.name === requestedName) ?? null;
    const selectedName = selectedDevice?.name ?? null;
    const pairedFingerprints = new Set(paired.map((peer) => peer.fingerprint));
    const selectedIsPaired = selectedDevice !== null && pairedFingerprints.has(selectedDevice.fingerprint);

    // B goes back to the device list, which is where every send starts.
    useEffect(() => {
        startNavigation({
            onBack: () => {
                // An incoming offer has to be answered first; B declines it.
                const decline = document.querySelector('[data-offer-decline]');
                if (decline) {
                    decline.click();
                    return;
                }
                const tiles = [...document.querySelectorAll('.tile')];
                const target = tiles.find((tile) => tile.getAttribute('aria-pressed') === 'true') ?? tiles[0];
                target?.focus();
                window.scrollTo({top: 0});
            },
        });
    }, []);

    return html`
        <header>
            <h1>DeckDrop</h1>
            <p>${status}</p>
        </header>
        <${Offers} offers=${offers} />
        <main class="columns">
            <div class="column">
                <${Devices} devices=${devices} pairedFingerprints=${pairedFingerprints} selectedName=${selectedName} onSelect=${setRequestedName} />
                <${Send} selectedName=${selectedName} canSend=${selectedIsPaired} hint=${hint} setHint=${setHint} />
                <${Pairing} paired=${paired} selectedDevice=${selectedDevice} selectedIsPaired=${selectedIsPaired} />
            </div>
            <div class="column">
                <${Transfers} transfers=${transfers} setHint=${setHint} />
                <${Received} settingsVersion=${settingsVersion} />
                <${Settings} onSaved=${() => setSettingsVersion((version) => version + 1)} />
            </div>
        </main>`;
}
