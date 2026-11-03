import './style.css';
import {Events} from '@wailsio/runtime';
import {ChooseFiles, Devices, Info, SendFiles, Transfers} from '../bindings/github.com/TobyPinfold/deck-drop/service.js';

const status = document.querySelector('#status');
const deviceList = document.querySelector('#devices');
const emptyMessage = document.querySelector('#empty');
const chooseButton = document.querySelector('#choose');
const hint = document.querySelector('#hint');
const transferList = document.querySelector('#transfers');

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
    chooseButton.disabled = selectedName === null;
}

function renderTransfers(transfers) {
    transferList.replaceChildren(...transfers.map((transfer) => {
        const item = document.createElement('li');
        item.className = `transfer ${transfer.status}`;

        // File names come from the user's disk, so only ever set them as text.
        const name = document.createElement('span');
        name.textContent = `${transfer.name} → ${transfer.device}`;
        const status = document.createElement('span');
        status.className = 'transfer-status';
        status.textContent = transfer.status === 'failed'
            ? `Failed: ${transfer.error}`
            : transfer.status;
        item.append(name, status);
        return item;
    }));
}

async function sendFiles(paths) {
    if (!paths || paths.length === 0) {
        return;
    }
    if (selectedName === null) {
        hint.textContent = 'Select a device first, then drop your files again.';
        return;
    }
    try {
        await SendFiles(selectedName, paths);
        hint.textContent = `Sending to ${selectedName}`;
    } catch (error) {
        hint.textContent = `Could not send: ${error.message ?? error}`;
    }
}

chooseButton.addEventListener('click', async () => sendFiles(await ChooseFiles()));

Info().then((info) => {
    status.textContent = info.running
        ? `Receiving on port ${info.port}`
        : `Backend not running: ${info.error}`;
});

Events.On('devices', (event) => renderDevices(event.data ?? []));
Events.On('transfers', (event) => renderTransfers(event.data ?? []));
Events.On('files-dropped', (event) => sendFiles(event.data));
Devices().then((devices) => renderDevices(devices ?? []));
Transfers().then((transfers) => renderTransfers(transfers ?? []));
