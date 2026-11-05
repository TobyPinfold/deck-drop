// Connects keyboard and gamepad input to focus navigation.
import {createPressFilter, createRepeater, readInput} from './gamepad.js';
import {getNextFocus} from '@bbc/tv-lrud-spatial';

const HOLD_DELAY_MS = 500;
const HOLD_INTERVAL_MS = 160;
// Ignores a second move straight after one, in case a button press is
// reported by both the gamepad and the keyboard.
const MIN_MOVE_GAP_MS = 100;
// How long after one source reports A or B to ignore the other reporting it.
const DUPLICATE_PRESS_MS = 250;
// Pixels per second at a full tilt of the right stick.
const SCROLL_SPEED = 500;

const DIRECTION_KEYS = {up: 'ArrowUp', down: 'ArrowDown', left: 'ArrowLeft', right: 'ArrowRight'};
const KEY_DIRECTIONS = {ArrowUp: 'up', ArrowDown: 'down', ArrowLeft: 'left', ArrowRight: 'right'};
const CONFIRM_KEYS = new Set(['Enter', ' ']);

function scrollableAncestor(element) {
    for (let parent = element?.parentElement; parent; parent = parent.parentElement) {
        if (parent.scrollHeight > parent.clientHeight && getComputedStyle(parent).overflowY !== 'visible') {
            return parent;
        }
    }
    return document.scrollingElement;
}

// confirm presses the focused button. Text fields are left alone, so A does not
// submit or clear what is being typed.
function confirm() {
    const focused = document.activeElement;
    if (focused && focused !== document.body && focused.tagName !== 'INPUT') {
        focused.click();
    }
}

function focus(element) {
    element.focus();
    element.scrollIntoView({block: 'nearest', inline: 'nearest'});
}

// The library picks the best neighbour, working row by row.
let lastMoveAt = -Infinity;

function move(direction) {
    const now = performance.now();
    if (now - lastMoveAt < MIN_MOVE_GAP_MS) {
        return;
    }
    lastMoveAt = now;
    const current = document.activeElement;
    const dialogs = document.querySelectorAll('[aria-modal="true"]');
    const scope = dialogs[dialogs.length - 1];
    if (!current || current === document.body) {
        (scope ?? document).querySelector('button:not([disabled]), input:not([disabled])')?.focus();
        return;
    }
    // While a dialog is open, focus stays inside the topmost one.
    const next = getNextFocus(current, DIRECTION_KEYS[direction], scope);
    if (next) {
        focus(next);
    }
}

// startNavigation lets the D-pad, left stick and arrow keys move focus
// between buttons. A presses the focused button, B calls onBack, and the right
// stick scrolls.
export function startNavigation({onBack}) {
    const repeaters = {
        direction: createRepeater({delay: HOLD_DELAY_MS, interval: HOLD_INTERVAL_MS}),
        confirm: createRepeater({delay: Infinity, interval: Infinity}),
        back: createRepeater({delay: Infinity, interval: Infinity}),
    };
    const presses = createPressFilter(DUPLICATE_PRESS_MS);
    let heldDirection = null;
    let lastFrame = null;

    function poll(now) {
        const pad = [...(navigator.getGamepads?.() ?? [])].find((candidate) => candidate?.connected);
        const input = pad ? readInput(pad) : {direction: null, confirm: false, back: false, scroll: 0};

        // Repeating a held direction must restart when the direction changes.
        if (input.direction !== heldDirection) {
            repeaters.direction.fire(false, now);
            heldDirection = input.direction;
        }
        if (repeaters.direction.fire(input.direction !== null, now)) {
            move(input.direction);
        }
        if (repeaters.confirm.fire(input.confirm, now) && presses.accept('confirm', 'gamepad', now)) {
            confirm();
        }
        if (repeaters.back.fire(input.back, now) && presses.accept('back', 'gamepad', now)) {
            onBack();
        }
        if (input.scroll !== 0 && lastFrame !== null) {
            const seconds = Math.min(now - lastFrame, 50) / 1000;
            scrollableAncestor(document.activeElement)?.scrollBy(0, input.scroll * SCROLL_SPEED * seconds);
        }
        lastFrame = now;
        requestAnimationFrame(poll);
    }
    requestAnimationFrame(poll);

    document.addEventListener('keydown', (event) => {
        const direction = KEY_DIRECTIONS[event.key];
        // Left and right move the caret inside a text field.
        const editing = document.activeElement?.tagName === 'INPUT';
        if (direction && !(editing && (direction === 'left' || direction === 'right'))) {
            event.preventDefault();
            move(direction);
        } else if (CONFIRM_KEYS.has(event.key) && !editing) {
            // Buttons are pressed here rather than by the browser, so a key
            // that duplicates A can be dropped. A held key presses once.
            event.preventDefault();
            if (!event.repeat && presses.accept('confirm', 'keyboard', performance.now())) {
                confirm();
            }
        } else if (event.key === 'Escape') {
            if (!event.repeat && presses.accept('back', 'keyboard', performance.now())) {
                onBack();
            }
        }
    });
    // Space presses a button when released, so stop that too.
    document.addEventListener('keyup', (event) => {
        if (event.key === ' ' && document.activeElement?.tagName !== 'INPUT') {
            event.preventDefault();
        }
    });
}
