import './style.css';
import {Events} from '@wailsio/runtime';
import {ChooseFiles, Devices, Info, PauseTransfer, ResumeTransfer, SendFiles, Transfers} from '../bindings/github.com/TobyPinfold/deck-drop/service.js';

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

const transferRows = new Map();

function formatBytes(bytes) {
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let value = bytes;
    let unit = 0;
    while (value >= 1000 && unit < units.length - 1) {
        value /= 1000;
        unit++;
    }
    return `${unit === 0 ? value : value.toFixed(1)} ${units[unit]}`;
}

function createTransferRow(transfer) {
    const item = document.createElement('li');
    item.className = 'transfer';

    // File names come from the user's disk, so only ever set them as text.
    const name = document.createElement('span');
    name.className = 'transfer-name';
    name.textContent = `${transfer.name} → ${transfer.device}`;

    const status = document.createElement('span');
    status.className = 'transfer-status';

    const bar = document.createElement('div');
    bar.className = 'bar';
    const fill = document.createElement('div');
    fill.className = 'bar-fill';
    bar.append(fill);

    const detail = document.createElement('span');
    detail.className = 'transfer-detail';

    const toggle = document.createElement('button');
    toggle.className = 'action small';
    toggle.type = 'button';

    item.append(name, status, bar, detail, toggle);
    return {item, status, fill, detail, toggle, id: transfer.id, action: null};
}

// Rows are updated in place, because progress arrives several times a second
// and rebuilding them would drop the focus from the pause button.
function updateTransferRow(row, transfer) {
    const percent = transfer.total > 0 ? (transfer.sent / transfer.total) * 100 : 0;
    row.item.dataset.status = transfer.status;
    row.fill.style.width = `${transfer.status === 'done' ? 100 : percent}%`;
    row.status.textContent = transfer.status === 'failed'
        ? `Failed: ${transfer.error}`
        : transfer.status;

    row.detail.textContent = transfer.total > 0
        ? `${formatBytes(transfer.sent)} of ${formatBytes(transfer.total)}`
        + (transfer.status === 'sending' && transfer.speed > 0 ? ` · ${formatBytes(transfer.speed)}/s` : '')
        : '';

    const actions = {
        queued: ['Pause', PauseTransfer],
        sending: ['Pause', PauseTransfer],
        paused: ['Resume', ResumeTransfer],
        failed: ['Retry', ResumeTransfer],
    };
    const [label, action] = actions[transfer.status] ?? [];
    row.toggle.hidden = action === undefined;
    row.toggle.textContent = label ?? '';
    row.action = action ?? null;
}

function renderTransfers(transfers) {
    for (const transfer of transfers) {
        let row = transferRows.get(transfer.id);
        if (!row) {
            row = createTransferRow(transfer);
            row.toggle.addEventListener('click', () => row.action?.(row.id).catch((error) => {
                hint.textContent = `Could not change the transfer: ${error.message ?? error}`;
            }));
            transferRows.set(transfer.id, row);
            transferList.append(row.item);
        }
        updateTransferRow(row, transfer);
    }
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
