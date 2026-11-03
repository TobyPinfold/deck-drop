import assert from 'node:assert/strict';
import {test} from 'node:test';
import {findNext} from './navigation.js';

function rect(left, top, width = 100, height = 50) {
    return {left, top, right: left + width, bottom: top + height};
}

// A 2x2 grid:
//   0 1
//   2 3
const grid = [rect(0, 0), rect(150, 0), rect(0, 100), rect(150, 100)];

test('moves to the neighbour in each direction', () => {
    assert.equal(findNext(grid[0], grid, 'right'), 1);
    assert.equal(findNext(grid[1], grid, 'left'), 0);
    assert.equal(findNext(grid[0], grid, 'down'), 2);
    assert.equal(findNext(grid[2], grid, 'up'), 0);
});

test('returns -1 at the edge', () => {
    assert.equal(findNext(grid[0], grid, 'left'), -1);
    assert.equal(findNext(grid[0], grid, 'up'), -1);
    assert.equal(findNext(grid[3], grid, 'right'), -1);
    assert.equal(findNext(grid[3], grid, 'down'), -1);
});

test('prefers the aligned element over a nearer diagonal one', () => {
    const current = rect(0, 100);
    const diagonal = rect(120, 40);
    const aligned = rect(300, 100);
    assert.equal(findNext(current, [diagonal, aligned], 'right'), 1);
});

test('ignores the current element and elements behind', () => {
    const candidates = [grid[0], rect(-200, 0)];
    assert.equal(findNext(grid[0], candidates, 'right'), -1);
});

test('goes down into a wide element below from a narrow one', () => {
    const narrow = rect(0, 0, 80, 40);
    const wide = rect(0, 100, 600, 40);
    assert.equal(findNext(narrow, [wide], 'down'), 0);
});
