import {useEffect} from 'preact/hooks';
import {html} from '../html.js';

// Tiles are keyed by device name, so a re-render keeps the focused tile.
export function Devices({devices, pairedFingerprints, selectedName, onSelect}) {
    // Give the first tile focus once, so the gamepad has somewhere to start.
    useEffect(() => {
        if (document.activeElement === document.body) {
            document.querySelector('.tile')?.focus();
        }
    }, [devices.length > 0]);

    return html`
        <h2>Nearby devices</h2>
        <ul class="devices">
            ${devices.map((device) => html`
                <li key=${device.name}>
                    <button class="tile" type="button" data-device=${device.name}
                            aria-pressed=${device.name === selectedName}
                            onClick=${() => onSelect(device.name)}>
                        <span class="tile-name">${device.name}</span>
                        <span class="tile-address">${device.host}:${device.port}</span>
                        <span class="tile-badge">${pairedFingerprints.has(device.fingerprint) ? 'Paired' : 'Not paired'}</span>
                    </button>
                </li>`)}
        </ul>
        ${devices.length === 0 && html`<p class="empty">Looking for paired devices. To pair a new one, turn on pairing mode below.</p>`}`;
}
