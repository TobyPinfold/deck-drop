// Turns a gamepad's state into navigation input. This uses the "standard"
// gamepad layout, which Steam Deck controls and most controllers report.

const BUTTON_A = 0;
const BUTTON_B = 1;
const DPAD = {up: 12, down: 13, left: 14, right: 15};
const STICK_X = 0;
const STICK_Y = 1;
const RIGHT_STICK_Y = 3;

export const DEADZONE = 0.5;
const SCROLL_DEADZONE = 0.2;

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

// readInput returns the navigation input a pad is currently producing.
export function readInput(pad) {
    const dpad = Object.keys(DPAD).find((direction) => isPressed(pad, DPAD[direction])) ?? null;
    const scroll = pad.axes[RIGHT_STICK_Y] ?? 0;

    return {
        direction: dpad ?? stickDirection(pad.axes[STICK_X] ?? 0, pad.axes[STICK_Y] ?? 0),
        confirm: isPressed(pad, BUTTON_A),
        back: isPressed(pad, BUTTON_B),
        scroll: Math.abs(scroll) >= SCROLL_DEADZONE ? scroll : 0,
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
