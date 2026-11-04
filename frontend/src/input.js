// Connects keyboard and gamepad input to focus navigation.
import {createRepeater, readInput} from './gamepad.js';
import {getNextFocus} from '@bbc/tv-lrud-spatial';

const HOLD_DELAY_MS = 500;
const HOLD_INTERVAL_MS = 160;
// Ignores a second move straight after one, in case a button press is
// reported by both the gamepad and the keyboard.
const MIN_MOVE_GAP_MS = 100;
// Pixels per second at a full tilt of the right stick.
const SCROLL_SPEED = 500;

const DIRECTION_KEYS = {up: 'ArrowUp', down: 'ArrowDown', left: 'ArrowLeft', right: 'ArrowRight'};
const KEY_DIRECTIONS = {ArrowUp: 'up', ArrowDown: 'down', ArrowLeft: 'left', ArrowRight: 'right'};

function scrollableAncestor(element) {
    for (let parent = element?.parentElement; parent; parent = parent.parentElement) {
        if (parent.scrollHeight > parent.clientHeight && getComputedStyle(parent).overflowY !== 'visible') {
            return parent;
        }
    }
    return document.scrollingElement;
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
    if (!current || current === document.body) {
        document.querySelector('button:not([disabled]), input:not([disabled])')?.focus();
        return;
    }
    const next = getNextFocus(current, DIRECTION_KEYS[direction]);
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
        if (repeaters.confirm.fire(input.confirm, now)) {
            if (document.activeElement?.tagName !== 'INPUT') {
                document.activeElement?.click();
            }
        }
        if (repeaters.back.fire(input.back, now)) {
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
        } else if (event.key === 'Escape') {
            onBack();
        }
    });
}
