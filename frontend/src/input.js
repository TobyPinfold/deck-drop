// Connects keyboard and gamepad input to focus navigation.
import {createRepeater, readInput} from './gamepad.js';
import {findNext} from './navigation.js';

const HOLD_DELAY_MS = 400;
const HOLD_INTERVAL_MS = 120;
const SCROLL_SPEED = 24;

const KEY_DIRECTIONS = {ArrowUp: 'up', ArrowDown: 'down', ArrowLeft: 'left', ArrowRight: 'right'};

function focusableElements() {
    return [...document.querySelectorAll('button:not([disabled])')].filter((element) => element.offsetParent !== null);
}

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

function move(direction) {
    const elements = focusableElements();
    const current = elements.indexOf(document.activeElement);
    if (current < 0) {
        elements[0]?.focus();
        return;
    }

    const rects = elements.map((element) => element.getBoundingClientRect());
    const next = findNext(rects[current], rects, direction);
    if (next >= 0) {
        focus(elements[next]);
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
            document.activeElement?.click();
        }
        if (repeaters.back.fire(input.back, now)) {
            onBack();
        }
        if (input.scroll !== 0) {
            scrollableAncestor(document.activeElement)?.scrollBy(0, input.scroll * SCROLL_SPEED);
        }
        requestAnimationFrame(poll);
    }
    requestAnimationFrame(poll);

    document.addEventListener('keydown', (event) => {
        const direction = KEY_DIRECTIONS[event.key];
        if (direction) {
            event.preventDefault();
            move(direction);
        } else if (event.key === 'Escape') {
            onBack();
        }
    });
}
