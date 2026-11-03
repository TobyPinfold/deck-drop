import './style.css';
import {startNavigation} from './input.js';
import {Events} from '@wailsio/runtime';
import {
    ChooseFiles, Devices, Info, OpenReceiveFolder, PauseTransfer, ReceiveDir, ReceivedFiles, ResumeTransfer, SendFiles, Transfers,
} from '../bindings/github.com/TobyPinfold/deck-drop/service.js';

const status = document.querySelector('#status');
const deviceList = document.querySelector('#devices');
const emptyMessage = document.querySelector('#empty');
const chooseButton = document.querySelector('#choose');
const hint = document.querySelector('#hint');
const transferList = document.querySelector('#transfers');
const transfersSection = document.querySelector('#transfers-section');
const openFolderButton = document.querySelector('#open-folder');
const receiveDirLabel = document.querySelector('#receive-dir');
const receivedList = document.querySelector('#received');
const receivedEmpty = document.querySelector('#received-empty');

const receivedRefreshMs = 2000;

let selectedName = null;
let knownDevices = [];

// Tiles are rebuilt on every update, so remember which one had focus.
function focusedDeviceName() {
    return document.activeElement?.dataset?.device ?? null;
}

function focusDeviceTile(name) {
    const tiles = [...deviceList.querySelectorAll('.tile')];
    const tile = tiles.find((candidate) => candidate.dataset.device === name) ?? null;
    tile?.focus();
    return tile !== null;
}

function renderDevices(devices) {
    knownDevices = devices;
    const focusedName = focusedDeviceName();
    const nothingFocused = document.activeElement === document.body;
    if (!devices.some((device) => device.name === selectedName)) {
        selectedName = null;
    }

    deviceList.replaceChildren(...devices.map((device) => {
        const tile = document.createElement('button');
        tile.className = 'tile';
        tile.type = 'button';
        tile.dataset.device = device.name;
        tile.setAttribute('aria-pressed', String(device.name === selectedName));
        tile.addEventListener('click', () => {
            selectedName = device.name;
            renderDevices(knownDevices);
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

    if (focusedName !== null) {
        focusDeviceTile(focusedName);
    } else if (nothingFocused) {
        deviceList.querySelector('.tile')?.focus();
    }
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
    transfersSection.hidden = transfers.length === 0;
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

let lastReceived = '';

function renderReceived(files) {
    // Skip rebuilding when nothing changed, as this runs on a timer.
    const snapshot = JSON.stringify(files);
    if (snapshot === lastReceived) {
        return;
    }
    lastReceived = snapshot;

    receivedList.replaceChildren(...files.map((file) => {
        const item = document.createElement('li');
        item.className = 'received-file';
        const name = document.createElement('span');
        name.textContent = file.name;
        const detail = document.createElement('span');
        detail.className = 'transfer-detail';
        detail.textContent = `${formatBytes(file.size)} · ${new Date(file.modified).toLocaleString()}`;
        item.append(name, detail);
        return item;
    }));
    receivedEmpty.hidden = files.length > 0;
}

function refreshReceived() {
    ReceivedFiles().then((files) => renderReceived(files ?? []), () => {});
}

openFolderButton.addEventListener('click', () => {
    OpenReceiveFolder().catch((error) => {
        receiveDirLabel.textContent = `Could not open the folder: ${error.message ?? error}`;
    });
});

ReceiveDir().then((directory) => {
    receiveDirLabel.textContent = directory;
});
refreshReceived();
setInterval(refreshReceived, receivedRefreshMs);

// B goes back to the device list, which is where every send starts.
startNavigation({
    onBack: () => {
        if (!focusDeviceTile(selectedName)) {
            deviceList.querySelector('.tile')?.focus();
        }
        window.scrollTo({top: 0});
    },
});
