import {useEffect, useState} from 'preact/hooks';
import {
    CancelPairing,
    PairWith,
    StartPairing,
    Unpair,
} from '../../bindings/github.com/TobyPinfold/deck-drop/service.js';
import {appendDigit, errorMessage, secondsLeft} from '../format.js';
import {html} from '../html.js';

const CODE_LENGTH = 6;
const KEYPAD = ['1', '2', '3', '4', '5', '6', '7', '8', '9', '⌫', '0', 'Pair'];

// Shows a one-time code for another device to enter.
function ShowCode({paired}) {
    const [code, setCode] = useState(null);
    const [now, setNow] = useState(Date.now());
    const [message, setMessage] = useState('');

    // Pairing finished, so the code has been used up.
    useEffect(() => setCode(null), [paired.length]);

    useEffect(() => {
        if (!code) {
            return undefined;
        }
        const timer = setInterval(() => setNow(Date.now()), 1000);
        return () => clearInterval(timer);
    }, [code]);

    async function start() {
        try {
            setCode(await StartPairing());
            setMessage('');
            setNow(Date.now());
        } catch (error) {
            setMessage(`Could not start pairing: ${errorMessage(error)}`);
        }
    }

    function cancel() {
        CancelPairing();
        setCode(null);
    }

    const remaining = code ? secondsLeft(code.expiresAt, now) : 0;
    if (code && remaining > 0) {
        return html`
            <div class="panel">
                <div>
                    <p class="code" aria-label="Pairing code">${code.value}</p>
                    <p class="hint">Enter this code on the other device. Expires in ${remaining}s.</p>
                </div>
                <button class="action" type="button" onClick=${cancel}>Cancel</button>
            </div>`;
    }
    return html`
        <div class="panel">
            <button class="action" type="button" onClick=${start}>Show pairing code</button>
            <p class="hint">${message || (code ? 'The code expired.' : 'Let another device pair with this one.')}</p>
        </div>`;
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
        <div class="panel pairing-entry">
            <p>Enter the code shown on <strong>${deviceName}</strong></p>
            <p class="code" aria-label="Entered code">${digits.padEnd(CODE_LENGTH, '•')}</p>
            <div class="keypad">
                ${KEYPAD.map((key) => html`
                    <button class="action" type="button" key=${key}
                            disabled=${busy || (key === 'Pair' && digits.length < CODE_LENGTH)}
                            onClick=${() => press(key)}>${key}</button>`)}
            </div>
            <p class="hint">${message}</p>
            <button class="action" type="button" onClick=${onDone}>Cancel</button>
        </div>`;
}

export function Pairing({paired, selectedDevice, selectedIsPaired}) {
    const [entering, setEntering] = useState(false);
    const [message, setMessage] = useState('');

    // Pairing with a device that goes away or changes is abandoned.
    useEffect(() => setEntering(false), [selectedDevice?.name, selectedIsPaired]);

    async function unpair(peer) {
        try {
            await Unpair(peer.fingerprint);
            setMessage('');
        } catch (error) {
            setMessage(`Could not unpair: ${errorMessage(error)}`);
        }
    }

    const canPair = selectedDevice && !selectedIsPaired;
    return html`
        <h2>Pairing</h2>
        ${entering && canPair
            ? html`<${EnterCode} deviceName=${selectedDevice.name} onDone=${() => setEntering(false)} />`
            : html`
                <div class="panel">
                    <button class="action" type="button" disabled=${!canPair} onClick=${() => setEntering(true)}>
                        ${canPair ? `Pair with ${selectedDevice.name}` : 'Pair with selected device'}
                    </button>
                    <p class="hint">${canPair
                        ? 'Enter the code it shows.'
                        : 'Select an unpaired device to enter its code.'}</p>
                </div>`}
        <${ShowCode} paired=${paired} />
        <h3>Paired devices</h3>
        <ul class="paired">
            ${paired.map((peer) => html`
                <li key=${peer.fingerprint}>
                    <span>${peer.name}</span>
                    <button class="action small" type="button" onClick=${() => unpair(peer)}>Unpair</button>
                </li>`)}
        </ul>
        ${paired.length === 0 && html`<p class="empty">No paired devices yet. Only paired devices can send files here.</p>`}
        ${message && html`<p class="hint">${message}</p>`}`;
}
