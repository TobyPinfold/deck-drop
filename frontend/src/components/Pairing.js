import {useEffect, useRef, useState} from 'preact/hooks';
import {
    CancelPairing,
    PairWith,
    SetPairingMode,
    StartPairing,
} from '../../bindings/github.com/TobyPinfold/deck-drop/service.js';
import {appendDigit, errorMessage, secondsLeft} from '../format.js';
import {Modal} from './Modal.js';
import {html} from '../html.js';

const CODE_LENGTH = 6;
const KEYPAD = ['1', '2', '3', '4', '5', '6', '7', '8', '9', '⌫', '0', 'Pair'];

// Shows a one-time code for another device to enter. The code is discarded
// when this view goes away.
function ShowCode() {
    const [code, setCode] = useState(null);
    const [now, setNow] = useState(Date.now());
    const [message, setMessage] = useState('');

    async function start() {
        try {
            setCode(await StartPairing());
            setMessage('');
            setNow(Date.now());
        } catch (error) {
            setMessage(`Could not start pairing: ${errorMessage(error)}`);
        }
    }

    useEffect(() => {
        start();
        return () => CancelPairing();
    }, []);

    useEffect(() => {
        if (!code) {
            return undefined;
        }
        const timer = setInterval(() => setNow(Date.now()), 1000);
        return () => clearInterval(timer);
    }, [code]);

    const remaining = code ? secondsLeft(code.expiresAt, now) : 0;
    if (code && remaining > 0) {
        return html`
            <div class="code-block">
                <p class="code" aria-label="Pairing code">
                    ${[...code.value].map((digit, index) => html`<span key=${index} class="code-digit" aria-hidden="true">${digit}</span>`)}
                </p>
                <div class="code-timer"><div class="code-timer-fill" style=${{width: `${Math.min(100, remaining / 1.2)}%`}}></div></div>
                <p class="hint">Enter this code on the other device. Expires in ${remaining}s.</p>
            </div>`;
    }
    return html`
        <div class="modal-actions">
            <button class="action primary" type="button" data-autofocus onClick=${start}>New code</button>
        </div>
        <p class="hint">${message || (code ? 'The code expired.' : 'Getting a code…')}</p>`;
}

// Enter the code another device is showing. The keypad works with the D-pad
// and A; typing digits works too.
function EnterCode({deviceName, onDone}) {
    const [digits, setDigits] = useState('');
    const [message, setMessage] = useState('');
    const [busy, setBusy] = useState(false);

    async function submit(entered) {
        setBusy(true);
        setMessage('Pairing…');
        try {
            await PairWith(deviceName, entered);
            onDone();
        } catch (error) {
            setDigits('');
            setMessage(`Could not pair: ${errorMessage(error)}`);
        } finally {
            setBusy(false);
        }
    }

    function press(key) {
        if (busy) {
            return;
        }
        if (key === '⌫') {
            setDigits((current) => current.slice(0, -1));
        } else if (key === 'Pair') {
            if (digits.length === CODE_LENGTH) {
                submit(digits);
            }
        } else {
            setDigits((current) => appendDigit(current, key, CODE_LENGTH));
        }
    }

    useEffect(() => {
        const onKey = (event) => {
            if (/^\d$/.test(event.key)) {
                press(event.key);
            } else if (event.key === 'Backspace') {
                press('⌫');
            }
        };
        document.addEventListener('keydown', onKey);
        return () => document.removeEventListener('keydown', onKey);
    }, [digits, busy]);

    return html`
        <div class="pairing-entry">
            <p>Enter the code shown on <strong>${deviceName}</strong></p>
            <p class="code" aria-label="Entered code">${digits.padEnd(CODE_LENGTH, '•')}</p>
            <div class="keypad">
                ${KEYPAD.map((key) => html`
                    <button class="${key === 'Pair' ? 'action primary' : 'action'}" type="button" key=${key}
                            data-autofocus=${key === '5' ? '' : undefined}
                            disabled=${busy || (key === 'Pair' && digits.length < CODE_LENGTH)}
                            onClick=${() => press(key)}>${key}</button>`)}
            </div>
            <p class="hint">${message}</p>
        </div>`;
}

// Unpaired devices nearby, to pick the one whose code you will enter.
function ChooseDevice({devices, onChoose}) {
    return html`
        <p class="hint">Show a code on the other device, then pick it here.</p>
        <ul class="devices">
            ${devices.map((device) => html`
                <li key=${device.name}>
                    <button class="tile" type="button" onClick=${() => onChoose(device)}>
                        <span class="tile-name">${device.name}</span>
                        <span class="tile-address">${device.host}:${device.port}</span>
                    </button>
                </li>`)}
        </ul>
        ${devices.length === 0 && html`
            <p class="empty searching"><span class="radar" aria-hidden="true"></span>Looking for devices…</p>`}`;
}

// Walks through pairing: either show a code here, or pick a device and enter
// the code it shows. It closes itself once the pairing has worked.
export function PairDialog({devices, paired, onClose, notify}) {
    const [view, setView] = useState({name: 'menu'});
    const knownPeers = useRef(new Set(paired.map((peer) => peer.fingerprint)));
    const pairedFingerprints = new Set(paired.map((peer) => peer.fingerprint));

    // Unpaired devices only show up while pairing mode is on.
    useEffect(() => () => { SetPairingMode(false); }, []);

    useEffect(() => {
        const added = paired.find((peer) => !knownPeers.current.has(peer.fingerprint));
        if (added) {
            notify(`Paired with ${added.name}`);
            onClose();
        }
    }, [paired.length]);

    function startEntering() {
        SetPairingMode(true);
        setView({name: 'choose'});
    }

    const atMenu = view.name === 'menu';
    return html`
        <${Modal} title="Pair a device" focusKey=${view.name} onClose=${atMenu ? onClose : () => setView({name: 'menu'})}
                  closeLabel=${atMenu ? 'Close' : 'Back'}>
            ${atMenu && html`
                <p class="hint">Pairing lets two devices send files to each other. You need to do it once.</p>
                <div class="choices">
                    <button class="action primary" type="button" data-autofocus onClick=${() => setView({name: 'show'})}>Show a code</button>
                    <button class="action primary" type="button" onClick=${startEntering}>Enter a code</button>
                </div>`}
            ${view.name === 'show' && html`<${ShowCode} />`}
            ${view.name === 'choose' && html`
                <${ChooseDevice} devices=${devices.filter((device) => !pairedFingerprints.has(device.fingerprint))}
                                 onChoose=${(device) => setView({name: 'enter', device: device.name})} />`}
            ${view.name === 'enter' && html`<${EnterCode} deviceName=${view.device} onDone=${onClose} />`}
        <//>`;
}
