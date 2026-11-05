// Turns a gamepad's state into navigation input. This uses the "standard"
// gamepad layout, which Steam Deck controls and most controllers report.

const BUTTON_A = 0;
const BUTTON_B = 1;
const DPAD = {up: 12, down: 13, left: 14, right: 15};
const STICK_X = 0;
const STICK_Y = 1;
const RIGHT_STICK_Y = 3;

export const DEADZONE = 0.5;
const SCROLL_DEADZONE = 0.3;

const isPressed = (pad, index) => Boolean(pad.buttons[index]?.pressed);

function stickDirection(x, y) {
    if (Math.max(Math.abs(x), Math.abs(y)) < DEADZONE) {
        return null;
    }
    if (Math.abs(x) > Math.abs(y)) {
        return x > 0 ? 'right' : 'left';
    }
    return y > 0 ? 'down' : 'up';
}

// scrollSpeed eases the right stick in: a light tilt scrolls slowly and a full
// tilt scrolls at full speed, and a drifting stick does nothing. The result is
// between -1 and 1.
function scrollSpeed(tilt) {
    const amount = (Math.abs(tilt) - SCROLL_DEADZONE) / (1 - SCROLL_DEADZONE);
    return amount > 0 ? Math.sign(tilt) * amount ** 2 : 0;
}

// readInput returns the navigation input a pad is currently producing.
export function readInput(pad) {
    const dpad = Object.keys(DPAD).find((direction) => isPressed(pad, DPAD[direction])) ?? null;
    const tilt = pad.axes[RIGHT_STICK_Y] ?? 0;

    return {
        direction: dpad ?? stickDirection(pad.axes[STICK_X] ?? 0, pad.axes[STICK_Y] ?? 0),
        confirm: isPressed(pad, BUTTON_A),
        back: isPressed(pad, BUTTON_B),
        scroll: scrollSpeed(tilt),
    };
}

// createRepeater turns a held input into discrete events: one at once, then
// after delay, then every interval. Use Infinity for both to fire once per press.
export function createRepeater({delay, interval}) {
    let nextAt = null;

    return {
        fire(active, now) {
            if (!active) {
                nextAt = null;
                return false;
            }
            if (nextAt === null) {
                nextAt = now + delay;
                return true;
            }
            if (now >= nextAt) {
                nextAt = now + interval;
                return true;
            }
            return false;
        },
    };
}

// createPressFilter drops a press that one source reports straight after
// another source reported the same press. In Game Mode, Steam reports a button
// both as a gamepad button and as a key, which would otherwise act twice: a
// toggle would flip back, or a dialog would skip a step. Presses from the same
// source always count, so quick taps still work.
export function createPressFilter(gap) {
    const last = new Map();

    return {
        accept(action, source, now) {
            const previous = last.get(action);
            if (previous && previous.source !== source && now - previous.at < gap) {
                return false;
            }
            last.set(action, {source, at: now});
            return true;
        },
    };
}
