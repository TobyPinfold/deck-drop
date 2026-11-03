import './style.css';
import {Events} from '@wailsio/runtime';
import {Devices, Info} from '../bindings/github.com/TobyPinfold/deck-drop/service.js';

const status = document.querySelector('#status');
const deviceList = document.querySelector('#devices');
const emptyMessage = document.querySelector('#empty');

let selectedName = null;

function renderDevices(devices) {
    if (!devices.some((device) => device.name === selectedName)) {
        selectedName = null;
    }

    deviceList.replaceChildren(...devices.map((device) => {
        const tile = document.createElement('button');
        tile.className = 'tile';
        tile.type = 'button';
        tile.setAttribute('aria-pressed', String(device.name === selectedName));
        tile.addEventListener('click', () => {
            selectedName = device.name;
            renderDevices(devices);
            deviceList.querySelector('[aria-pressed="true"]').focus();
        });

        // Device names come from the network, so only ever set them as text.
        const name = document.createElement('span');
        name.className = 'tile-name';
        name.textContent = device.name;
        const address = document.createElement('span');
        address.className = 'tile-address';
        address.textContent = `${device.host}:${device.port}`;
        tile.append(name, address);

        const item = document.createElement('li');
        item.append(tile);
        return item;
    }));
    emptyMessage.hidden = devices.length > 0;
}

Info().then((info) => {
    status.textContent = info.running
        ? `Receiving on port ${info.port}`
        : `Backend not running: ${info.error}`;
});

Events.On('devices', (event) => renderDevices(event.data ?? []));
Devices().then((devices) => renderDevices(devices ?? []));
