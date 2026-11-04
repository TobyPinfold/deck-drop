import assert from 'node:assert/strict';
import {test} from 'node:test';
import {createRepeater, readInput} from './gamepad.js';

function pad({pressed = [], axes = [0, 0, 0, 0]} = {}) {
    const buttons = Array.from({length: 17}, (_, index) => ({pressed: pressed.includes(index)}));
    return {buttons, axes};
}

const A = 0;
const B = 1;
const DPAD_UP = 12;
const DPAD_DOWN = 13;
const DPAD_LEFT = 14;
const DPAD_RIGHT = 15;

test('reads the D-pad as a direction', () => {
    assert.equal(readInput(pad({pressed: [DPAD_UP]})).direction, 'up');
    assert.equal(readInput(pad({pressed: [DPAD_DOWN]})).direction, 'down');
    assert.equal(readInput(pad({pressed: [DPAD_LEFT]})).direction, 'left');
    assert.equal(readInput(pad({pressed: [DPAD_RIGHT]})).direction, 'right');
});

test('reads the left stick as a direction beyond the dead zone', () => {
    assert.equal(readInput(pad({axes: [0.9, 0.1, 0, 0]})).direction, 'right');
    assert.equal(readInput(pad({axes: [-0.2, -0.9, 0, 0]})).direction, 'up');
    assert.equal(readInput(pad({axes: [0.3, 0.3, 0, 0]})).direction, null);
});

test('uses the dominant stick axis on diagonals', () => {
    assert.equal(readInput(pad({axes: [0.6, 0.9, 0, 0]})).direction, 'down');
});

test('reads A as confirm and B as back', () => {
    const input = readInput(pad({pressed: [A]}));
    assert.equal(input.confirm, true);
    assert.equal(input.back, false);
    assert.equal(readInput(pad({pressed: [B]})).back, true);
});

test('reads the right stick as vertical scrolling', () => {
    assert.equal(readInput(pad({axes: [0, 0, 0, 1]})).scroll, 1);
    assert.equal(readInput(pad({axes: [0, 0, 0, -1]})).scroll, -1);
    assert.equal(readInput(pad({axes: [0, 0, 0, 0.2]})).scroll, 0);
    assert.ok(readInput(pad({axes: [0, 0, 0, 0.5]})).scroll < 0.2);
});

test('an idle pad produces no input', () => {
    assert.deepEqual(readInput(pad()), {direction: null, confirm: false, back: false, scroll: 0});
});

test('repeater fires at once, waits, then repeats while held', () => {
    const repeater = createRepeater({delay: 400, interval: 100});
    assert.equal(repeater.fire(true, 0), true);
    assert.equal(repeater.fire(true, 200), false);
    assert.equal(repeater.fire(true, 400), true);
    assert.equal(repeater.fire(true, 450), false);
    assert.equal(repeater.fire(true, 500), true);
});

test('repeater fires again after release', () => {
    const repeater = createRepeater({delay: 400, interval: 100});
    repeater.fire(true, 0);
    repeater.fire(false, 50);
    assert.equal(repeater.fire(true, 60), true);
});

test('a repeater without repeats fires once per press', () => {
    const repeater = createRepeater({delay: Infinity, interval: Infinity});
    assert.equal(repeater.fire(true, 0), true);
    assert.equal(repeater.fire(true, 100000), false);
});
