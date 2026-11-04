import {useEffect} from 'preact/hooks';
import {html} from '../html.js';

// Tiles are keyed by device name, so a re-render keeps the focused tile.
export function Devices({devices, pairedFingerprints, onOpen}) {
    // Give the first tile (or the first action) focus once, so the gamepad has somewhere to start.
    useEffect(() => {
        if (document.activeElement === document.body) {
            (document.querySelector('.tile') ?? document.querySelector('.actions button'))?.focus();
        }
    }, [devices.length > 0]);

    return html`
        <h2>Nearby devices</h2>
        <ul class="devices">
            ${devices.map((device) => html`
                <li key=${device.name}>
                    <button class="tile" type="button" data-device=${device.name} onClick=${() => onOpen(device)}>
                        <span class="tile-name">${device.name}</span>
                        <span class="tile-address">${device.host}:${device.port}</span>
                        <span class="tile-badge">${pairedFingerprints.has(device.fingerprint) ? 'Paired' : 'Not paired'}</span>
                    </button>
                </li>`)}
        </ul>
        ${devices.length === 0 && html`
            <p class="empty searching"><span class="radar" aria-hidden="true"></span>Looking for paired devices. Choose <strong>Pair a device</strong> to add one.</p>`}`;
}
